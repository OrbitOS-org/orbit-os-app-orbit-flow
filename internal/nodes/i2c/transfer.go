// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package i2c

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/flow"
	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/nodes"
)

// I2CTransferNode performs a write, read, or write-then-read on a
// configured I2C Bus at a given 7-bit address. Address and read length can
// be overridden per-message; write bytes come from the configured static
// value if set, otherwise from msg.payload — same "config is a default,
// msg wins" convention used by mqtt-out's topic.
//
// Implements flow.ConfigProvider.
type I2CTransferNode struct {
	cfg flow.NodeConfig
	nodes.BaseNode
	configLookup flow.ConfigLookupFunc

	busID string
	bus   *I2CBusConfig

	address    uint32
	writeData  []byte // nil means "use msg.payload"
	readLength uint32
}

// NewI2CTransferNode is the NodeFactory for the i2c-transfer node type.
func NewI2CTransferNode(cfg flow.NodeConfig) (flow.NodeInstance, error) {
	return &I2CTransferNode{cfg: cfg}, nil
}

// Init validates that an I2C Bus config is referenced and parses the
// address, optional static write bytes, and default read length.
func (n *I2CTransferNode) Init() error {
	props := n.cfg.Properties

	n.busID, _ = props["bus"].(string)
	if n.busID == "" {
		return fmt.Errorf("i2c-transfer %s: no I2C Bus configured", n.cfg.ID)
	}

	addr := nodes.IntVal(props, "address", -1)
	if addr < 0 || addr > 0x7F {
		return fmt.Errorf("i2c-transfer %s: address must be between 0x00 and 0x7F", n.cfg.ID)
	}
	n.address = uint32(addr)

	n.readLength = uint32(nodes.IntVal(props, "readLength", 0))

	if s, _ := props["writeData"].(string); strings.TrimSpace(s) != "" {
		data, err := nodes.ParseHexBytes(s)
		if err != nil {
			return fmt.Errorf("i2c-transfer %s: writeData: %w", n.cfg.ID, err)
		}
		n.writeData = data
	}

	return nil
}

// SetConfigLookup implements flow.ConfigProvider.
func (n *I2CTransferNode) SetConfigLookup(fn flow.ConfigLookupFunc) { n.configLookup = fn }

// Start resolves the referenced I2C Bus config.
func (n *I2CTransferNode) Start() error {
	bus, err := nodes.ResolveConfigInstance[I2CBusConfig](n.configLookup, n.busID, n.Status, nodes.ResolveConfigParams{
		NodeKind:   "i2c-transfer",
		NodeID:     n.cfg.ID,
		ConfigKind: "bus",
		TypeLabel:  "an I2C Bus",
	})
	if err != nil {
		return err
	}
	n.bus = bus
	slog.Info("i2c-transfer started", "node_id", n.cfg.ID, "address", fmt.Sprintf("0x%02X", n.address))
	return nil
}

// Stop is a no-op — no held resources.
func (n *I2CTransferNode) Stop() error { return nil }

// HandleMessage resolves address/readLength/write-data (config, with
// msg.address / msg.readLength overrides) and performs the transfer.
// readLength == 0 means write-only, matching I2CBus.Transfer's own
// convention — the message is forwarded unchanged in that case, with
// msg.payload replaced by the read bytes otherwise.
func (n *I2CTransferNode) HandleMessage(msg *flow.Message) ([][]*flow.Message, error) {
	if msg == nil {
		return nil, nil
	}

	address := n.address
	if v, ok := msg.Get("address").(float64); ok {
		address = uint32(v)
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
			return nil, fmt.Errorf("i2c-transfer: %w", err)
		}
	}

	data, err := n.bus.Transfer(address, writeData, readLength)
	if err != nil {
		n.statusError(err.Error())
		return nil, fmt.Errorf("i2c-transfer: %w", err)
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

func (n *I2CTransferNode) statusError(label string) {
	if n.Status != nil {
		n.Status("red", label)
	}
}

// I2CTransferTypeInfo returns the node type metadata for the palette.
func I2CTransferTypeInfo() flow.NodeTypeInfo {
	return flow.NodeTypeInfo{
		Type:        "i2c-transfer",
		Category:    "hardware",
		Label:       "I2C Transfer",
		Description: "Writes and/or reads bytes on a configured I2C bus",
		Icon:        "i2c-transfer",
		Defaults: map[string]any{
			"bus":        "",
			"address":    0,
			"writeData":  "",
			"readLength": 0,
		},
		Inputs:  1,
		Outputs: 1,
	}
}
