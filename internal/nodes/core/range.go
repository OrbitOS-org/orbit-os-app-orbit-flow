// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package core

import (
	"fmt"
	"math"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/flow"
	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/nodes"
)

// Range actions, with Node-RED's names.
const (
	rangeScale = "scale" // scale, values outside the input range go outside the output range
	rangeClamp = "clamp" // limit to the input range first
	rangeRoll  = "roll"  // wrap around within the input range first
	rangeDrop  = "drop"  // drop values outside the input range
)

// RangeNode is Node-RED's "range": it maps a number from one range to
// another, e.g. a 12-bit ADC reading (0–4095) to a percentage (0–100).
type RangeNode struct {
	nodes.BaseNode
	config   flow.NodeConfig
	action   string
	minIn    float64
	maxIn    float64
	minOut   float64
	maxOut   float64
	round    bool
	property string
}

// NewRangeNode is the NodeFactory for the range node type.
func NewRangeNode(config flow.NodeConfig) (flow.NodeInstance, error) {
	return &RangeNode{config: config}, nil
}

// Init parses the configuration.
func (n *RangeNode) Init() error {
	p := n.config.Properties
	n.action = nodes.StringVal(p, "action", rangeScale)
	switch n.action {
	case rangeScale, rangeClamp, rangeRoll, rangeDrop:
	default:
		return fmt.Errorf("range: unknown action %q", n.action)
	}
	var ok bool
	for _, f := range []struct {
		key string
		dst *float64
	}{{"minin", &n.minIn}, {"maxin", &n.maxIn}, {"minout", &n.minOut}, {"maxout", &n.maxOut}} {
		if *f.dst, ok = toNumber(p[f.key]); !ok {
			return fmt.Errorf("range: %s must be a number", f.key)
		}
	}
	if n.minIn == n.maxIn {
		return fmt.Errorf("range: the input range is empty (from %v to %v)", n.minIn, n.maxIn)
	}
	n.round = nodes.BoolVal(p, "round", false)
	n.property = nodes.StringVal(p, "property", "payload")
	return nil
}

// Start has nothing to start.
func (n *RangeNode) Start() error { return nil }

// Stop has nothing to stop.
func (n *RangeNode) Stop() error { return nil }

// HandleMessage scales the property and forwards the message.
func (n *RangeNode) HandleMessage(msg *flow.Message) ([][]*flow.Message, error) {
	raw := msg.Get(n.property)
	if raw == nil {
		return nil, nil
	}
	v, ok := toNumber(raw)
	if !ok {
		if n.Status != nil {
			n.Status("yellow", fmt.Sprintf("not a number: %v", raw))
		}
		return nil, nil
	}
	out, keep := n.scale(v)
	if !keep {
		return nil, nil
	}
	msg.Set(n.property, out)
	return [][]*flow.Message{{msg}}, nil
}

// scale applies the action; keep is false when the value is dropped.
func (n *RangeNode) scale(v float64) (float64, bool) {
	lo, hi := math.Min(n.minIn, n.maxIn), math.Max(n.minIn, n.maxIn)
	switch n.action {
	case rangeClamp:
		v = math.Max(lo, math.Min(hi, v))
	case rangeRoll:
		span := hi - lo
		v = math.Mod(math.Mod(v-lo, span)+span, span) + lo
	case rangeDrop:
		if v < lo || v > hi {
			return 0, false
		}
	}
	out := (v-n.minIn)/(n.maxIn-n.minIn)*(n.maxOut-n.minOut) + n.minOut
	if n.round {
		out = math.Round(out)
	}
	return out, true
}

// RangeTypeInfo describes the range node.
func RangeTypeInfo() flow.NodeTypeInfo {
	return flow.NodeTypeInfo{
		Type:        "range",
		Category:    "function",
		Label:       "Range",
		Description: "Maps a number from one range to another",
		Icon:        "mdi-arrow-expand-horizontal",
		Keywords:    []string{"scale", "map", "convert", "percent"},
		Defaults: map[string]any{
			"action":   rangeScale,
			"minin":    0.0,
			"maxin":    4095.0,
			"minout":   0.0,
			"maxout":   100.0,
			"round":    false,
			"property": "payload",
		},
		Inputs:  1,
		Outputs: 1,
	}
}
