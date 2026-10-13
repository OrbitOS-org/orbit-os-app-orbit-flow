// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package spi

import (
	"fmt"
	"log/slog"
	"sync"

	orbitclient "github.com/OrbitOS-org/orbit-os-sdk-go/v26/client"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/flow"
	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/nodes"
	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/orbitsdk"
)

// SpiDeviceConfig is a config node that configures a physical SPI device
// (bus + chip select) and serializes transfers on it, shared by any number
// of SPI Transfer nodes — mirroring I2CBusConfig's role. Unlike I2C, a chip
// select is inherently point-to-point (only ever one device on that
// line), so there's no address to pass per-transfer — the config alone
// identifies the peripheral.
//
// Implements flow.ConfigInstance.
type SpiDeviceConfig struct {
	id string

	bus, cs     uint32
	maxSpeedHz  uint32
	bitsPerWord uint32
	mode        int
	lsbFirst    bool

	mu          sync.Mutex // protects handle + status fields
	handle      *orbitclient.SpiDevice
	statusFuncs []flow.StatusFunc
	currentFill string
	currentText string

	transferMu sync.Mutex // serializes wire access
}

// NewSpiDeviceConfig parses a ConfigNode definition. The actual Open() call
// is deferred to Start(), matching I2CBusConfig's convention.
func NewSpiDeviceConfig(cfg flow.ConfigNode) (flow.ConfigInstance, error) {
	props := cfg.Config

	bus := nodes.IntVal(props, "bus", -1)
	if bus < 0 {
		return nil, fmt.Errorf("spi-device %s: bus is required", cfg.ID)
	}
	cs := nodes.IntVal(props, "chipSelect", -1)
	if cs < 0 {
		return nil, fmt.Errorf("spi-device %s: chipSelect is required", cfg.ID)
	}
	maxSpeedHz := nodes.IntVal(props, "maxSpeedHz", 1_000_000)
	if maxSpeedHz <= 0 {
		return nil, fmt.Errorf("spi-device %s: maxSpeedHz must be positive", cfg.ID)
	}
	bitsPerWord := nodes.IntVal(props, "bitsPerWord", 8)
	mode := nodes.IntVal(props, "mode", 0)
	if mode < 0 || mode > 3 {
		return nil, fmt.Errorf("spi-device %s: mode must be 0-3", cfg.ID)
	}

	return &SpiDeviceConfig{
		id:          cfg.ID,
		bus:         uint32(bus),
		cs:          uint32(cs),
		maxSpeedHz:  uint32(maxSpeedHz),
		bitsPerWord: uint32(bitsPerWord),
		mode:        mode,
		lsbFirst:    nodes.BoolVal(props, "lsbFirst", false),
		currentFill: "grey",
		currentText: "not started",
	}, nil
}

// Start opens the device. A failure does not block deploy — the device may
// be temporarily unreachable at deploy time, matching I2CBusConfig.Start.
func (c *SpiDeviceConfig) Start() error {
	orbit := orbitsdk.Client()
	if orbit == nil {
		c.setStatus("red", "OrbitOS SDK not connected")
		slog.Warn("spi-device start: OrbitOS SDK client not connected", "id", c.id)
		return nil
	}

	handle, err := orbit.SpiManager.Open(c.bus, c.cs, c.maxSpeedHz, c.bitsPerWord, c.mode, c.lsbFirst)
	if err != nil {
		c.setStatus("red", err.Error())
		slog.Warn("spi-device open failed", "id", c.id, "bus", c.bus, "cs", c.cs, "error", err)
		return nil
	}

	c.mu.Lock()
	c.handle = handle
	c.mu.Unlock()

	c.setStatus("green", fmt.Sprintf("spidev%d.%d ready", c.bus, c.cs))
	slog.Info("spi-device configured", "id", c.id, "bus", c.bus, "cs", c.cs, "maxSpeedHz", c.maxSpeedHz)
	return nil
}

// Stop releases the handle. SpiDevice has no Close() of its own —
// SetSpiConfig only applies transfer parameters — so this just forgets the
// handle.
func (c *SpiDeviceConfig) Stop() error {
	c.mu.Lock()
	c.handle = nil
	c.mu.Unlock()
	c.setStatus("grey", "stopped")
	return nil
}

// Status returns the current connection state.
func (c *SpiDeviceConfig) Status() (string, string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.currentFill, c.currentText
}

// RegisterStatusFunc allows referencing nodes to receive state changes. The
// current state is delivered immediately.
func (c *SpiDeviceConfig) RegisterStatusFunc(fn flow.StatusFunc) {
	c.mu.Lock()
	c.statusFuncs = append(c.statusFuncs, fn)
	fill, text := c.currentFill, c.currentText
	c.mu.Unlock()
	fn(fill, text)
}

// Transfer performs a full-duplex transfer, serialized against every other
// Transfer call on this device.
func (c *SpiDeviceConfig) Transfer(dataOut []byte, readLength uint32) ([]byte, error) {
	c.mu.Lock()
	handle := c.handle
	c.mu.Unlock()
	if handle == nil {
		return nil, fmt.Errorf("SPI device spidev%d.%d not connected", c.bus, c.cs)
	}

	c.transferMu.Lock()
	defer c.transferMu.Unlock()

	result, err := handle.Transfer(dataOut, readLength)
	if err != nil {
		c.setStatus("red", err.Error())
		return nil, err
	}
	c.setStatus("green", fmt.Sprintf("spidev%d.%d ready", c.bus, c.cs))
	return result, nil
}

func (c *SpiDeviceConfig) setStatus(fill, text string) {
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

// SpiDeviceConfigTypeInfo returns the config type metadata for the frontend.
func SpiDeviceConfigTypeInfo() flow.ConfigTypeInfo {
	return flow.ConfigTypeInfo{
		Type:        "spi-device",
		Label:       "SPI Device",
		Description: "Configures an SPI device (bus + chip select) for transfers",
		Defaults: map[string]any{
			"name":        "",
			"bus":         0,
			"chipSelect":  0,
			"maxSpeedHz":  1_000_000,
			"bitsPerWord": 8,
			"mode":        0,
			"lsbFirst":    false,
		},
	}
}
