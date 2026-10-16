// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package pwm

import (
	"fmt"
	"sync"
	"testing"

	orbitclient "github.com/OrbitOS-org/orbit-os-sdk-go/v26/client"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/flow"
)

// fakeDriver records the calls.
type fakeDriver struct {
	mu    sync.Mutex
	calls []string
}

func (f *fakeDriver) SetPwm(ch *orbitclient.PwmChannel, duty, hz float64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, fmt.Sprintf("set %d %.3f %.0f", ch.Channel, duty, hz))
	return nil
}

func (f *fakeDriver) StopPwm(ch *orbitclient.PwmChannel) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, fmt.Sprintf("stop %d", ch.Channel))
	return nil
}

func (f *fakeDriver) list() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.calls...)
}

func useFake(t *testing.T) *fakeDriver {
	f := &fakeDriver{}
	old := newDriver
	newDriver = func() driver { return f }
	t.Cleanup(func() { newDriver = old })
	return f
}

func msg(payload any) *flow.Message {
	m := flow.NewMessage()
	m.SetPayload(payload)
	return m
}

func newNode(t *testing.T, props map[string]any) *PwmOutNode {
	t.Helper()
	n, err := NewPwmOutNode(flow.NodeConfig{ID: "p1", Type: "pwm-out", Properties: props})
	if err != nil {
		t.Fatal(err)
	}
	if err := n.Init(); err != nil {
		t.Fatal(err)
	}
	if err := n.Start(); err != nil {
		t.Fatal(err)
	}
	return n.(*PwmOutNode)
}

func TestPwmOut_DutyScales(t *testing.T) {
	f := useFake(t)
	cases := []struct {
		scale string
		in    any
	}{
		{"percent", 40.0}, {"percent", "40%"}, {"unit", 0.4}, {"byte", 102.0},
	}
	for _, c := range cases {
		n := newNode(t, map[string]any{"channel": 1.0, "frequency": 25000.0, "scale": c.scale})
		if _, err := n.HandleMessage(msg(c.in)); err != nil {
			t.Fatalf("%s %v: %v", c.scale, c.in, err)
		}
	}
	for _, call := range f.list() {
		if call != "set 1 0.400 25000" {
			t.Errorf("got %q, want set 1 0.400 25000", call)
		}
	}
}

func TestPwmOut_ClampsAndRejects(t *testing.T) {
	f := useFake(t)
	n := newNode(t, map[string]any{"channel": 0.0})
	_, _ = n.HandleMessage(msg(150.0))
	_, _ = n.HandleMessage(msg(true))
	_, _ = n.HandleMessage(msg(-5.0))
	if _, err := n.HandleMessage(msg("fast")); err == nil {
		t.Error("want an error for a payload that is not a number")
	}
	got := f.list()
	want := []string{"set 0 1.000 1000", "set 0 1.000 1000", "set 0 0.000 1000"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestPwmOut_FrequencyFromMessage(t *testing.T) {
	f := useFake(t)
	n := newNode(t, map[string]any{"channel": 2.0})
	m := msg(50.0)
	m.Set("frequency", 50.0)
	_, _ = n.HandleMessage(m)
	if got := f.list(); len(got) != 1 || got[0] != "set 2 0.500 50" {
		t.Fatalf("got %v", got)
	}
}

func TestPwmOut_SafeStates(t *testing.T) {
	f := useFake(t)
	newNode(t, map[string]any{"channel": 3.0}).ApplySafeState()
	newNode(t, map[string]any{"channel": 3.0, "onStop": "value", "stopValue": 20.0}).ApplySafeState()
	newNode(t, map[string]any{"channel": 3.0, "onStop": "keep"}).ApplySafeState()
	want := []string{"set 3 0.000 1000", "stop 3", "set 3 0.200 1000"}
	if got := f.list(); fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if k := newNode(t, map[string]any{"channel": 3.0}).OutputKey(); k != "pwm:3" {
		t.Fatalf("key %q", k)
	}
}

func TestPwmOut_RequiresChannel(t *testing.T) {
	useFake(t)
	n, _ := NewPwmOutNode(flow.NodeConfig{ID: "p", Properties: map[string]any{}})
	if err := n.Init(); err == nil {
		t.Fatal("want an error without a channel")
	}
}
