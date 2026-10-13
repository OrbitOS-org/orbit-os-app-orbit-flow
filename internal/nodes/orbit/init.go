// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

// Package orbit holds flow nodes that only make sense on OrbitOS — they
// call an OrbitOS SDK manager via internal/orbitsdk rather than doing
// generic network/filesystem I/O, so they have no equivalent when
// loopze-edge runs outside the platform.
package orbit

import "github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/nodes"

func init() {
	nodes.RegisterGroup(nodes.Group{
		Name:        "orbit",
		Description: "OrbitOS platform nodes (mobile notifications, ...)",
		Nodes: []nodes.FlowNodeRegistration{
			{Type: "mobile-notify", Factory: NewMobileNotifyNode, Info: MobileNotifyTypeInfo()},
			{Type: "system-metrics", Factory: NewSystemMetricsNode, Info: SystemMetricsTypeInfo()},
			{Type: "device-identity", Factory: NewDeviceIdentityNode, Info: DeviceIdentityTypeInfo()},
		},
	})
}
