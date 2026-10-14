// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package spi

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/flow"
	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/nodes"
)

// SpiTransferNode performs a full-duplex transfer on a configured SPI
// Device. Read length can be overridden per-message; write bytes come from
// the configured static value if set, otherwise from msg.payload — same
// convention as i2c-transfer. Unlike I2C there's no address parameter: the
// chip select already identifies the one peripheral on that line.
//
// Implements flow.ConfigProvider.
type SpiTransferNode struct {
	cfg flow.NodeConfig
	nodes.BaseNode
	configLookup flow.ConfigLookupFunc

	deviceID string
	device   *SpiDeviceConfig

	writeData  []byte // nil means "use msg.payload"
	readLength uint32
}

// NewSpiTransferNode is the NodeFactory for the spi-transfer node type.
func NewSpiTransferNode(cfg flow.NodeConfig) (flow.NodeInstance, error) {
	return &SpiTransferNode{cfg: cfg}, nil
}

// Init validates that an SPI Device config is referenced and parses the
// optional static write bytes and default read length.
func (n *SpiTransferNode) Init() error {
	props := n.cfg.Properties

	n.deviceID, _ = props["device"].(string)
	if n.deviceID == "" {
		return fmt.Errorf("spi-transfer %s: no SPI Device configured", n.cfg.ID)
	}

	n.readLength = uint32(nodes.IntVal(props, "readLength", 0))

	if s, _ := props["writeData"].(string); strings.TrimSpace(s) != "" {
		data, err := nodes.ParseHexBytes(s)
		if err != nil {
			return fmt.Errorf("spi-transfer %s: writeData: %w", n.cfg.ID, err)
		}
		n.writeData = data
	}

	return nil
}

// SetConfigLookup implements flow.ConfigProvider.
func (n *SpiTransferNode) SetConfigLookup(fn flow.ConfigLookupFunc) { n.configLookup = fn }

// Start resolves the referenced SPI Device config.
func (n *SpiTransferNode) Start() error {
	device, err := nodes.ResolveConfigInstance[SpiDeviceConfig](n.configLookup, n.deviceID, n.Status, nodes.ResolveConfigParams{
		NodeKind:   "spi-transfer",
		NodeID:     n.cfg.ID,
		ConfigKind: "device",
		TypeLabel:  "an SPI Device",
	})
	if err != nil {
		return err
	}
	n.device = device
	slog.Info("spi-transfer started", "node_id", n.cfg.ID)
	return nil
}

// Stop is a no-op — no held resources.
func (n *SpiTransferNode) Stop() error { return nil }

// HandleMessage resolves readLength/write-data (config, with
// msg.readLength override) and performs the transfer. readLength == 0
// means write-only, matching SpiDevice.Transfer's own convention — the
// message is forwarded unchanged in that case, with msg.payload replaced
// by the read bytes otherwise.
func (n *SpiTransferNode) HandleMessage(msg *flow.Message) ([][]*flow.Message, error) {
	if msg == nil {
		return nil, nil
	}

	readLength := n.readLength
	if v, ok := msg.Get("readLength").(float64); ok {
		readLength = uint32(v)
	}

	writeData := n.writeData
	if writeData == nil {
		var err error
		writeData, err = nodes.PayloadToBytes(msg.Payload())
		if err != nil {
			n.statusError("payload error")
			return nil, fmt.Errorf("spi-transfer: %w", err)
		}
	}

	data, err := n.device.Transfer(writeData, readLength)
	if err != nil {
		n.statusError(err.Error())
		return nil, fmt.Errorf("spi-transfer: %w", err)
	}

	if n.Status != nil {
		n.Status("green", fmt.Sprintf("%d bytes", len(data)))
	}

	out := msg.Clone()
	if readLength > 0 {
		out.SetPayload(nodes.BytesToIntArray(data))
	}
	return [][]*flow.Message{{out}}, nil
}

func (n *SpiTransferNode) statusError(label string) {
	if n.Status != nil {
		n.Status("red", label)
	}
}

// SpiTransferTypeInfo returns the node type metadata for the palette.
func SpiTransferTypeInfo() flow.NodeTypeInfo {
	return flow.NodeTypeInfo{
		Type:        "spi-transfer",
		Category:    "hardware",
		Label:       "SPI Transfer",
		Description: "Writes and/or reads bytes on a configured SPI device (full-duplex)",
		Icon:        "spi-transfer",
		Defaults: map[string]any{
			"device":     "",
			"writeData":  "",
			"readLength": 0,
		},
		Inputs:  1,
		Outputs: 1,
	}
}
