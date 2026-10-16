// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package core

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/flow"
	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/nodes"
)

// Filter modes, with Node-RED's names.
const (
	filterRBE          = "rbe"          // pass when the value changes
	filterRBEI         = "rbei"         // same, but the first value does not pass
	filterDeadband     = "deadband"     // pass when it changes by more than gap
	filterDeadbandEq   = "deadbandEq"   // ... by gap or more
	filterNarrowband   = "narrowband"   // block when it changes by more than gap
	filterNarrowbandEq = "narrowbandEq" // ... by gap or more
)

// FilterNode is Node-RED's "filter" (report by exception, type rbe): it
// passes a message only when the watched property changes, or changes by
// more (or less) than a gap. Each msg.topic keeps its own last value unless
// "separate topics" is off. A message with msg.reset clears the stored
// value(s) and is then handled normally.
type FilterNode struct {
	nodes.BaseNode
	config    flow.NodeConfig
	mode      string
	gap       float64
	percent   bool
	compareIn bool // compare with the last input (true) or the last output
	property  string
	septopics bool
	topicProp string

	mu       sync.Mutex
	previous map[string]any
}

// NewFilterNode is the NodeFactory for the filter node type.
func NewFilterNode(config flow.NodeConfig) (flow.NodeInstance, error) {
	return &FilterNode{config: config, previous: map[string]any{}}, nil
}

// Init parses the configuration.
func (n *FilterNode) Init() error {
	p := n.config.Properties
	n.mode = nodes.StringVal(p, "func", filterRBE)
	switch n.mode {
	case filterRBE, filterRBEI, filterDeadband, filterDeadbandEq, filterNarrowband, filterNarrowbandEq:
	default:
		return fmt.Errorf("filter: unknown mode %q", n.mode)
	}
	gap := strings.TrimSpace(fmt.Sprint(p["gap"]))
	if p["gap"] == nil {
		gap = "0"
	}
	if strings.HasSuffix(gap, "%") {
		n.percent = true
		gap = strings.TrimSuffix(gap, "%")
	}
	if n.isBand() {
		g, err := strconv.ParseFloat(strings.TrimSpace(gap), 64)
		if err != nil || g < 0 {
			return fmt.Errorf("filter: the change must be a number of 0 or more, got %q", p["gap"])
		}
		n.gap = g
	}
	n.compareIn = nodes.StringVal(p, "inout", "out") == "in"
	n.property = nodes.StringVal(p, "property", "payload")
	n.septopics = nodes.BoolVal(p, "septopics", true)
	n.topicProp = nodes.StringVal(p, "topi", "topic")
	return nil
}

func (n *FilterNode) isBand() bool {
	return n.mode != filterRBE && n.mode != filterRBEI
}

// Start has nothing to start.
func (n *FilterNode) Start() error { return nil }

// Stop forgets the stored values.
func (n *FilterNode) Stop() error {
	n.mu.Lock()
	n.previous = map[string]any{}
	n.mu.Unlock()
	return nil
}

// HandleMessage passes or drops msg.
func (n *FilterNode) HandleMessage(msg *flow.Message) ([][]*flow.Message, error) {
	key := ""
	if n.septopics {
		key = topicKey(msg.Get(n.topicProp))
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	if msg.Get("reset") != nil {
		if n.septopics && key != "" {
			delete(n.previous, key)
		} else {
			n.previous = map[string]any{}
		}
	}

	value := msg.Get(n.property)
	if value == nil {
		return nil, nil
	}
	pass := [][]*flow.Message{{msg}}

	if !n.isBand() {
		prev, seen := n.previous[key]
		if seen && sameValue(prev, value) {
			return nil, nil
		}
		n.previous[key] = value
		if !seen && n.mode == filterRBEI {
			return nil, nil
		}
		return pass, nil
	}

	v, ok := toNumber(value)
	if !ok {
		n.warn(fmt.Sprintf("not a number: %v", value))
		return nil, nil
	}
	prevAny, seen := n.previous[key]
	if !seen {
		// Deadband: the first value always passes. Narrowband: the first
		// value is the reference and passes.
		n.previous[key] = v
		return pass, nil
	}
	prev := prevAny.(float64)
	gap := n.gap
	if n.percent {
		gap = math.Abs(prev * n.gap / 100)
	}
	diff := math.Abs(v - prev)
	var send bool
	switch n.mode {
	case filterDeadband:
		send = diff > gap
	case filterDeadbandEq:
		send = diff >= gap
	case filterNarrowband:
		send = diff <= gap
	case filterNarrowbandEq:
		send = diff < gap
	}
	if n.compareIn || send {
		n.previous[key] = v
	}
	if !send {
		return nil, nil
	}
	return pass, nil
}

func (n *FilterNode) warn(text string) {
	if n.Status != nil {
		n.Status("yellow", text)
	}
}

// FilterTypeInfo describes the filter node.
func FilterTypeInfo() flow.NodeTypeInfo {
	return flow.NodeTypeInfo{
		Type:        "filter",
		Category:    "function",
		Label:       "Filter",
		Description: "Passes a message only when the value changes (report by exception)",
		Icon:        "mdi-filter",
		Keywords:    []string{"rbe", "deadband", "change", "report by exception"},
		Defaults: map[string]any{
			"func":      filterRBE,
			"gap":       "",
			"inout":     "out",
			"property":  "payload",
			"septopics": true,
			"topi":      "topic",
		},
		Inputs:  1,
		Outputs: 1,
	}
}
