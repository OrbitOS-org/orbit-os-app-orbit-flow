// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package gpio

import (
	"fmt"
	"log/slog"
	"strings"

	orbitclient "github.com/OrbitOS-org/orbit-os-sdk-go/v26/client"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/flow"
	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/nodes"
)

// gpioAction selects what GpioWriteNode does with an incoming message.
type gpioAction string

const (
	// gpioActionPayload reads the level from msg.payload (via truthy) —
	// the original, still-default behaviour.
	gpioActionPayload gpioAction = "payload"
	// gpioActionOn/Off ignore msg.payload entirely: the message is just a
	// trigger, and the node always drives the same fixed level. This is
	// what lets "GPIO On" → Delay 1s → "GPIO Off" work as two nodes with
	// no Change node in between forcing msg.payload on each one.
	gpioActionOn  gpioAction = "on"
	gpioActionOff gpioAction = "off"
	// gpioActionToggle reads the pin's current level and writes the
	// opposite — the one mode that genuinely can't be derived from
	// msg.payload alone, since it depends on prior state.
	gpioActionToggle gpioAction = "toggle"
)

// GpioWriteNode drives a configured GPIO Pin and forwards the message
// unchanged on success — same pass-through convention as file-out and
// mobile-notify. See gpioAction for how the target level is decided.
//
// Implements flow.ConfigProvider.
type GpioWriteNode struct {
	cfg flow.NodeConfig
	nodes.BaseNode
	configLookup flow.ConfigLookupFunc

	pinID  string
	pin    *GpioPinConfig
	action gpioAction
	onStop string // "off" (default), "on" or "keep": the level when flows stop
}

// NewGpioWriteNode is the NodeFactory for the gpio-write node type.
func NewGpioWriteNode(cfg flow.NodeConfig) (flow.NodeInstance, error) {
	return &GpioWriteNode{cfg: cfg}, nil
}

// Init validates that a GPIO Pin config is referenced and parses the
// action mode.
func (n *GpioWriteNode) Init() error {
	n.pinID, _ = n.cfg.Properties["pin"].(string)
	if n.pinID == "" {
		return fmt.Errorf("gpio-write %s: no GPIO Pin configured", n.cfg.ID)
	}

	action, _ := n.cfg.Properties["action"].(string)
	switch gpioAction(action) {
	case gpioActionOn, gpioActionOff, gpioActionToggle:
		n.action = gpioAction(action)
	case gpioActionPayload, "":
		n.action = gpioActionPayload
	default:
		return fmt.Errorf("gpio-write %s: invalid action %q (expected payload|on|off|toggle)", n.cfg.ID, action)
	}

	n.onStop = nodes.StringVal(n.cfg.Properties, "onStop", "off")
	switch n.onStop {
	case "off", "on", "keep":
	default:
		return fmt.Errorf("gpio-write %s: invalid \"when flows stop\" %q (expected off|on|keep)", n.cfg.ID, n.onStop)
	}
	return nil
}

// OutputKey implements flow.SafeOutput: the pin this node drives.
func (n *GpioWriteNode) OutputKey() string {
	if n.pin == nil {
		return ""
	}
	return n.pin.Key()
}

// ApplySafeState implements flow.SafeOutput: drives the pin to the level
// chosen in "when flows stop" (off by default), or leaves it.
func (n *GpioWriteNode) ApplySafeState() {
	if n.pin == nil || n.onStop == "keep" {
		return
	}
	if err := n.pin.Write(n.onStop == "on"); err != nil {
		slog.Warn("gpio-write: safe state failed", "node_id", n.cfg.ID, "pin", n.pin.PinLabel(), "error", err)
	}
}

// SetConfigLookup implements flow.ConfigProvider.
func (n *GpioWriteNode) SetConfigLookup(fn flow.ConfigLookupFunc) { n.configLookup = fn }

// Start resolves the referenced GPIO Pin config and checks it was
// configured as an output.
func (n *GpioWriteNode) Start() error {
	pin, err := nodes.ResolveConfigInstance[GpioPinConfig](n.configLookup, n.pinID, n.Status, nodes.ResolveConfigParams{
		NodeKind:   "gpio-write",
		NodeID:     n.cfg.ID,
		ConfigKind: "pin",
		TypeLabel:  "a GPIO Pin",
	})
	if err != nil {
		return err
	}
	if pin.Direction() != orbitclient.GPIO_DIR_OUT {
		n.statusError("pin is configured as input")
		return fmt.Errorf("gpio-write %s: pin %q is configured as input, not output", n.cfg.ID, pin.PinLabel())
	}
	// Only an output pin is kept: the safe state never writes to an input.
	n.pin = pin
	slog.Info("gpio-write started", "node_id", n.cfg.ID, "pin", n.pin.PinLabel())
	return nil
}

// Stop is a no-op — no held resources.
func (n *GpioWriteNode) Stop() error { return nil }

// HandleMessage decides the target level per n.action and drives the pin;
// the incoming message is forwarded unchanged on success regardless of
// mode — even in on/off/toggle modes it's still useful downstream as a
// trigger/timing reference (e.g. into a Delay node).
func (n *GpioWriteNode) HandleMessage(msg *flow.Message) ([][]*flow.Message, error) {
	if msg == nil {
		return nil, nil
	}

	high, err := n.resolveLevel(msg)
	if err != nil {
		n.statusError(err.Error())
		return nil, fmt.Errorf("gpio-write: %w", err)
	}

	if err := n.pin.Write(high); err != nil {
		n.statusError(err.Error())
		return nil, fmt.Errorf("gpio-write: %w", err)
	}

	if n.Status != nil {
		label := "LOW"
		if high {
			label = "HIGH"
		}
		n.Status("green", label)
	}

	return [][]*flow.Message{{msg}}, nil
}

// resolveLevel decides the target level per the configured action.
func (n *GpioWriteNode) resolveLevel(msg *flow.Message) (bool, error) {
	switch n.action {
	case gpioActionOn:
		return true, nil
	case gpioActionOff:
		return false, nil
	case gpioActionToggle:
		current, err := n.pin.Read()
		if err != nil {
			return false, fmt.Errorf("toggle: read current level: %w", err)
		}
		return !current, nil
	default: // gpioActionPayload
		return truthy(msg.Payload()), nil
	}
}

func (n *GpioWriteNode) statusError(label string) {
	if n.Status != nil {
		n.Status("red", label)
	}
}

// truthy interprets a message payload as a digital level: booleans and
// numbers use their natural truthiness, strings accept the common on/off
// spellings, anything else (including nil) is false.
func truthy(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case float64:
		return x != 0
	case int:
		return x != 0
	case int64: // Go and expression nodes may give other integer types
		return x != 0
	case int32:
		return x != 0
	case uint16:
		return x != 0
	case uint32:
		return x != 0
	case uint64:
		return x != 0
	case string:
		switch strings.ToLower(strings.TrimSpace(x)) {
		case "1", "true", "on", "high":
			return true
		default:
			return false
		}
	default:
		return false
	}
}

// GpioWriteTypeInfo returns the node type metadata for the palette.
func GpioWriteTypeInfo() flow.NodeTypeInfo {
	return flow.NodeTypeInfo{
		Type:        "gpio-write",
		Category:    "hardware",
		Label:       "GPIO Write",
		Description: "Drives a configured GPIO pin from msg.payload, or a fixed on/off/toggle action",
		Icon:        "gpio-write",
		Defaults: map[string]any{
			"pin":    "",
			"action": string(gpioActionPayload),
			"onStop": "off",
		},
		Inputs:  1,
		Outputs: 1,
	}
}
