// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package gpio

import (
	"testing"

	orbitclient "github.com/OrbitOS-org/orbit-os-sdk-go/v26/client"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/flow"
)

func TestNewGpioPinConfig_RequiresLine(t *testing.T) {
	_, err := NewGpioPinConfig(flow.ConfigNode{
		ID:     "p1",
		Config: map[string]any{"direction": "input"},
	})
	if err == nil {
		t.Fatal("expected error for missing pin line, got nil")
	}
}

func TestNewGpioPinConfig_RejectsInvalidDirection(t *testing.T) {
	_, err := NewGpioPinConfig(flow.ConfigNode{
		ID:     "p1",
		Config: map[string]any{"line": float64(17), "direction": "sideways"},
	})
	if err == nil {
		t.Fatal("expected error for invalid direction, got nil")
	}
}

func TestNewGpioPinConfig_DefaultsToInput(t *testing.T) {
	inst, err := NewGpioPinConfig(flow.ConfigNode{
		ID:     "p1",
		Config: map[string]any{"line": float64(17)},
	})
	if err != nil {
		t.Fatalf("NewGpioPinConfig: %v", err)
	}
	pin := inst.(*GpioPinConfig)
	if pin.Direction() != orbitclient.GPIO_DIR_IN {
		t.Errorf("Direction() = %v, want GPIO_DIR_IN (default)", pin.Direction())
	}
}

func TestNewGpioPinConfig_OutputDirection(t *testing.T) {
	inst, err := NewGpioPinConfig(flow.ConfigNode{
		ID:     "p1",
		Config: map[string]any{"line": float64(17), "chip": float64(0), "direction": "output"},
	})
	if err != nil {
		t.Fatalf("NewGpioPinConfig: %v", err)
	}
	pin := inst.(*GpioPinConfig)
	if pin.Direction() != orbitclient.GPIO_DIR_OUT {
		t.Errorf("Direction() = %v, want GPIO_DIR_OUT", pin.Direction())
	}
}

func TestGpioPinConfig_PinLabel(t *testing.T) {
	named, _ := NewGpioPinConfig(flow.ConfigNode{
		ID:     "p1",
		Config: map[string]any{"line": float64(17), "chip": float64(0), "name": "GPIO17"},
	})
	if got := named.(*GpioPinConfig).PinLabel(); got != "GPIO17" {
		t.Errorf("PinLabel() = %q, want %q", got, "GPIO17")
	}

	unnamed, _ := NewGpioPinConfig(flow.ConfigNode{
		ID:     "p2",
		Config: map[string]any{"line": float64(4), "chip": float64(1)},
	})
	if got, want := unnamed.(*GpioPinConfig).PinLabel(), "chip1/line4"; got != want {
		t.Errorf("PinLabel() = %q, want %q", got, want)
	}
}

func TestGpioPinConfig_StatusLifecycle(t *testing.T) {
	inst, _ := NewGpioPinConfig(flow.ConfigNode{
		ID:     "p1",
		Config: map[string]any{"line": float64(17)},
	})
	pin := inst.(*GpioPinConfig)

	fill, text := pin.Status()
	if fill != "grey" || text != "not started" {
		t.Errorf("initial status = (%q, %q), want (grey, not started)", fill, text)
	}

	var got [][2]string
	pin.RegisterStatusFunc(func(f, tx string) { got = append(got, [2]string{f, tx}) })
	if len(got) != 1 || got[0][0] != "grey" {
		t.Errorf("RegisterStatusFunc should deliver current state immediately, got %v", got)
	}

	_ = pin.Stop()
	fill, text = pin.Status()
	if fill != "grey" || text != "stopped" {
		t.Errorf("status after Stop = (%q, %q), want (grey, stopped)", fill, text)
	}
}
