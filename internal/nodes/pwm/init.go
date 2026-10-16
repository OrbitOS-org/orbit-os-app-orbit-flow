// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package pwm

import "github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/nodes"

// init registers the PWM node group.
func init() {
	nodes.RegisterGroup(nodes.Group{
		Name:        "pwm",
		Description: "PWM outputs (Orbit OS device PWM channels)",
		Nodes: []nodes.FlowNodeRegistration{
			{Type: "pwm-out", Factory: NewPwmOutNode, Info: PwmOutTypeInfo()},
		},
	})
}
