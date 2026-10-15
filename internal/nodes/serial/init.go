// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

// Package serial holds the Serial In / Serial Out / Serial Request nodes and
// their shared Serial Port config node, named and behaving like Node-RED's
// serial nodes. They reach the device's serial ports (on-board UARTs and
// USB-serial adapters) through the Orbit OS UART API, so they only work
// when this app runs on Orbit OS.
package serial

import "github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/nodes"

func init() {
	nodes.RegisterGroup(nodes.Group{
		Name:        "serial",
		Description: "Serial ports (UART, RS-232, RS-485, USB-serial)",
		ConfigNodes: []nodes.ConfigNodeRegistration{
			{Type: "serial-port", Factory: NewSerialPortConfig, Info: SerialPortConfigTypeInfo()},
		},
		Nodes: []nodes.FlowNodeRegistration{
			{Type: "serial-in", Factory: NewSerialInNode, Info: SerialInTypeInfo()},
			{Type: "serial-out", Factory: NewSerialOutNode, Info: SerialOutTypeInfo()},
			{Type: "serial-request", Factory: NewSerialRequestNode, Info: SerialRequestTypeInfo()},
		},
	})
}
