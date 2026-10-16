// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package gpio

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"

	orbitclient "github.com/OrbitOS-org/orbit-os-sdk-go/v26/client"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/flow"
	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/nodes"
	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/orbitsdk"
)

// GpioPinConfig is a config node that reserves a single physical GPIO line
// and its direction, shared by any number of GPIO Read / GPIO Write nodes
// referencing it — mirroring ModbusServer's role for modbus-read/write.
//
// Direction is a property of the physical pin, not of an individual flow
// node: without a shared owner, a Read node and a Write node both pointing
// at the same line would fight over SetDirection on every deploy. Centering
// ownership here means it's set once, deliberately, and Read/Write nodes
// simply check it matches what they expect (see read.go / write.go).
//
// Unlike ModbusServer, there is no persistent connection to hold: GpioManager
// calls are stateless (each identifies the pin directly), so this type is
// mostly bookkeeping — the pin identity, the direction, and status
// propagation to whichever nodes reference it.
//
// Implements flow.ConfigInstance.
type GpioPinConfig struct {
	id string

	pin       *orbitclient.GpioPin
	direction orbitclient.GpioDirection

	mu          sync.Mutex
	statusFuncs []flow.StatusFunc
	currentFill string
	currentText string
}

// NewGpioPinConfig parses a ConfigNode definition. The actual SetDirection
// call is deferred to Start(), matching ModbusServer's "connect happens at
// deploy" convention.
func NewGpioPinConfig(cfg flow.ConfigNode) (flow.ConfigInstance, error) {
	props := cfg.Config

	line := nodes.IntVal(props, "line", -1)
	if line < 0 {
		return nil, fmt.Errorf("gpio-pin %s: pin is required", cfg.ID)
	}
	chip := nodes.IntVal(props, "chip", 0)
	name, _ := props["name"].(string)

	dirStr, _ := props["direction"].(string)
	var dir orbitclient.GpioDirection
	switch dirStr {
	case "output":
		dir = orbitclient.GPIO_DIR_OUT
	case "input", "":
		dir = orbitclient.GPIO_DIR_IN
	default:
		return nil, fmt.Errorf("gpio-pin %s: invalid direction %q (expected input|output)", cfg.ID, dirStr)
	}

	return &GpioPinConfig{
		id: cfg.ID,
		pin: &orbitclient.GpioPin{
			Name:       name,
			Number:     int32(line),
			ChipNumber: int32(chip),
		},
		direction:   dir,
		currentFill: "grey",
		currentText: "not started",
	}, nil
}

// Start applies the configured direction to the physical line. A failure
// here (e.g. no OrbitOS connection, or the line doesn't exist on this
// device) does not block deploy — status reflects it, matching
// ModbusServer.Start's "device may be unreachable" tolerance.
func (c *GpioPinConfig) Start() error {
	orbit := orbitsdk.Client()
	if orbit == nil {
		c.setStatus("red", "OrbitOS SDK not connected")
		slog.Warn("gpio-pin start: OrbitOS SDK client not connected", "id", c.id)
		return nil
	}
	err := orbit.GpioManager.SetDirection(c.pin, c.direction)
	// A name the device does not know (often a label typed in the name
	// field, e.g. "Button") makes the runtime look the pin up by that name
	// and fail; the chip and line are always saved, so use them instead.
	if err != nil && c.pin.Name != "" && strings.Contains(err.Error(), "not found") {
		byLine := &orbitclient.GpioPin{Number: c.pin.Number, ChipNumber: c.pin.ChipNumber}
		if err2 := orbit.GpioManager.SetDirection(byLine, c.direction); err2 == nil {
			slog.Info("gpio-pin: name unknown to the device, using chip and line", "id", c.id, "name", c.pin.Name, "chip", c.pin.ChipNumber, "line", c.pin.Number)
			c.pin, err = byLine, nil
		}
	}
	if err != nil {
		c.setStatus("red", err.Error())
		slog.Warn("gpio-pin set direction failed", "id", c.id, "pin", c.PinLabel(), "error", err)
		return nil
	}
	c.setStatus("green", c.directionLabel()+" ready")
	slog.Info("gpio-pin configured", "id", c.id, "pin", c.PinLabel(), "direction", c.directionLabel())
	return nil
}

// Stop is a no-op — no held resources (GpioManager calls are stateless).
func (c *GpioPinConfig) Stop() error {
	c.setStatus("grey", "stopped")
	return nil
}

// Status returns the current connection state.
func (c *GpioPinConfig) Status() (string, string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.currentFill, c.currentText
}

// RegisterStatusFunc allows referencing nodes to receive state changes. The
// current state is delivered immediately.
func (c *GpioPinConfig) RegisterStatusFunc(fn flow.StatusFunc) {
	c.mu.Lock()
	c.statusFuncs = append(c.statusFuncs, fn)
	fill, text := c.currentFill, c.currentText
	c.mu.Unlock()
	fn(fill, text)
}

// Direction returns the configured direction, for Read/Write nodes to
// validate against at Start().
func (c *GpioPinConfig) Direction() orbitclient.GpioDirection { return c.direction }

// PinLabel renders the pin identity for logs/status text: its configured
// name if set, otherwise "chipN/lineM".
func (c *GpioPinConfig) PinLabel() string {
	if c.pin.Name != "" {
		return c.pin.Name
	}
	return fmt.Sprintf("chip%d/line%d", c.pin.ChipNumber, c.pin.Number)
}

// Key names the physical line for the engine's safe-output tracking.
func (c *GpioPinConfig) Key() string {
	return fmt.Sprintf("gpio:%d:%d", c.pin.ChipNumber, c.pin.Number)
}

// Read returns the current level of the configured pin.
func (c *GpioPinConfig) Read() (bool, error) {
	orbit := orbitsdk.Client()
	if orbit == nil {
		return false, fmt.Errorf("OrbitOS SDK client not connected")
	}
	level, err := orbit.GpioManager.GetLevel(c.pin)
	if err != nil {
		return false, err
	}
	return level == orbitclient.GPIO_LEVEL_HIGH, nil
}

// Write drives the configured pin high (true) or low (false).
func (c *GpioPinConfig) Write(high bool) error {
	orbit := orbitsdk.Client()
	if orbit == nil {
		return fmt.Errorf("OrbitOS SDK client not connected")
	}
	level := orbitclient.GPIO_LEVEL_LOW
	if high {
		level = orbitclient.GPIO_LEVEL_HIGH
	}
	return orbit.GpioManager.SetLevel(c.pin, level)
}

func (c *GpioPinConfig) directionLabel() string {
	if c.direction == orbitclient.GPIO_DIR_OUT {
		return "output"
	}
	return "input"
}

func (c *GpioPinConfig) setStatus(fill, text string) {
	c.mu.Lock()
	if c.currentFill == fill && c.currentText == text {
		c.mu.Unlock()
		return
	}
	c.currentFill, c.currentText = fill, text
	fns := append([]flow.StatusFunc(nil), c.statusFuncs...)
	c.mu.Unlock()
	for _, fn := range fns {
		fn(fill, text)
	}
}

// GpioPinConfigTypeInfo returns the config type metadata for the frontend.
func GpioPinConfigTypeInfo() flow.ConfigTypeInfo {
	return flow.ConfigTypeInfo{
		Type:        "gpio-pin",
		Label:       "GPIO Pin",
		Description: "Reserves a physical GPIO line and its direction",
		Defaults: map[string]any{
			"name":      "",
			"chip":      0,
			"line":      -1,
			"direction": "input",
		},
	}
}
