// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package i2c

import (
	"testing"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/flow"
)

func TestNewI2CBusConfig_RequiresBus(t *testing.T) {
	_, err := NewI2CBusConfig(flow.ConfigNode{
		ID:     "b1",
		Config: map[string]any{"clockHz": float64(100_000)},
	})
	if err == nil {
		t.Fatal("expected error for missing bus, got nil")
	}
}

func TestNewI2CBusConfig_RejectsNonPositiveClock(t *testing.T) {
	_, err := NewI2CBusConfig(flow.ConfigNode{
		ID:     "b1",
		Config: map[string]any{"bus": float64(1), "clockHz": float64(0)},
	})
	if err == nil {
		t.Fatal("expected error for non-positive clockHz, got nil")
	}
}

func TestNewI2CBusConfig_Defaults(t *testing.T) {
	inst, err := NewI2CBusConfig(flow.ConfigNode{
		ID:     "b1",
		Config: map[string]any{"bus": float64(1)},
	})
	if err != nil {
		t.Fatalf("NewI2CBusConfig: %v", err)
	}
	c := inst.(*I2CBusConfig)
	if c.clockHz != 100_000 {
		t.Errorf("clockHz = %d, want default 100000", c.clockHz)
	}
	if c.tenBitAddr || c.clockStretching {
		t.Error("tenBitAddr/clockStretching should default to false")
	}
}

func TestI2CBusConfig_StatusLifecycle(t *testing.T) {
	inst, _ := NewI2CBusConfig(flow.ConfigNode{ID: "b1", Config: map[string]any{"bus": float64(1)}})
	c := inst.(*I2CBusConfig)

	fill, text := c.Status()
	if fill != "grey" || text != "not started" {
		t.Errorf("initial status = (%q, %q), want (grey, not started)", fill, text)
	}

	var got [][2]string
	c.RegisterStatusFunc(func(f, tx string) { got = append(got, [2]string{f, tx}) })
	if len(got) != 1 || got[0][0] != "grey" {
		t.Errorf("RegisterStatusFunc should deliver current state immediately, got %v", got)
	}

	_ = c.Stop()
	fill, text = c.Status()
	if fill != "grey" || text != "stopped" {
		t.Errorf("status after Stop = (%q, %q), want (grey, stopped)", fill, text)
	}
}

func TestI2CBusConfig_TransferWithoutHandle(t *testing.T) {
	inst, _ := NewI2CBusConfig(flow.ConfigNode{ID: "b1", Config: map[string]any{"bus": float64(1)}})
	c := inst.(*I2CBusConfig)

	// Start() was never called (no OrbitOS client in this test binary
	// anyway), so there's no handle — Transfer must fail loudly rather
	// than panic on a nil handle.
	if _, err := c.Transfer(0x48, nil, 2); err == nil {
		t.Fatal("expected error transferring on a bus with no open handle, got nil")
	}
}

func TestI2CBusConfig_StartTolerantOfNoOrbitClient(t *testing.T) {
	inst, _ := NewI2CBusConfig(flow.ConfigNode{ID: "b1", Config: map[string]any{"bus": float64(1)}})
	c := inst.(*I2CBusConfig)
	if err := c.Start(); err != nil {
		t.Fatalf("Start should tolerate no OrbitOS connection (mirrors ModbusServer), got error: %v", err)
	}
	fill, _ := c.Status()
	if fill != "red" {
		t.Errorf("status fill = %q, want red after a failed connect attempt", fill)
	}
}
