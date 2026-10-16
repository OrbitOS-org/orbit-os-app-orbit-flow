// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package flow_test

import (
	"sort"
	"sync"
	"testing"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/config"
	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/flow"
)

// outputNode drives the output named in its "pin" property and records
// when the engine asks for the safe state.
type outputNode struct {
	pin  string
	safe func(pin string)
}

func (n *outputNode) Init() error                 { return nil }
func (n *outputNode) SetSend(_ flow.SendFunc)     {}
func (n *outputNode) SetStatus(_ flow.StatusFunc) {}
func (n *outputNode) SetDebug(_ flow.DebugFunc)   {}
func (n *outputNode) Start() error                { return nil }
func (n *outputNode) Stop() error                 { return nil }
func (n *outputNode) HandleMessage(_ *flow.Message) ([][]*flow.Message, error) {
	return nil, nil
}
func (n *outputNode) OutputKey() string { return "gpio:" + n.pin }
func (n *outputNode) ApplySafeState()   { n.safe(n.pin) }

func newSafeRig(t *testing.T) (*flow.Engine, func() []string) {
	t.Helper()
	engine := flow.NewEngine(&config.Config{})
	var mu sync.Mutex
	var safe []string
	engine.Registry().Register("test-out", func(c flow.NodeConfig) (flow.NodeInstance, error) {
		pin, _ := c.Properties["pin"].(string)
		return &outputNode{pin: pin, safe: func(p string) {
			mu.Lock()
			safe = append(safe, p)
			mu.Unlock()
		}}, nil
	}, flow.NodeTypeInfo{Type: "test-out", Inputs: 1, Outputs: 0})
	if err := engine.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = engine.Stop() })
	return engine, func() []string {
		mu.Lock()
		defer mu.Unlock()
		out := append([]string(nil), safe...)
		safe = nil
		sort.Strings(out)
		return out
	}
}

func outFlow(disabled bool, nodes ...flow.Node) []flow.Flow {
	return []flow.Flow{{ID: "f1", Type: "tab", Disabled: disabled, Nodes: nodes}}
}

func out(id, pin, note string) flow.Node {
	return flow.Node{ID: id, Type: "test-out", Z: "f1", Inputs: 1, Config: map[string]any{"pin": pin, "note": note}}
}

func TestSafeOutputs(t *testing.T) {
	for _, mode := range []flow.DeployMode{flow.DeployFull, flow.DeployModifiedFlows, flow.DeployModifiedNodes} {
		t.Run(string(mode), func(t *testing.T) {
			engine, safe := newSafeRig(t)
			deploy := func(f []flow.Flow) {
				t.Helper()
				if err := engine.Deploy(f, nil, mode); err != nil {
					t.Fatal(err)
				}
			}

			deploy(outFlow(false, out("a", "17", ""), out("b", "23", "")))
			if got := safe(); len(got) != 0 {
				t.Fatalf("first deploy: %v", got)
			}

			// The node is changed and restarted, but still drives its pin: no safe state.
			deploy(outFlow(false, out("a", "17", "changed"), out("b", "23", "")))
			if got := safe(); len(got) != 0 {
				t.Fatalf("deploy keeping the nodes: %v", got)
			}

			// Node b deleted, node a moved to another pin: 23 and 17 go safe.
			deploy(outFlow(false, out("a", "5", "")))
			if got := safe(); len(got) != 2 || got[0] != "17" || got[1] != "23" {
				t.Fatalf("delete and move: %v", got)
			}

			// Flow disabled.
			deploy(outFlow(true, out("a", "5", "")))
			if got := safe(); len(got) != 1 || got[0] != "5" {
				t.Fatalf("disabled flow: %v", got)
			}

			// Stop Flows (an empty deploy) and the engine stopping.
			deploy(outFlow(false, out("a", "5", "")))
			deploy(nil)
			if got := safe(); len(got) != 1 || got[0] != "5" {
				t.Fatalf("stop flows: %v", got)
			}
			deploy(outFlow(false, out("a", "6", "")))
			_ = engine.Stop()
			if got := safe(); len(got) != 1 || got[0] != "6" {
				t.Fatalf("engine stop: %v", got)
			}
		})
	}
}
