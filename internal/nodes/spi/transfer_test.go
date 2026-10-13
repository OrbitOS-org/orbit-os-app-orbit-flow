// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package spi

import (
	"testing"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/flow"
)

func lookupReturning(id string, inst flow.ConfigInstance) flow.ConfigLookupFunc {
	return func(configID string) (flow.ConfigInstance, bool) {
		if configID == id {
			return inst, true
		}
		return nil, false
	}
}

func TestSpiTransferNode_Init_RequiresDevice(t *testing.T) {
	n := &SpiTransferNode{cfg: flow.NodeConfig{ID: "n1", Properties: map[string]any{}}}
	if err := n.Init(); err == nil {
		t.Fatal("expected error for missing device reference, got nil")
	}
}

func TestSpiTransferNode_Init_ParsesWriteData(t *testing.T) {
	n := &SpiTransferNode{cfg: flow.NodeConfig{ID: "n1", Properties: map[string]any{
		"device": "d1", "writeData": "0x9F 0x00 0x00",
	}}}
	if err := n.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if len(n.writeData) != 3 || n.writeData[0] != 0x9F {
		t.Errorf("writeData = % X, want [9F 00 00]", n.writeData)
	}
}

func TestSpiTransferNode_Init_EmptyWriteDataMeansUseMsgPayload(t *testing.T) {
	n := &SpiTransferNode{cfg: flow.NodeConfig{ID: "n1", Properties: map[string]any{"device": "d1"}}}
	if err := n.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if n.writeData != nil {
		t.Errorf("writeData = % X, want nil (falls back to msg.payload)", n.writeData)
	}
}

func TestSpiTransferNode_Start_ResolvesDevice(t *testing.T) {
	deviceInst, _ := NewSpiDeviceConfig(flow.ConfigNode{ID: "d1", Config: map[string]any{"bus": float64(0), "chipSelect": float64(0)}})

	n := &SpiTransferNode{cfg: flow.NodeConfig{ID: "n1"}, deviceID: "d1"}
	n.SetConfigLookup(lookupReturning("d1", deviceInst))

	if err := n.Start(); err != nil {
		t.Fatalf("Start: unexpected error: %v", err)
	}
	if n.device == nil {
		t.Fatal("Start should have resolved n.device")
	}
}

func TestSpiTransferNode_Start_MissingConfig(t *testing.T) {
	n := &SpiTransferNode{cfg: flow.NodeConfig{ID: "n1"}, deviceID: "does-not-exist"}
	n.SetConfigLookup(lookupReturning("d1", nil))

	if err := n.Start(); err == nil {
		t.Fatal("expected error when the referenced SPI Device config doesn't exist, got nil")
	}
}

func TestSpiTransferNode_HandleMessage_NilMessage(t *testing.T) {
	n := &SpiTransferNode{cfg: flow.NodeConfig{ID: "n1"}}
	if out, err := n.HandleMessage(nil); out != nil || err != nil {
		t.Fatalf("HandleMessage(nil) = (%v, %v), want (nil, nil)", out, err)
	}
}

func TestSpiTransferNode_HandleMessage_NoHandleFails(t *testing.T) {
	deviceInst, _ := NewSpiDeviceConfig(flow.ConfigNode{ID: "d1", Config: map[string]any{"bus": float64(0), "chipSelect": float64(0)}})
	n := &SpiTransferNode{
		cfg:       flow.NodeConfig{ID: "n1"},
		device:    deviceInst.(*SpiDeviceConfig),
		writeData: []byte{0x01},
	}
	if _, err := n.HandleMessage(flow.NewMessage()); err == nil {
		t.Fatal("expected error transferring on a device with no open handle, got nil")
	}
}
