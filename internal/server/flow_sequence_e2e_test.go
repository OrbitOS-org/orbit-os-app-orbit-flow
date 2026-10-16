// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package server_test

import (
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"testing"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/config"
	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/flow"
	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/nodes/core"
	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/nodes/network"
)

// listNode sets msg.payload to a fixed list.
type listNode struct{ list []any }

func (n *listNode) Init() error                 { return nil }
func (n *listNode) SetSend(_ flow.SendFunc)     {}
func (n *listNode) SetStatus(_ flow.StatusFunc) {}
func (n *listNode) SetDebug(_ flow.DebugFunc)   {}
func (n *listNode) Start() error                { return nil }
func (n *listNode) Stop() error                 { return nil }
func (n *listNode) HandleMessage(msg *flow.Message) ([][]*flow.Message, error) {
	msg.SetPayload(n.list)
	return [][]*flow.Message{{msg}}, nil
}

// TestEndToEndSequenceNodes runs split → range → filter → join inside a real
// engine between http-in and http-response, and a split/join of a list much
// longer than a node's input buffer (64), which must lose nothing.
func TestEndToEndSequenceNodes(t *testing.T) {
	cfg := &config.Config{HTTPNodeRoot: "/endpoint"}
	engine := flow.NewEngine(cfg)
	reg := engine.Registry()
	reg.Register("http-in", network.NewHTTPInNode, network.HTTPInTypeInfo())
	reg.Register("http-response", network.NewHTTPResponseNode, network.HTTPResponseTypeInfo())
	reg.Register("split", core.NewSplitNode, core.SplitTypeInfo())
	reg.Register("join", core.NewJoinNode, core.JoinTypeInfo())
	reg.Register("range", core.NewRangeNode, core.RangeTypeInfo())
	reg.Register("filter", core.NewFilterNode, core.FilterTypeInfo())
	big := make([]any, 2000)
	for i := range big {
		big[i] = float64(i)
	}
	reg.Register("test-adc", func(_ flow.NodeConfig) (flow.NodeInstance, error) {
		return &listNode{list: []any{0.0, 1024.0, 2048.0, 2048.0, 4095.0, 9999.0}}, nil
	}, flow.NodeTypeInfo{Type: "test-adc", Inputs: 1, Outputs: 1})
	reg.Register("test-big", func(_ flow.NodeConfig) (flow.NodeInstance, error) {
		return &listNode{list: big}, nil
	}, flow.NodeTypeInfo{Type: "test-big", Inputs: 1, Outputs: 1})

	tsrv, _ := buildE2EHarness(t, engine)
	defer tsrv.Close()
	if err := engine.Start(); err != nil {
		t.Fatal(err)
	}
	defer engine.Stop()

	node := func(id, typ string, wires [][]string, cfg map[string]any) flow.Node {
		return flow.Node{ID: id, Type: typ, Z: "f1", Inputs: 1, Outputs: len(wires), Wires: wires, Config: cfg}
	}
	in := func(id, path, next string) flow.Node {
		return flow.Node{ID: id, Type: "http-in", Z: "f1", Inputs: 0, Outputs: 1, Wires: [][]string{{next}},
			Config: map[string]any{"method": "GET", "path": path, "bodyParse": "auto"}}
	}
	resp := func(id string) flow.Node {
		return flow.Node{ID: id, Type: "http-response", Z: "f1", Inputs: 1, Outputs: 0, Config: map[string]any{"statusCode": 200}}
	}
	flows := []flow.Flow{{ID: "f1", Type: "tab", Nodes: []flow.Node{
		in("in1", "/adc", "adc"),
		node("adc", "test-adc", [][]string{{"sp"}}, nil),
		node("sp", "split", [][]string{{"rg"}}, nil),
		node("rg", "range", [][]string{{"fl"}}, map[string]any{"action": "drop", "minin": 0.0, "maxin": 4095.0, "minout": 0.0, "maxout": 1000.0, "round": true}),
		node("fl", "filter", [][]string{{"jn"}}, map[string]any{"func": "rbe", "septopics": false}),
		node("jn", "join", [][]string{{"out1"}}, map[string]any{"mode": "custom", "build": "array", "count": 4.0}),
		resp("out1"),
		in("in2", "/big", "big"),
		node("big", "test-big", [][]string{{"sp2"}}, nil),
		node("sp2", "split", [][]string{{"jn2"}}, nil),
		node("jn2", "join", [][]string{{"out2"}}, map[string]any{"mode": "auto"}),
		resp("out2"),
	}}}
	if err := engine.Deploy(flows, nil, flow.DeployFull); err != nil {
		t.Fatalf("deploy: %v", err)
	}

	get := func(path string) []any {
		t.Helper()
		r, err := http.Get(tsrv.URL + "/endpoint" + path)
		if err != nil {
			t.Fatal(err)
		}
		defer r.Body.Close()
		body, _ := io.ReadAll(r.Body)
		var v []any
		if err := json.Unmarshal(body, &v); err != nil {
			t.Fatalf("%s: %v (%s)", path, err, body)
		}
		return v
	}

	// 9999 is outside the input range (dropped) and the second 2048 is not a
	// change (filtered): four values reach the join. (The target range goes
	// to 1000 because http-response sends a list of 0–255 numbers as bytes.)
	if got, want := get("/adc"), []any{0.0, 250.0, 500.0, 1000.0}; !reflect.DeepEqual(got, want) {
		t.Fatalf("/adc: got %v, want %v", got, want)
	}
	if got := get("/big"); !reflect.DeepEqual(got, big) {
		t.Fatalf("/big: got %d items back, want %d in order", len(got), len(big))
	}
}
