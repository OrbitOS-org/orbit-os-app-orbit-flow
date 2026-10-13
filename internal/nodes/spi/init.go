// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

// Package spi holds flow nodes for OrbitOS's SpiService — full-duplex
// transfers with peripherals wired to the device's SPI pins. Like package
// orbit, these only work when this app is actually running on OrbitOS.
package spi

import "github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/nodes"

func init() {
	nodes.RegisterGroup(nodes.Group{
		Name:        "spi",
		Description: "SPI transfers (OrbitOS device as master)",
		ConfigNodes: []nodes.ConfigNodeRegistration{
			{Type: "spi-device", Factory: NewSpiDeviceConfig, Info: SpiDeviceConfigTypeInfo()},
		},
		Nodes: []nodes.FlowNodeRegistration{
			{Type: "spi-transfer", Factory: NewSpiTransferNode, Info: SpiTransferTypeInfo()},
		},
	})
}
