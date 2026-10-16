// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package core

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/flow"
	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/nodes"
)

// Join modes and outputs, with Node-RED's names.
const (
	joinAuto   = "auto"   // undo a split, using msg.parts
	joinCustom = "custom" // build a value from several messages

	joinString = "string"
	joinBuffer = "buffer"
	joinArray  = "array"
	joinObject = "object" // { msg.<key>: value }
	joinMerged = "merged" // merge the values (objects) into one object
)

// JoinNode is Node-RED's "join". In automatic mode it puts back together a
// sequence made by split or batch, using msg.parts. In manual mode it
// builds a text, bytes, list or object from several messages and sends it
// after a number of messages, after a time from the first one, or when a
// message has msg.complete. msg.reset discards what has been collected.
type JoinNode struct {
	nodes.BaseNode
	config   flow.NodeConfig
	mode     string
	build    string
	property string // "" = the whole message
	key      string // object: the message property used as the key
	joiner   string
	count    int
	timeout  time.Duration

	mu      sync.Mutex
	groups  map[string]*joinGroup // auto: one per msg.parts.id
	manual  *joinGroup
	stopped bool
}

// joinGroup is a sequence being collected.
type joinGroup struct {
	last   *flow.Message // the latest message; the output is built on it
	items  map[int]any   // auto: index → value
	parts  map[string]any
	values []any    // manual: values in arrival order
	keys   []string // manual object: key of each value
	timer  *time.Timer
}

// NewJoinNode is the NodeFactory for the join node type.
func NewJoinNode(config flow.NodeConfig) (flow.NodeInstance, error) {
	return &JoinNode{config: config, groups: map[string]*joinGroup{}}, nil
}

// Init parses the configuration.
func (n *JoinNode) Init() error {
	p := n.config.Properties
	n.mode = nodes.StringVal(p, "mode", joinAuto)
	if n.mode != joinAuto && n.mode != joinCustom {
		return fmt.Errorf("join: unknown mode %q", n.mode)
	}
	n.build = nodes.StringVal(p, "build", joinArray)
	switch n.build {
	case joinString, joinBuffer, joinArray, joinObject, joinMerged:
	default:
		return fmt.Errorf("join: unknown output %q", n.build)
	}
	n.property = nodes.StringVal(p, "property", "payload")
	if nodes.StringVal(p, "propertyType", "msg") == "full" {
		n.property = ""
	}
	n.key = nodes.StringVal(p, "key", "topic")
	n.joiner = unescapeSeparator(nodes.StringVal(p, "joiner", `\n`))
	n.count = nodes.IntVal(p, "count", 0)
	if t := nodes.FloatVal(p, "timeout", 0); t > 0 {
		n.timeout = unitDuration(t, "s")
	}
	return nil
}

// Start has nothing to start.
func (n *JoinNode) Start() error {
	n.mu.Lock()
	n.stopped = false
	n.mu.Unlock()
	return nil
}

// Stop discards everything collected.
func (n *JoinNode) Stop() error {
	n.mu.Lock()
	n.stopped = true
	n.resetAll()
	n.mu.Unlock()
	return nil
}

// resetAll discards every group. Caller holds mu.
func (n *JoinNode) resetAll() {
	for id, g := range n.groups {
		g.stopTimer()
		delete(n.groups, id)
	}
	if n.manual != nil {
		n.manual.stopTimer()
		n.manual = nil
	}
	n.updateStatus()
}

func (g *joinGroup) stopTimer() {
	if g.timer != nil {
		g.timer.Stop()
	}
}

// HandleMessage collects msg and sends the joined message when complete.
func (n *JoinNode) HandleMessage(msg *flow.Message) ([][]*flow.Message, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.stopped {
		return nil, nil
	}
	if msg.Get("reset") != nil {
		n.resetAll()
		return nil, nil
	}
	var out *flow.Message
	if n.mode == joinAuto {
		out = n.handleAuto(msg)
	} else {
		out = n.handleManual(msg)
	}
	n.updateStatus()
	if out == nil {
		return nil, nil
	}
	return [][]*flow.Message{{out}}, nil
}

// ---- automatic --------------------------------------------------------------

func (n *JoinNode) handleAuto(msg *flow.Message) *flow.Message {
	parts, _ := msg.Get("parts").(map[string]any)
	id, _ := parts["id"].(string)
	idx, okIdx := toNumber(parts["index"])
	if parts == nil || id == "" || !okIdx {
		if n.Status != nil {
			n.Status("yellow", "msg.parts missing: use Split or Batch before, or the manual mode")
		}
		return nil
	}
	g := n.groups[id]
	if g == nil {
		g = &joinGroup{items: map[int]any{}, parts: parts}
		n.groups[id] = g
	}
	g.last = msg
	if parts["type"] == "object" {
		k, _ := parts["key"].(string)
		g.items[int(idx)] = keyed{key: k, value: n.valueOf(msg, parts)}
	} else {
		g.items[int(idx)] = n.valueOf(msg, parts)
	}

	count, hasCount := toNumber(parts["count"])
	complete := msg.Get("complete") != nil
	if !complete && (!hasCount || len(g.items) < int(count)) {
		return nil
	}
	delete(n.groups, id)
	return n.assembleAuto(g)
}

// valueOf reads the value a sequence message carries.
func (n *JoinNode) valueOf(msg *flow.Message, parts map[string]any) any {
	prop := n.property
	if n.mode == joinAuto {
		prop = "payload"
		if p, ok := parts["property"].(string); ok && p != "" {
			prop = p
		}
	}
	if prop == "" {
		return msg.Data()
	}
	return msg.Get(prop)
}

// assembleAuto rebuilds the value the split started from.
func (n *JoinNode) assembleAuto(g *joinGroup) *flow.Message {
	indexes := make([]int, 0, len(g.items))
	for i := range g.items {
		indexes = append(indexes, i)
	}
	sort.Ints(indexes)

	typ, _ := g.parts["type"].(string)
	var result any
	switch typ {
	case "string":
		ch, _ := g.parts["ch"].(string)
		strs := make([]string, 0, len(indexes))
		for _, i := range indexes {
			strs = append(strs, fmt.Sprint(nilToEmpty(g.items[i])))
		}
		result = strings.Join(strs, ch)
	case "buffer":
		ch, _ := g.parts["ch"].(string)
		chunks := make([][]byte, 0, len(indexes))
		for _, i := range indexes {
			b, _ := g.items[i].([]byte)
			chunks = append(chunks, b)
		}
		result = bytes.Join(chunks, []byte(ch))
	case "object":
		obj := map[string]any{}
		// The key is on each message; the group only kept the values.
		for _, i := range indexes {
			if kv, ok := g.items[i].(keyed); ok {
				obj[kv.key] = kv.value
			}
		}
		result = obj
	default: // "array", and batch sequences (no type)
		arr := make([]any, 0, len(indexes))
		chunked := false
		if l, ok := toNumber(g.parts["len"]); ok && l > 1 {
			chunked = true
		}
		for _, i := range indexes {
			if sub, ok := g.items[i].([]any); ok && chunked {
				arr = append(arr, sub...)
			} else {
				arr = append(arr, g.items[i])
			}
		}
		result = arr
	}

	out := g.last.Clone()
	prop := "payload"
	if p, ok := g.parts["property"].(string); ok && p != "" {
		prop = p
	}
	out.Set(prop, result)
	if outer, ok := g.parts["parts"].(map[string]any); ok {
		out.Set("parts", outer)
	} else {
		out.Delete("parts")
	}
	out.Delete("complete")
	return out
}

// keyed is an object item: the split key with its value.
type keyed struct {
	key   string
	value any
}

func nilToEmpty(v any) any {
	if v == nil {
		return ""
	}
	return v
}

// ---- manual -----------------------------------------------------------------

func (n *JoinNode) handleManual(msg *flow.Message) *flow.Message {
	complete := msg.Get("complete") != nil
	value := n.valueOf(msg, nil)
	if value == nil && complete {
		// A bare msg.complete only sends what is there.
		return n.flushManual()
	}
	g := n.manual
	if g == nil {
		g = &joinGroup{}
		n.manual = g
		if n.timeout > 0 {
			g.timer = time.AfterFunc(n.timeout, func() { n.timeoutManual(g) })
		}
	}
	g.last = msg
	g.values = append(g.values, value)
	if n.build == joinObject {
		g.keys = append(g.keys, topicKey(msg.Get(n.key)))
	}
	if complete || (n.count > 0 && n.items(g) >= n.count) {
		return n.flushManual()
	}
	return nil
}

// items counts the parts collected: distinct keys for an object, as in
// Node-RED; otherwise messages.
func (n *JoinNode) items(g *joinGroup) int {
	if n.build != joinObject {
		return len(g.values)
	}
	seen := map[string]bool{}
	for _, k := range g.keys {
		seen[k] = true
	}
	return len(seen)
}

func (n *JoinNode) timeoutManual(g *joinGroup) {
	n.mu.Lock()
	if n.stopped || n.manual != g {
		n.mu.Unlock()
		return
	}
	out := n.flushManual()
	n.updateStatus()
	n.mu.Unlock()
	if out != nil && n.Send != nil {
		n.Send(0, out)
	}
}

// flushManual builds and clears the manual group. Caller holds mu.
func (n *JoinNode) flushManual() *flow.Message {
	g := n.manual
	n.manual = nil
	if g == nil || len(g.values) == 0 {
		return nil
	}
	g.stopTimer()

	var result any
	switch n.build {
	case joinString:
		strs := make([]string, 0, len(g.values))
		for _, v := range g.values {
			strs = append(strs, fmt.Sprint(nilToEmpty(v)))
		}
		result = strings.Join(strs, n.joiner)
	case joinBuffer:
		chunks := make([][]byte, 0, len(g.values))
		for _, v := range g.values {
			switch b := v.(type) {
			case []byte:
				chunks = append(chunks, b)
			default:
				chunks = append(chunks, []byte(fmt.Sprint(nilToEmpty(b))))
			}
		}
		result = bytes.Join(chunks, []byte(n.joiner))
	case joinObject:
		obj := map[string]any{}
		for i, v := range g.values {
			obj[g.keys[i]] = v
		}
		result = obj
	case joinMerged:
		obj := map[string]any{}
		for _, v := range g.values {
			if m, ok := v.(map[string]any); ok {
				for k, val := range m {
					obj[k] = val
				}
			}
		}
		result = obj
	default:
		result = append([]any(nil), g.values...)
	}

	out := g.last.Clone()
	if n.property == "" {
		// Whole messages: the result goes in msg.payload.
		out = flow.NewMessage()
		out.SetPayload(result)
	} else {
		out.Set(n.property, result)
	}
	out.Delete("complete")
	out.Delete("parts")
	return out
}

// updateStatus shows how many parts are waiting. Caller holds mu.
func (n *JoinNode) updateStatus() {
	if n.Status == nil {
		return
	}
	waiting := 0
	for _, g := range n.groups {
		waiting += len(g.items)
	}
	if n.manual != nil {
		waiting += len(n.manual.values)
	}
	if waiting == 0 {
		n.Status("", "")
		return
	}
	n.Status("blue", fmt.Sprintf("%d waiting", waiting))
}

// JoinTypeInfo describes the join node.
func JoinTypeInfo() flow.NodeTypeInfo {
	return flow.NodeTypeInfo{
		Type:        "join",
		Category:    "sequence",
		Label:       "Join",
		Description: "Joins a sequence of messages into one",
		Icon:        "mdi-call-merge",
		Keywords:    []string{"array", "combine", "merge", "collect", "sequence"},
		Defaults: map[string]any{
			"mode":         joinAuto,
			"build":        joinArray,
			"property":     "payload",
			"propertyType": "msg",
			"key":          "topic",
			"joiner":       `\n`,
			"count":        0.0,
			"timeout":      0.0,
		},
		Inputs:  1,
		Outputs: 1,
	}
}
