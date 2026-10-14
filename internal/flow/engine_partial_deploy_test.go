// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package flow_test

import (
	"sync"
	"testing"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/config"
	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/flow"
)

// countNode counts how many times the engine starts it.
type countNode struct {
	errNode
	id     string
	starts map[string]int
	mu     *sync.Mutex
}

func (n *countNode) Start() error {
	n.mu.Lock()
	n.starts[n.id]++
	n.mu.Unlock()
	return nil
}

func (n *countNode) Stop() error {
	n.mu.Lock()
	n.starts[n.id+":stops"]++
	n.mu.Unlock()
	return nil
}

func twoFlows(stopB bool) []flow.Flow {
	node := func(id, flowID string) flow.Node {
		return flow.Node{ID: id, Type: "count", Z: flowID, Inputs: 1, Outputs: 0, Config: map[string]any{}}
	}
	return []flow.Flow{
		{ID: "A", Label: "A", Nodes: []flow.Node{node("a1", "A")}},
		{ID: "B", Label: "B", Disabled: stopB, Nodes: []flow.Node{node("b1", "B")}},
	}
}

// A modified-flows deploy (used to stop, start or deploy one flow) must not
// start the nodes of unchanged flows again. It used to: they ran two loops
// and the next stop panicked with "close of closed channel".
func TestModifiedFlowsDeployLeavesOtherFlowsAlone(t *testing.T) {
	engine := flow.NewEngine(&config.Config{})
	var mu sync.Mutex
	starts := map[string]int{}
	engine.Registry().Register("count", func(c flow.NodeConfig) (flow.NodeInstance, error) {
		return &countNode{id: c.ID, starts: starts, mu: &mu}, nil
	}, flow.NodeTypeInfo{Type: "count", Inputs: 1, Outputs: 0})
	if err := engine.Start(); err != nil {
		t.Fatal(err)
	}

	if err := engine.Deploy(twoFlows(false), nil, flow.DeployFull); err != nil {
		t.Fatal(err)
	}
	// Stop B, start B, stop B again: each time only B changes.
	for i, stopB := range []bool{true, false, true, false} {
		if err := engine.Deploy(twoFlows(stopB), nil, flow.DeployModifiedFlows); err != nil {
			t.Fatalf("partial deploy %d: %v", i, err)
		}
	}

	mu.Lock()
	a := starts["a1"]
	mu.Unlock()
	if a != 1 {
		t.Errorf("node of the unchanged flow A was started %d times, want 1", a)
	}
	if err := engine.Stop(); err != nil { // panicked before the fix
		t.Fatal(err)
	}
}

// The default Deploy (modified nodes) must apply disabling and enabling a
// flow. It used to see no changed node and keep the flow running.
func TestModifiedNodesDeployAppliesDisable(t *testing.T) {
	engine := flow.NewEngine(&config.Config{})
	var mu sync.Mutex
	starts := map[string]int{}
	engine.Registry().Register("count", func(c flow.NodeConfig) (flow.NodeInstance, error) {
		return &countNode{id: c.ID, starts: starts, mu: &mu}, nil
	}, flow.NodeTypeInfo{Type: "count", Inputs: 1, Outputs: 0})
	if err := engine.Start(); err != nil {
		t.Fatal(err)
	}
	if err := engine.Deploy(twoFlows(false), nil, flow.DeployFull); err != nil {
		t.Fatal(err)
	}
	get := func(k string) int { mu.Lock(); defer mu.Unlock(); return starts[k] }

	if err := engine.Deploy(twoFlows(true), nil, flow.DeployModifiedNodes); err != nil {
		t.Fatal(err)
	}
	if get("b1:stops") != 1 {
		t.Errorf("disabling flow B stopped its node %d times, want 1", get("b1:stops"))
	}
	if err := engine.Deploy(twoFlows(false), nil, flow.DeployModifiedNodes); err != nil {
		t.Fatal(err)
	}
	if get("b1") != 2 {
		t.Errorf("node of flow B was started %d times, want 2 (deploy, enable)", get("b1"))
	}
	if get("a1") != 1 || get("a1:stops") != 0 {
		t.Errorf("node of flow A: %d starts, %d stops, want 1 and 0", get("a1"), get("a1:stops"))
	}
	if err := engine.Stop(); err != nil {
		t.Fatal(err)
	}
}
