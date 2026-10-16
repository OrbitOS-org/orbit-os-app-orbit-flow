// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package core

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/flow"
	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/nodes"
)

// SplitNode is Node-RED's "split": it turns one message into a sequence,
// one message per item of a list, per line (or other separator) of a text,
// per key of an object, or per chunk of bytes. Each message carries
// msg.parts {id, index, count, type, ...}, which the join node uses to put
// the sequence back together. A message that is already part of a sequence
// keeps its old msg.parts in msg.parts.parts.
type SplitNode struct {
	nodes.BaseNode
	config   flow.NodeConfig
	property string
	sep      string // text and bytes separator
	byLength bool   // text and bytes: fixed-length chunks instead of a separator
	length   int    // text and bytes chunk length
	arrayLen int    // list chunk size
	addName  string // object: copy the key to this message property ("" = no)
}

// NewSplitNode is the NodeFactory for the split node type.
func NewSplitNode(config flow.NodeConfig) (flow.NodeInstance, error) {
	return &SplitNode{config: config}, nil
}

// Init parses the configuration.
func (n *SplitNode) Init() error {
	p := n.config.Properties
	n.property = nodes.StringVal(p, "property", "payload")
	splt := nodes.StringVal(p, "splt", `\n`)
	n.byLength = nodes.StringVal(p, "spltType", "str") == "len"
	if n.byLength {
		l, err := strconv.Atoi(strings.TrimSpace(splt))
		if err != nil || l < 1 {
			return fmt.Errorf("split: the length must be a whole number of 1 or more, got %q", splt)
		}
		n.length = l
	} else {
		n.sep = unescapeSeparator(splt)
		if n.sep == "" {
			return fmt.Errorf("split: the separator is empty")
		}
	}
	n.arrayLen = nodes.IntVal(p, "arraySplt", 1)
	if n.arrayLen < 1 {
		return fmt.Errorf("split: the list chunk size must be 1 or more")
	}
	n.addName = nodes.StringVal(p, "addname", "")
	return nil
}

// unescapeSeparator turns the \n, \r, \t and \\ typed in the editor into
// the characters, as Node-RED does.
func unescapeSeparator(s string) string {
	r := strings.NewReplacer(`\\`, `\`, `\n`, "\n", `\r`, "\r", `\t`, "\t")
	return r.Replace(s)
}

// Start has nothing to start.
func (n *SplitNode) Start() error { return nil }

// Stop has nothing to stop.
func (n *SplitNode) Stop() error { return nil }

// HandleMessage splits the property into a sequence of messages.
func (n *SplitNode) HandleMessage(msg *flow.Message) ([][]*flow.Message, error) {
	value := msg.Get(n.property)
	var pieces []any
	parts := map[string]any{}
	if old, ok := msg.Get("parts").(map[string]any); ok {
		parts["parts"] = old
	}
	// Each piece is a copy of the message without the value being split, so
	// a long list is not copied once per item.
	base := msg.Clone()
	base.Delete(n.property)

	switch v := value.(type) {
	case nil:
		return nil, nil
	case string:
		parts["type"] = "string"
		if n.byLength {
			r := []rune(v)
			for i := 0; i < len(r); i += n.length {
				pieces = append(pieces, string(r[i:min(i+n.length, len(r))]))
			}
			parts["len"] = float64(n.length)
		} else {
			for _, s := range strings.Split(v, n.sep) {
				pieces = append(pieces, s)
			}
			parts["ch"] = n.sep
		}
	case []byte:
		parts["type"] = "buffer"
		if n.byLength {
			for i := 0; i < len(v); i += n.length {
				pieces = append(pieces, append([]byte(nil), v[i:min(i+n.length, len(v))]...))
			}
			parts["len"] = float64(n.length)
		} else {
			for _, b := range bytes.Split(v, []byte(n.sep)) {
				pieces = append(pieces, b)
			}
			parts["ch"] = n.sep
		}
	case []any:
		parts["type"] = "array"
		parts["len"] = float64(n.arrayLen)
		for i := 0; i < len(v); i += n.arrayLen {
			if n.arrayLen == 1 {
				pieces = append(pieces, v[i])
			} else {
				pieces = append(pieces, append([]any(nil), v[i:min(i+n.arrayLen, len(v))]...))
			}
		}
	case map[string]any:
		parts["type"] = "object"
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys) // Go maps have no order; keep the output stable
		out := make([]*flow.Message, 0, len(keys))
		id := newSeqID()
		for i, k := range keys {
			m := n.piece(base, v[k], id, i, len(keys), parts)
			m.Set("parts.key", k)
			if n.addName != "" {
				m.Set(n.addName, k)
			}
			out = append(out, m)
		}
		return [][]*flow.Message{out}, nil
	default:
		// A single value (number, true/false) is a sequence of one, as in Node-RED.
		return [][]*flow.Message{{msg}}, nil
	}

	id := newSeqID()
	out := make([]*flow.Message, 0, len(pieces))
	for i, piece := range pieces {
		out = append(out, n.piece(base, piece, id, i, len(pieces), parts))
	}
	return [][]*flow.Message{out}, nil
}

// piece builds message i of count: a copy of base with the property set to
// the piece and msg.parts describing its place in the sequence.
func (n *SplitNode) piece(base *flow.Message, value any, id string, i, count int, common map[string]any) *flow.Message {
	m := base.Clone()
	parts := map[string]any{"id": id, "index": float64(i), "count": float64(count)}
	for k, v := range common {
		parts[k] = v
	}
	if n.property != "payload" {
		parts["property"] = n.property
	}
	m.Set(n.property, value)
	m.Set("parts", parts)
	return m
}

// SplitTypeInfo describes the split node.
func SplitTypeInfo() flow.NodeTypeInfo {
	return flow.NodeTypeInfo{
		Type:        "split",
		Category:    "sequence",
		Label:       "Split",
		Description: "Splits a list, text, object or bytes into one message per item",
		Icon:        "mdi-call-split",
		Keywords:    []string{"array", "lines", "each", "sequence"},
		Defaults: map[string]any{
			"property":  "payload",
			"splt":      `\n`,
			"spltType":  "str",
			"arraySplt": 1.0,
			"addname":   "",
		},
		Inputs:  1,
		Outputs: 1,
	}
}
