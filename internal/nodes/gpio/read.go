// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package gpio

import (
	"fmt"
	"log/slog"

	orbitclient "github.com/OrbitOS-org/orbit-os-sdk-go/v26/client"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/flow"
	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/nodes"
)

// GpioReadNode reads the current level of a configured GPIO Pin on each
// incoming message and forwards it as msg.payload (bool). It does not poll
// on its own — GpioManager has no change-notification mechanism, so
// reacting to a level change means wiring an Inject (interval) or another
// trigger into this node's input.
//
// Implements flow.ConfigProvider.
type GpioReadNode struct {
	cfg flow.NodeConfig
	nodes.BaseNode
	configLookup flow.ConfigLookupFunc

	pinID string
	pin   *GpioPinConfig
}

// NewGpioReadNode is the NodeFactory for the gpio-read node type.
func NewGpioReadNode(cfg flow.NodeConfig) (flow.NodeInstance, error) {
	return &GpioReadNode{cfg: cfg}, nil
}

// Init validates that a GPIO Pin config is referenced.
func (n *GpioReadNode) Init() error {
	n.pinID, _ = n.cfg.Properties["pin"].(string)
	if n.pinID == "" {
		return fmt.Errorf("gpio-read %s: no GPIO Pin configured", n.cfg.ID)
	}
	return nil
}

// SetConfigLookup implements flow.ConfigProvider.
func (n *GpioReadNode) SetConfigLookup(fn flow.ConfigLookupFunc) { n.configLookup = fn }

// Start resolves the referenced GPIO Pin config and checks it was
// configured as an input — reading an output-direction pin is almost
// certainly a wiring mistake in the flow, caught here rather than silently
// returning whatever GetLevel happens to report.
func (n *GpioReadNode) Start() error {
	pin, err := nodes.ResolveConfigInstance[GpioPinConfig](n.configLookup, n.pinID, n.Status, nodes.ResolveConfigParams{
		NodeKind:   "gpio-read",
		NodeID:     n.cfg.ID,
		ConfigKind: "pin",
		TypeLabel:  "a GPIO Pin",
	})
	if err != nil {
		return err
	}
	n.pin = pin
	if n.pin.Direction() != orbitclient.GPIO_DIR_IN {
		n.statusError("pin is configured as output")
		return fmt.Errorf("gpio-read %s: pin %q is configured as output, not input", n.cfg.ID, n.pin.PinLabel())
	}
	slog.Info("gpio-read started", "node_id", n.cfg.ID, "pin", n.pin.PinLabel())
	return nil
}

// Stop is a no-op — no held resources.
func (n *GpioReadNode) Stop() error { return nil }

// HandleMessage reads the pin and forwards the level as msg.payload.
func (n *GpioReadNode) HandleMessage(msg *flow.Message) ([][]*flow.Message, error) {
	if msg == nil {
		return nil, nil
	}

	level, err := n.pin.Read()
	if err != nil {
		n.statusError(err.Error())
		return nil, fmt.Errorf("gpio-read: %w", err)
	}

	if n.Status != nil {
		n.Status("green", fmt.Sprintf("%v", level))
	}

	out := msg.Clone()
	out.SetPayload(level)
	return [][]*flow.Message{{out}}, nil
}

func (n *GpioReadNode) statusError(label string) {
	if n.Status != nil {
		n.Status("red", label)
	}
}

// GpioReadTypeInfo returns the node type metadata for the palette.
func GpioReadTypeInfo() flow.NodeTypeInfo {
	return flow.NodeTypeInfo{
		Type:        "gpio-read",
		Category:    "hardware",
		Label:       "GPIO Read",
		Description: "Reads the current level of a configured GPIO pin",
		Icon:        "gpio-read",
		Defaults: map[string]any{
			"pin": "",
		},
		Inputs:  1,
		Outputs: 1,
	}
}
