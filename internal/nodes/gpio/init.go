// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

// Package gpio holds flow nodes for OrbitOS's GpioService — reading and
// driving physical digital I/O lines on the device. Like package orbit,
// these only work when this app is actually running on OrbitOS.
package gpio

import "github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/nodes"

func init() {
	nodes.RegisterGroup(nodes.Group{
		Name:        "gpio",
		Description: "GPIO digital I/O (OrbitOS device pins)",
		ConfigNodes: []nodes.ConfigNodeRegistration{
			{Type: "gpio-pin", Factory: NewGpioPinConfig, Info: GpioPinConfigTypeInfo()},
		},
		Nodes: []nodes.FlowNodeRegistration{
			{Type: "gpio-read", Factory: NewGpioReadNode, Info: GpioReadTypeInfo()},
			{Type: "gpio-write", Factory: NewGpioWriteNode, Info: GpioWriteTypeInfo()},
		},
	})
}
