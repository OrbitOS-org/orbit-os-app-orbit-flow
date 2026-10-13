// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package orbit

import (
	"testing"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/flow"
)

func TestMobileNotifyResolveBody_EmptyTemplateFallsBackToPayload(t *testing.T) {
	n := &MobileNotifyNode{body: ""}
	msg := flow.NewMessage()
	msg.SetPayload("sensor offline")

	got, err := n.resolveBody(msg)
	if err != nil {
		t.Fatalf("resolveBody: %v", err)
	}
	if got != "sensor offline" {
		t.Errorf("resolveBody = %q, want %q", got, "sensor offline")
	}
}

func TestMobileNotifyResolveBody_RendersTemplateAgainstMessage(t *testing.T) {
	n := &MobileNotifyNode{body: "{{topic}}: {{payload}}"}
	msg := flow.NewMessage()
	msg.SetTopic("plc/alarm")
	msg.SetPayload("high pressure")

	got, err := n.resolveBody(msg)
	if err != nil {
		t.Fatalf("resolveBody: %v", err)
	}
	if want := "plc/alarm: high pressure"; got != want {
		t.Errorf("resolveBody = %q, want %q", got, want)
	}
}

func TestMobileNotifyHandleMessage_NoOrbitClient(t *testing.T) {
	// orbitsdk.SetClient is never called in this package's test binary, so
	// orbitsdk.Client() is nil — mirrors the node running before main()
	// connects, or loopze-edge outside OrbitOS.
	n := &MobileNotifyNode{cfg: flow.NodeConfig{ID: "n1"}, body: "hi"}
	if err := n.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}

	msg := flow.NewMessage()
	if _, err := n.HandleMessage(msg); err == nil {
		t.Fatal("HandleMessage: expected an error with no OrbitOS SDK client connected, got nil")
	}
}

func TestMobileNotifyHandleMessage_NilMessage(t *testing.T) {
	n := &MobileNotifyNode{cfg: flow.NodeConfig{ID: "n1"}}
	out, err := n.HandleMessage(nil)
	if err != nil || out != nil {
		t.Fatalf("HandleMessage(nil) = (%v, %v), want (nil, nil)", out, err)
	}
}
