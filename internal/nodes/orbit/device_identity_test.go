// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package orbit

import (
	"testing"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/flow"
)

func TestDeviceIdentityNode_Fetch_NoOrbitClient(t *testing.T) {
	n := &DeviceIdentityNode{cfg: flow.NodeConfig{ID: "n1"}}
	if _, err := n.fetch(); err == nil {
		t.Fatal("expected error with no OrbitOS SDK client connected, got nil")
	}
	if n.cached != nil {
		t.Error("cached should stay nil after a failed fetch")
	}
}

func TestDeviceIdentityNode_Start_DoesNotFailDeployOnPrefetchFailure(t *testing.T) {
	n := &DeviceIdentityNode{cfg: flow.NodeConfig{ID: "n1"}}
	if err := n.Start(); err != nil {
		t.Fatalf("Start should tolerate a prefetch failure (mirrors ModbusServer), got error: %v", err)
	}
}

func TestDeviceIdentityNode_HandleMessage_NoOrbitClient(t *testing.T) {
	n := &DeviceIdentityNode{cfg: flow.NodeConfig{ID: "n1"}}
	if _, err := n.HandleMessage(flow.NewMessage()); err == nil {
		t.Fatal("expected error with no OrbitOS SDK client connected and no cache, got nil")
	}
}

func TestDeviceIdentityNode_HandleMessage_NilMessage(t *testing.T) {
	n := &DeviceIdentityNode{cfg: flow.NodeConfig{ID: "n1"}}
	if out, err := n.HandleMessage(nil); out != nil || err != nil {
		t.Fatalf("HandleMessage(nil) = (%v, %v), want (nil, nil)", out, err)
	}
}

func TestDeviceIdentityNode_HandleMessage_UsesCacheWithoutRefetching(t *testing.T) {
	n := &DeviceIdentityNode{
		cfg:    flow.NodeConfig{ID: "n1"},
		cached: map[string]any{"deviceName": "test-device"},
	}
	out, err := n.HandleMessage(flow.NewMessage())
	if err != nil {
		t.Fatalf("HandleMessage: unexpected error with a pre-populated cache: %v", err)
	}
	msgs := out[0]
	if len(msgs) != 1 {
		t.Fatalf("expected exactly 1 output message, got %d", len(msgs))
	}
	payload, ok := msgs[0].Payload().(map[string]any)
	if !ok || payload["deviceName"] != "test-device" {
		t.Errorf("payload = %v, want the cached identity map", msgs[0].Payload())
	}
}
