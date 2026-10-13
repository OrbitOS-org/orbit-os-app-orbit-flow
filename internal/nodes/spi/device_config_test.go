// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package spi

import (
	"testing"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/flow"
)

func TestNewSpiDeviceConfig_RequiresBus(t *testing.T) {
	_, err := NewSpiDeviceConfig(flow.ConfigNode{
		ID:     "d1",
		Config: map[string]any{"chipSelect": float64(0)},
	})
	if err == nil {
		t.Fatal("expected error for missing bus, got nil")
	}
}

func TestNewSpiDeviceConfig_RequiresChipSelect(t *testing.T) {
	_, err := NewSpiDeviceConfig(flow.ConfigNode{
		ID:     "d1",
		Config: map[string]any{"bus": float64(0)},
	})
	if err == nil {
		t.Fatal("expected error for missing chipSelect, got nil")
	}
}

func TestNewSpiDeviceConfig_RejectsNonPositiveSpeed(t *testing.T) {
	_, err := NewSpiDeviceConfig(flow.ConfigNode{
		ID:     "d1",
		Config: map[string]any{"bus": float64(0), "chipSelect": float64(0), "maxSpeedHz": float64(0)},
	})
	if err == nil {
		t.Fatal("expected error for non-positive maxSpeedHz, got nil")
	}
}

func TestNewSpiDeviceConfig_RejectsInvalidMode(t *testing.T) {
	for _, mode := range []float64{-1, 4, 10} {
		_, err := NewSpiDeviceConfig(flow.ConfigNode{
			ID:     "d1",
			Config: map[string]any{"bus": float64(0), "chipSelect": float64(0), "mode": mode},
		})
		if err == nil {
			t.Errorf("mode %v: expected error, got nil", mode)
		}
	}
}

func TestNewSpiDeviceConfig_Defaults(t *testing.T) {
	inst, err := NewSpiDeviceConfig(flow.ConfigNode{
		ID:     "d1",
		Config: map[string]any{"bus": float64(0), "chipSelect": float64(0)},
	})
	if err != nil {
		t.Fatalf("NewSpiDeviceConfig: %v", err)
	}
	c := inst.(*SpiDeviceConfig)
	if c.maxSpeedHz != 1_000_000 {
		t.Errorf("maxSpeedHz = %d, want default 1000000", c.maxSpeedHz)
	}
	if c.bitsPerWord != 8 {
		t.Errorf("bitsPerWord = %d, want default 8", c.bitsPerWord)
	}
	if c.mode != 0 {
		t.Errorf("mode = %d, want default 0", c.mode)
	}
}

func TestSpiDeviceConfig_StatusLifecycle(t *testing.T) {
	inst, _ := NewSpiDeviceConfig(flow.ConfigNode{ID: "d1", Config: map[string]any{"bus": float64(0), "chipSelect": float64(0)}})
	c := inst.(*SpiDeviceConfig)

	fill, text := c.Status()
	if fill != "grey" || text != "not started" {
		t.Errorf("initial status = (%q, %q), want (grey, not started)", fill, text)
	}

	_ = c.Stop()
	fill, text = c.Status()
	if fill != "grey" || text != "stopped" {
		t.Errorf("status after Stop = (%q, %q), want (grey, stopped)", fill, text)
	}
}

func TestSpiDeviceConfig_TransferWithoutHandle(t *testing.T) {
	inst, _ := NewSpiDeviceConfig(flow.ConfigNode{ID: "d1", Config: map[string]any{"bus": float64(0), "chipSelect": float64(0)}})
	c := inst.(*SpiDeviceConfig)
	if _, err := c.Transfer([]byte{0x01}, 1); err == nil {
		t.Fatal("expected error transferring on a device with no open handle, got nil")
	}
}
