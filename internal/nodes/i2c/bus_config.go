// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package i2c

import (
	"fmt"
	"log/slog"
	"sync"

	orbitclient "github.com/OrbitOS-org/orbit-os-sdk-go/v26/client"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/flow"
	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/nodes"
	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/orbitsdk"
)

// I2CBusConfig is a config node that configures a physical I2C bus and
// serializes transfers on it, shared by any number of I2C Transfer nodes —
// mirroring ModbusServer's role for modbus-read/write and GpioPinConfig's
// for gpio-read/write. I2C is a genuinely shared bus (unlike a GPIO pin's
// direction, there's no "conflict" to detect here — every peripheral has
// its own address — but concurrent transfers from two nodes must still not
// interleave on the wire, hence transferMu).
//
// Implements flow.ConfigInstance.
type I2CBusConfig struct {
	id string

	bus             uint32
	clockHz         uint32
	tenBitAddr      bool
	clockStretching bool

	mu          sync.Mutex // protects handle + status fields
	handle      *orbitclient.I2CBus
	statusFuncs []flow.StatusFunc
	currentFill string
	currentText string

	transferMu sync.Mutex // serializes wire access
}

// NewI2CBusConfig parses a ConfigNode definition. The actual Open() call is
// deferred to Start(), matching ModbusServer's "connect happens at deploy"
// convention.
func NewI2CBusConfig(cfg flow.ConfigNode) (flow.ConfigInstance, error) {
	props := cfg.Config

	bus := nodes.IntVal(props, "bus", -1)
	if bus < 0 {
		return nil, fmt.Errorf("i2c-bus %s: bus is required", cfg.ID)
	}
	clockHz := nodes.IntVal(props, "clockHz", 100_000)
	if clockHz <= 0 {
		return nil, fmt.Errorf("i2c-bus %s: clockHz must be positive", cfg.ID)
	}

	return &I2CBusConfig{
		id:              cfg.ID,
		bus:             uint32(bus),
		clockHz:         uint32(clockHz),
		tenBitAddr:      nodes.BoolVal(props, "tenBitAddr", false),
		clockStretching: nodes.BoolVal(props, "clockStretching", false),
		currentFill:     "grey",
		currentText:     "not started",
	}, nil
}

// Start opens the bus. A failure does not block deploy — the device may be
// temporarily unreachable at deploy time, matching ModbusServer.Start.
func (c *I2CBusConfig) Start() error {
	orbit := orbitsdk.Client()
	if orbit == nil {
		c.setStatus("red", "OrbitOS SDK not connected")
		slog.Warn("i2c-bus start: OrbitOS SDK client not connected", "id", c.id)
		return nil
	}

	handle, err := orbit.I2CManager.Open(c.bus, c.clockHz, c.tenBitAddr, c.clockStretching)
	if err != nil {
		c.setStatus("red", err.Error())
		slog.Warn("i2c-bus open failed", "id", c.id, "bus", c.bus, "error", err)
		return nil
	}

	c.mu.Lock()
	c.handle = handle
	c.mu.Unlock()

	c.setStatus("green", fmt.Sprintf("bus %d ready", c.bus))
	slog.Info("i2c-bus configured", "id", c.id, "bus", c.bus, "clockHz", c.clockHz)
	return nil
}

// Stop releases the handle. I2CBus has no Close() of its own — SetI2CConfig
// only applies transfer parameters, it doesn't hold a persistent per-client
// lock on the Gravity side — so this just forgets the handle.
func (c *I2CBusConfig) Stop() error {
	c.mu.Lock()
	c.handle = nil
	c.mu.Unlock()
	c.setStatus("grey", "stopped")
	return nil
}

// Status returns the current connection state.
func (c *I2CBusConfig) Status() (string, string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.currentFill, c.currentText
}

// RegisterStatusFunc allows referencing nodes to receive state changes. The
// current state is delivered immediately.
func (c *I2CBusConfig) RegisterStatusFunc(fn flow.StatusFunc) {
	c.mu.Lock()
	c.statusFuncs = append(c.statusFuncs, fn)
	fill, text := c.currentFill, c.currentText
	c.mu.Unlock()
	fn(fill, text)
}

// Transfer performs a write/read/write-then-read on the given 7-bit address,
// serialized against every other Transfer call on this bus.
func (c *I2CBusConfig) Transfer(addr uint32, data []byte, readLen uint32) ([]byte, error) {
	c.mu.Lock()
	handle := c.handle
	c.mu.Unlock()
	if handle == nil {
		return nil, fmt.Errorf("I2C bus %d not connected", c.bus)
	}

	c.transferMu.Lock()
	defer c.transferMu.Unlock()

	result, err := handle.Transfer(addr, data, readLen, 0)
	if err != nil {
		c.setStatus("red", err.Error())
		return nil, err
	}
	c.setStatus("green", fmt.Sprintf("bus %d ready", c.bus))
	return result, nil
}

func (c *I2CBusConfig) setStatus(fill, text string) {
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

// I2CBusConfigTypeInfo returns the config type metadata for the frontend.
func I2CBusConfigTypeInfo() flow.ConfigTypeInfo {
	return flow.ConfigTypeInfo{
		Type:        "i2c-bus",
		Label:       "I2C Bus",
		Description: "Configures an I2C bus for transfers",
		Defaults: map[string]any{
			"name":            "",
			"bus":             1,
			"clockHz":         100_000,
			"tenBitAddr":      false,
			"clockStretching": false,
		},
	}
}
