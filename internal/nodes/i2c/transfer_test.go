// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package i2c

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

func TestI2CTransferNode_Init_RequiresBus(t *testing.T) {
	n := &I2CTransferNode{cfg: flow.NodeConfig{ID: "n1", Properties: map[string]any{"address": float64(0x48)}}}
	if err := n.Init(); err == nil {
		t.Fatal("expected error for missing bus reference, got nil")
	}
}

func TestI2CTransferNode_Init_RequiresValidAddress(t *testing.T) {
	for _, addr := range []float64{-1, 0x80, 200} {
		n := &I2CTransferNode{cfg: flow.NodeConfig{ID: "n1", Properties: map[string]any{"bus": "b1", "address": addr}}}
		if err := n.Init(); err == nil {
			t.Errorf("address %v: expected error, got nil", addr)
		}
	}
}

func TestI2CTransferNode_Init_ParsesWriteData(t *testing.T) {
	n := &I2CTransferNode{cfg: flow.NodeConfig{ID: "n1", Properties: map[string]any{
		"bus": "b1", "address": float64(0x48), "writeData": "0x01 0x02",
	}}}
	if err := n.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if len(n.writeData) != 2 || n.writeData[0] != 0x01 || n.writeData[1] != 0x02 {
		t.Errorf("writeData = % X, want [01 02]", n.writeData)
	}
}

func TestI2CTransferNode_Init_EmptyWriteDataMeansUseMsgPayload(t *testing.T) {
	n := &I2CTransferNode{cfg: flow.NodeConfig{ID: "n1", Properties: map[string]any{"bus": "b1", "address": float64(0x48)}}}
	if err := n.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if n.writeData != nil {
		t.Errorf("writeData = % X, want nil (falls back to msg.payload)", n.writeData)
	}
}

func TestI2CTransferNode_Start_ResolvesBus(t *testing.T) {
	busInst, _ := NewI2CBusConfig(flow.ConfigNode{ID: "b1", Config: map[string]any{"bus": float64(1)}})

	n := &I2CTransferNode{cfg: flow.NodeConfig{ID: "n1"}, busID: "b1"}
	n.SetConfigLookup(lookupReturning("b1", busInst))

	if err := n.Start(); err != nil {
		t.Fatalf("Start: unexpected error: %v", err)
	}
	if n.bus == nil {
		t.Fatal("Start should have resolved n.bus")
	}
}

func TestI2CTransferNode_Start_MissingConfig(t *testing.T) {
	n := &I2CTransferNode{cfg: flow.NodeConfig{ID: "n1"}, busID: "does-not-exist"}
	n.SetConfigLookup(lookupReturning("b1", nil))

	if err := n.Start(); err == nil {
		t.Fatal("expected error when the referenced I2C Bus config doesn't exist, got nil")
	}
}

func TestI2CTransferNode_HandleMessage_NilMessage(t *testing.T) {
	n := &I2CTransferNode{cfg: flow.NodeConfig{ID: "n1"}}
	if out, err := n.HandleMessage(nil); out != nil || err != nil {
		t.Fatalf("HandleMessage(nil) = (%v, %v), want (nil, nil)", out, err)
	}
}

func TestI2CTransferNode_HandleMessage_NoHandleFails(t *testing.T) {
	busInst, _ := NewI2CBusConfig(flow.ConfigNode{ID: "b1", Config: map[string]any{"bus": float64(1)}})
	n := &I2CTransferNode{
		cfg:       flow.NodeConfig{ID: "n1"},
		bus:       busInst.(*I2CBusConfig),
		address:   0x48,
		writeData: []byte{0x01},
	}
	if _, err := n.HandleMessage(flow.NewMessage()); err == nil {
		t.Fatal("expected error transferring on a bus with no open handle, got nil")
	}
}
