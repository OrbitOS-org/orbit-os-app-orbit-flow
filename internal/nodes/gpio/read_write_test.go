// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package gpio

import (
	"testing"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/flow"
)

func TestTruthy(t *testing.T) {
	cases := []struct {
		in   any
		want bool
	}{
		{true, true},
		{false, false},
		{float64(1), true},
		{float64(0), false},
		{int(1), true},
		{int(0), false},
		{"true", true},
		{"1", true},
		{"on", true},
		{"HIGH", true},
		{"false", false},
		{"off", false},
		{"", false},
		{nil, false},
		{"garbage", false},
	}
	for _, c := range cases {
		if got := truthy(c.in); got != c.want {
			t.Errorf("truthy(%#v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func lookupReturning(id string, inst flow.ConfigInstance) flow.ConfigLookupFunc {
	return func(configID string) (flow.ConfigInstance, bool) {
		if configID == id {
			return inst, true
		}
		return nil, false
	}
}

func TestGpioReadNode_Init_RequiresPin(t *testing.T) {
	n := &GpioReadNode{cfg: flow.NodeConfig{ID: "n1", Properties: map[string]any{}}}
	if err := n.Init(); err == nil {
		t.Fatal("expected error for missing pin reference, got nil")
	}
}

func TestGpioReadNode_Start_RejectsOutputPin(t *testing.T) {
	pinInst, _ := NewGpioPinConfig(flow.ConfigNode{
		ID:     "p1",
		Config: map[string]any{"line": float64(17), "direction": "output"},
	})

	n := &GpioReadNode{cfg: flow.NodeConfig{ID: "n1"}, pinID: "p1"}
	n.SetConfigLookup(lookupReturning("p1", pinInst))

	if err := n.Start(); err == nil {
		t.Fatal("expected error when gpio-read references an output-direction pin, got nil")
	}
}

func TestGpioReadNode_Start_AcceptsInputPin(t *testing.T) {
	pinInst, _ := NewGpioPinConfig(flow.ConfigNode{
		ID:     "p1",
		Config: map[string]any{"line": float64(17), "direction": "input"},
	})

	n := &GpioReadNode{cfg: flow.NodeConfig{ID: "n1"}, pinID: "p1"}
	n.SetConfigLookup(lookupReturning("p1", pinInst))

	if err := n.Start(); err != nil {
		t.Fatalf("Start: unexpected error for matching input pin: %v", err)
	}
}

func TestGpioWriteNode_Init_RequiresPin(t *testing.T) {
	n := &GpioWriteNode{cfg: flow.NodeConfig{ID: "n1", Properties: map[string]any{}}}
	if err := n.Init(); err == nil {
		t.Fatal("expected error for missing pin reference, got nil")
	}
}

func TestGpioWriteNode_Init_ActionDefaultsToPayload(t *testing.T) {
	n := &GpioWriteNode{cfg: flow.NodeConfig{ID: "n1", Properties: map[string]any{"pin": "p1"}}}
	if err := n.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if n.action != gpioActionPayload {
		t.Errorf("action = %q, want %q (default)", n.action, gpioActionPayload)
	}
}

func TestGpioWriteNode_Init_ParsesValidActions(t *testing.T) {
	for _, a := range []string{"payload", "on", "off", "toggle"} {
		n := &GpioWriteNode{cfg: flow.NodeConfig{ID: "n1", Properties: map[string]any{"pin": "p1", "action": a}}}
		if err := n.Init(); err != nil {
			t.Errorf("Init with action %q: unexpected error: %v", a, err)
		}
		if string(n.action) != a {
			t.Errorf("action = %q, want %q", n.action, a)
		}
	}
}

func TestGpioWriteNode_Init_RejectsInvalidAction(t *testing.T) {
	n := &GpioWriteNode{cfg: flow.NodeConfig{ID: "n1", Properties: map[string]any{"pin": "p1", "action": "sideways"}}}
	if err := n.Init(); err == nil {
		t.Fatal("expected error for invalid action, got nil")
	}
}

func TestGpioWriteNode_ResolveLevel_OnAlwaysTrue(t *testing.T) {
	n := &GpioWriteNode{action: gpioActionOn}
	msg := flow.NewMessage()
	msg.SetPayload(false) // action overrides payload entirely
	got, err := n.resolveLevel(msg)
	if err != nil {
		t.Fatalf("resolveLevel: %v", err)
	}
	if !got {
		t.Error("resolveLevel with action=on should be true regardless of payload")
	}
}

func TestGpioWriteNode_ResolveLevel_OffAlwaysFalse(t *testing.T) {
	n := &GpioWriteNode{action: gpioActionOff}
	msg := flow.NewMessage()
	msg.SetPayload(true)
	got, err := n.resolveLevel(msg)
	if err != nil {
		t.Fatalf("resolveLevel: %v", err)
	}
	if got {
		t.Error("resolveLevel with action=off should be false regardless of payload")
	}
}

func TestGpioWriteNode_ResolveLevel_PayloadUsesTruthy(t *testing.T) {
	n := &GpioWriteNode{action: gpioActionPayload}
	msg := flow.NewMessage()
	msg.SetPayload("on")
	got, err := n.resolveLevel(msg)
	if err != nil {
		t.Fatalf("resolveLevel: %v", err)
	}
	if !got {
		t.Error(`resolveLevel with action=payload and payload="on" should be true`)
	}
}

func TestGpioWriteNode_ResolveLevel_TogglePropagatesReadError(t *testing.T) {
	pinInst, _ := NewGpioPinConfig(flow.ConfigNode{
		ID:     "p1",
		Config: map[string]any{"line": float64(17), "direction": "output"},
	})
	n := &GpioWriteNode{action: gpioActionToggle, pin: pinInst.(*GpioPinConfig)}

	// No orbitsdk.Client() installed in this test binary, so the read that
	// toggle needs before it can flip the level must fail loudly rather
	// than silently defaulting to some level.
	if _, err := n.resolveLevel(flow.NewMessage()); err == nil {
		t.Fatal("resolveLevel with action=toggle: expected error with no OrbitOS SDK client connected, got nil")
	}
}

func TestGpioWriteNode_Start_RejectsInputPin(t *testing.T) {
	pinInst, _ := NewGpioPinConfig(flow.ConfigNode{
		ID:     "p1",
		Config: map[string]any{"line": float64(17), "direction": "input"},
	})

	n := &GpioWriteNode{cfg: flow.NodeConfig{ID: "n1"}, pinID: "p1"}
	n.SetConfigLookup(lookupReturning("p1", pinInst))

	if err := n.Start(); err == nil {
		t.Fatal("expected error when gpio-write references an input-direction pin, got nil")
	}
}

func TestGpioWriteNode_Start_AcceptsOutputPin(t *testing.T) {
	pinInst, _ := NewGpioPinConfig(flow.ConfigNode{
		ID:     "p1",
		Config: map[string]any{"line": float64(17), "direction": "output"},
	})

	n := &GpioWriteNode{cfg: flow.NodeConfig{ID: "n1"}, pinID: "p1"}
	n.SetConfigLookup(lookupReturning("p1", pinInst))

	if err := n.Start(); err != nil {
		t.Fatalf("Start: unexpected error for matching output pin: %v", err)
	}
}

func TestGpioReadWriteNode_HandleMessage_NilMessage(t *testing.T) {
	r := &GpioReadNode{cfg: flow.NodeConfig{ID: "n1"}}
	if out, err := r.HandleMessage(nil); out != nil || err != nil {
		t.Fatalf("gpio-read HandleMessage(nil) = (%v, %v), want (nil, nil)", out, err)
	}

	w := &GpioWriteNode{cfg: flow.NodeConfig{ID: "n2"}}
	if out, err := w.HandleMessage(nil); out != nil || err != nil {
		t.Fatalf("gpio-write HandleMessage(nil) = (%v, %v), want (nil, nil)", out, err)
	}
}
