// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package core

import (
	"fmt"
	"sync"
	"time"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/flow"
	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/nodes"
)

// What the trigger does after sending the first message.
const (
	triggerWait   = "wait"   // wait, then send the second message
	triggerReset  = "reset"  // wait until reset, send nothing else
	triggerResend = "resend" // resend the first message every period until reset
)

// Value types for the two messages, with Node-RED's names. str, num, bool,
// json and date set msg.payload; these three are special.
const (
	triggerPay    = "pay"  // the message that started the trigger
	triggerLatest = "payl" // the latest message received (second message only)
	triggerNul    = "nul"  // nothing
)

// TriggerNode is Node-RED's "trigger": when a message arrives it sends a
// first message, then waits and sends a second one (e.g. "on", 60 s,
// "off"). While it waits, new messages are ignored, or restart the wait
// when "extend" is on. msg.reset (or a payload equal to the reset value)
// stops it without sending the second message.
type TriggerNode struct {
	nodes.BaseNode
	config    flow.NodeConfig
	op1, op2  any
	op1Type   string
	op2Type   string
	then      string
	duration  time.Duration
	extend    bool
	override  bool // msg.delay (ms) overrides the wait
	second    bool // send the second message on output 2
	resetVal  string
	byTopic   bool
	topicProp string

	mu      sync.Mutex
	active  map[string]*triggerRun
	stopped bool
}

// triggerRun is one running trigger (one per topic when byTopic).
type triggerRun struct {
	first  *flow.Message
	latest *flow.Message
	timer  *time.Timer
	ticker *time.Ticker
	stop   chan struct{}
}

// NewTriggerNode is the NodeFactory for the trigger node type.
func NewTriggerNode(config flow.NodeConfig) (flow.NodeInstance, error) {
	return &TriggerNode{config: config, active: map[string]*triggerRun{}}, nil
}

// Init parses the configuration.
func (n *TriggerNode) Init() error {
	p := n.config.Properties
	var err error
	n.op1Type = nodes.StringVal(p, "op1type", "str")
	if n.op1Type == triggerLatest {
		n.op1Type = triggerPay
	}
	if n.op1, err = triggerValue(n.op1Type, nodes.StringVal(p, "op1", "1")); err != nil {
		return fmt.Errorf("trigger: first message: %w", err)
	}
	n.op2Type = nodes.StringVal(p, "op2type", "str")
	if n.op2, err = triggerValue(n.op2Type, nodes.StringVal(p, "op2", "0")); err != nil {
		return fmt.Errorf("trigger: second message: %w", err)
	}
	n.then = nodes.StringVal(p, "then", triggerWait)
	switch n.then {
	case triggerWait, triggerReset, triggerResend:
	default:
		return fmt.Errorf("trigger: unknown option %q", n.then)
	}
	n.duration = unitDuration(nodes.FloatVal(p, "duration", 250), nodes.StringVal(p, "units", "ms"))
	if n.then != triggerReset && n.duration <= 0 {
		return fmt.Errorf("trigger: the time must be more than 0")
	}
	n.extend = nodes.BoolVal(p, "extend", false)
	n.override = nodes.BoolVal(p, "overrideDelay", false)
	n.second = nodes.BoolVal(p, "second", false)
	n.resetVal = nodes.StringVal(p, "reset", "")
	n.byTopic = nodes.StringVal(p, "bytopic", "all") == "topic"
	n.topicProp = nodes.StringVal(p, "topic", "topic")
	return nil
}

// triggerValue resolves a fixed value; the special types resolve per message.
func triggerValue(typ, value string) (any, error) {
	switch typ {
	case triggerPay, triggerLatest, triggerNul:
		return nil, nil
	case "str", "num", "bool", "json", "date":
		return nodes.ResolveValue(typ, value, "", nil, nodes.ValueContext{})
	}
	return nil, fmt.Errorf("unknown value type %q", typ)
}

// Start has nothing to start: runs begin with messages.
func (n *TriggerNode) Start() error {
	n.mu.Lock()
	n.stopped = false
	n.mu.Unlock()
	return nil
}

// Stop cancels every run without sending the second message.
func (n *TriggerNode) Stop() error {
	n.mu.Lock()
	n.stopped = true
	for k, r := range n.active {
		r.cancel()
		delete(n.active, k)
	}
	n.mu.Unlock()
	return nil
}

func (r *triggerRun) cancel() {
	if r.timer != nil {
		r.timer.Stop()
	}
	if r.ticker != nil {
		r.ticker.Stop()
	}
	close(r.stop)
}

// HandleMessage starts, extends or resets a run.
func (n *TriggerNode) HandleMessage(msg *flow.Message) ([][]*flow.Message, error) {
	key := ""
	if n.byTopic {
		key = topicKey(msg.Get(n.topicProp))
	}

	n.mu.Lock()
	defer n.mu.Unlock()
	if n.stopped {
		return nil, nil
	}

	if n.isReset(msg) {
		if n.byTopic && key != "" {
			if r := n.active[key]; r != nil {
				r.cancel()
				delete(n.active, key)
			}
		} else {
			for k, r := range n.active {
				r.cancel()
				delete(n.active, k)
			}
		}
		n.updateStatus()
		return nil, nil
	}

	if r := n.active[key]; r != nil {
		r.latest = msg
		if n.then == triggerWait && n.extend {
			r.timer.Reset(n.waitFor(msg))
		}
		return nil, nil
	}

	r := &triggerRun{first: msg, latest: msg, stop: make(chan struct{})}
	n.active[key] = r
	switch n.then {
	case triggerWait:
		r.timer = time.AfterFunc(n.waitFor(msg), func() { n.finish(key, r) })
	case triggerResend:
		r.ticker = time.NewTicker(n.waitFor(msg))
		go n.resend(key, r)
	}
	n.updateStatus()

	out := n.build(n.op1Type, n.op1, msg, msg)
	if out == nil {
		return nil, nil
	}
	return [][]*flow.Message{{out}}, nil
}

func (n *TriggerNode) isReset(msg *flow.Message) bool {
	if msg.Get("reset") != nil {
		return true
	}
	if n.resetVal == "" {
		return false
	}
	p := msg.Payload()
	return p != nil && fmt.Sprint(p) == n.resetVal
}

// waitFor returns the wait for this run: msg.delay (ms) when allowed.
func (n *TriggerNode) waitFor(msg *flow.Message) time.Duration {
	if n.override {
		if ms, ok := toNumber(msg.Get("delay")); ok && ms > 0 {
			return time.Duration(ms * float64(time.Millisecond))
		}
	}
	return n.duration
}

// finish sends the second message when the wait ends.
func (n *TriggerNode) finish(key string, r *triggerRun) {
	n.mu.Lock()
	if n.stopped || n.active[key] != r {
		n.mu.Unlock()
		return // reset or replaced meanwhile
	}
	delete(n.active, key)
	n.updateStatus()
	out := n.build(n.op2Type, n.op2, r.first, r.latest)
	n.mu.Unlock()
	if out == nil || n.Send == nil {
		return
	}
	port := 0
	if n.second {
		port = 1
	}
	n.Send(port, out)
}

// resend repeats the first message every period until the run is reset.
func (n *TriggerNode) resend(key string, r *triggerRun) {
	for {
		select {
		case <-r.stop:
			return
		case <-r.ticker.C:
			n.mu.Lock()
			if n.stopped || n.active[key] != r {
				n.mu.Unlock()
				return
			}
			out := n.build(n.op1Type, n.op1, r.latest, r.latest)
			n.mu.Unlock()
			if out != nil && n.Send != nil {
				n.Send(0, out)
			}
		}
	}
}

// build makes an outgoing message: a copy of the original with the payload
// set, the original or latest message itself, or nil for "nothing".
func (n *TriggerNode) build(typ string, value any, first, latest *flow.Message) *flow.Message {
	switch typ {
	case triggerNul:
		return nil
	case triggerPay:
		return first.Clone()
	case triggerLatest:
		return latest.Clone()
	case "date":
		value, _ = nodes.ResolveValue("date", "", "", nil, nodes.ValueContext{})
	}
	out := first.Clone()
	out.SetPayload(value)
	return out
}

// updateStatus shows a blue dot while a run is active. Caller holds mu.
func (n *TriggerNode) updateStatus() {
	if n.Status == nil {
		return
	}
	switch len(n.active) {
	case 0:
		n.Status("", "")
	case 1:
		n.Status("blue", "waiting")
	default:
		n.Status("blue", fmt.Sprintf("waiting: %d topics", len(n.active)))
	}
}

// TriggerTypeInfo describes the trigger node.
func TriggerTypeInfo() flow.NodeTypeInfo {
	return flow.NodeTypeInfo{
		Type:        "trigger",
		Category:    "function",
		Label:       "Trigger",
		Description: "Sends a message, waits, then sends a second one",
		Icon:        "mdi-timer-outline",
		Keywords:    []string{"timer", "timeout", "watchdog", "pulse", "debounce"},
		Defaults: map[string]any{
			"op1":           "1",
			"op1type":       "str",
			"op2":           "0",
			"op2type":       "str",
			"then":          triggerWait,
			"duration":      250.0,
			"units":         "ms",
			"extend":        false,
			"overrideDelay": false,
			"second":        false,
			"reset":         "",
			"bytopic":       "all",
			"topic":         "topic",
		},
		Inputs:  1,
		Outputs: 1,
	}
}
