// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package orbit

import (
	"fmt"
	"log/slog"

	"github.com/cbroglie/mustache"

	orbitclient "github.com/OrbitOS-org/orbit-os-sdk-go/v26/client"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/flow"
	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/nodes"
	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/orbitsdk"
)

// MobileNotifyNode sends a push notification to the OrbitOS device owner's
// phone via MobileNotificationManager.Send — no API keys to manage, since
// the call is forwarded through the Gravity runtime to the Settings app,
// which holds the FCM credentials. The notification title is set by the
// runtime itself ("<device name> · Orbit Flow"); this node only controls the
// body and a few optional delivery hints.
//
// Sink node: 1 input, 1 output (pass-through on success, carrying
// msg.sent/msg.failed — same convention as file-out's status pass-through).
type MobileNotifyNode struct {
	cfg flow.NodeConfig
	nodes.BaseNode

	body        string // mustache template against the message; "" falls back to payload
	imageURL    string
	collapseKey string
	ttlSeconds  int32
}

// NewMobileNotifyNode is the NodeFactory for the mobile-notify node type.
func NewMobileNotifyNode(cfg flow.NodeConfig) (flow.NodeInstance, error) {
	return &MobileNotifyNode{cfg: cfg}, nil
}

// Init parses configuration. No validation beyond type coercion — an empty
// body is only an error once we know msg.payload can't fill it either,
// which HandleMessage checks per-message.
func (n *MobileNotifyNode) Init() error {
	p := n.cfg.Properties
	n.body = nodes.StringVal(p, "body", "")
	n.imageURL = nodes.StringVal(p, "imageUrl", "")
	n.collapseKey = nodes.StringVal(p, "collapseKey", "")
	n.ttlSeconds = int32(nodes.IntVal(p, "ttlSeconds", 0))
	return nil
}

// Start logs deployment.
func (n *MobileNotifyNode) Start() error {
	slog.Info("mobile-notify node started", "node_id", n.cfg.ID)
	return nil
}

// Stop is a no-op — no held resources or background goroutines.
func (n *MobileNotifyNode) Stop() error {
	slog.Info("mobile-notify node stopped", "node_id", n.cfg.ID)
	return nil
}

// HandleMessage renders the body template, sends the notification via the
// OrbitOS SDK, and forwards the message with sent/failed counts attached.
func (n *MobileNotifyNode) HandleMessage(msg *flow.Message) ([][]*flow.Message, error) {
	if msg == nil {
		return nil, nil
	}

	orbit := orbitsdk.Client()
	if orbit == nil {
		n.statusError("no OrbitOS connection")
		return nil, fmt.Errorf("mobile-notify: OrbitOS SDK client not connected")
	}

	body, err := n.resolveBody(msg)
	if err != nil {
		n.statusError("body error")
		return nil, fmt.Errorf("mobile-notify: %w", err)
	}
	if body == "" {
		n.statusError("empty body")
		return nil, fmt.Errorf("mobile-notify: notification body is empty (set Body, or send a non-empty msg.payload)")
	}

	sent, failed, err := orbit.MobileNotificationManager.Send(body, &orbitclient.SendOptions{
		ImageUrl:    n.imageURL,
		CollapseKey: n.collapseKey,
		TtlSeconds:  n.ttlSeconds,
	})
	if err != nil {
		n.statusError(err.Error())
		return nil, fmt.Errorf("mobile-notify: send: %w", err)
	}

	if n.Status != nil {
		n.Status("green", fmt.Sprintf("sent %d, failed %d", sent, failed))
	}

	out := msg.Clone()
	out.Set("sent", sent)
	out.Set("failed", failed)
	return [][]*flow.Message{{out}}, nil
}

// resolveBody renders the configured mustache template against the
// message's data (so {{payload}}, {{topic}}, {{payload.field}}, ... all
// work, matching the templating convention used elsewhere, e.g. file-out's
// path template). An empty configured template falls back to the raw
// stringified payload, so simply wiring a node into mobile-notify with no
// configuration does something sensible.
func (n *MobileNotifyNode) resolveBody(msg *flow.Message) (string, error) {
	if n.body == "" {
		return fmt.Sprintf("%v", msg.Payload()), nil
	}
	return mustache.Render(n.body, msg.DataView())
}

func (n *MobileNotifyNode) statusError(label string) {
	if n.Status != nil {
		n.Status("red", label)
	}
}

// MobileNotifyTypeInfo returns the node type metadata for the palette.
func MobileNotifyTypeInfo() flow.NodeTypeInfo {
	return flow.NodeTypeInfo{
		Type:        "mobile-notify",
		Category:    "orbit",
		Label:       "Mobile Notify",
		Description: "Send a push notification to the OrbitOS device owner's phone",
		Icon:        "mobile-notify",
		Defaults: map[string]any{
			"body":        "",
			"imageUrl":    "",
			"collapseKey": "",
			"ttlSeconds":  0,
		},
		Inputs:  1,
		Outputs: 1,
	}
}
