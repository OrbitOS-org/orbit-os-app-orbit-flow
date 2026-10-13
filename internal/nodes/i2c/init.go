// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

// Package i2c holds flow nodes for OrbitOS's I2CService — bus-addressed
// read/write transfers with peripherals wired to the device's I2C pins.
// Like package orbit, these only work when this app is actually running on
// OrbitOS.
package i2c

import "github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/nodes"

func init() {
	nodes.RegisterGroup(nodes.Group{
		Name:        "i2c",
		Description: "I2C bus transfers (OrbitOS device as master)",
		ConfigNodes: []nodes.ConfigNodeRegistration{
			{Type: "i2c-bus", Factory: NewI2CBusConfig, Info: I2CBusConfigTypeInfo()},
		},
		Nodes: []nodes.FlowNodeRegistration{
			{Type: "i2c-transfer", Factory: NewI2CTransferNode, Info: I2CTransferTypeInfo()},
		},
	})
}
