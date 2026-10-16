// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package dashboard

import (
	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/flow"
	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/nodes"
)

// init registers the dashboard node group.
func init() {
	nodes.RegisterGroup(nodes.Group{
		Name:        "dashboard",
		Description: "Dashboard widgets: gauge, chart, text, switch, slider, button",
		ConfigNodes: []nodes.ConfigNodeRegistration{
			{Type: "ui-page", Factory: NewPageConfig, Info: flow.ConfigTypeInfo{
				Type: "ui-page", Label: "Dashboard Page", Description: "A tab of the dashboard",
				Defaults: map[string]any{"order": 0},
			}},
			{Type: "ui-group", Factory: NewGroupConfig, Info: flow.ConfigTypeInfo{
				Type: "ui-group", Label: "Dashboard Group", Description: "A section of a dashboard page",
				Defaults: map[string]any{"page": "", "width": 12, "order": 0},
			}},
		},
		Nodes: []nodes.FlowNodeRegistration{
			{Type: "ui-gauge", Factory: NewGaugeNode, Info: info("ui-gauge", "Gauge", "Shows a number on a dial", 1, 0,
				map[string]any{"min": 0.0, "max": 100.0, "units": ""}, "dial", "meter")},
			{Type: "ui-chart", Factory: NewChartNode, Info: info("ui-chart", "Chart", "Draws values over time, one line per msg.topic", 1, 0,
				map[string]any{"chartType": "line", "units": "", "removeOlder": 1.0, "removeOlderUnit": "3600", "removeOlderPoints": 1000.0}, "graph", "plot", "trend")},
			{Type: "ui-text", Factory: NewTextNode, Info: info("ui-text", "Text", "Shows msg.payload as text", 1, 0,
				map[string]any{"units": ""}, "label", "value")},
			{Type: "ui-switch", Factory: NewSwitchNode, Info: info("ui-switch", "Switch", "An on/off switch on the dashboard", 1, 1,
				map[string]any{"onvalue": "true", "onvalueType": "bool", "offvalue": "false", "offvalueType": "bool", "topic": "", "passthru": false}, "toggle")},
			{Type: "ui-slider", Factory: NewSliderNode, Info: info("ui-slider", "Slider", "A slider that sends a number", 1, 1,
				map[string]any{"min": 0.0, "max": 100.0, "step": 1.0, "topic": "", "passthru": false}, "range", "dimmer")},
			{Type: "ui-button", Factory: NewButtonNode, Info: info("ui-button", "Button", "A button that sends a message", 1, 1,
				map[string]any{"payload": "", "payloadType": "str", "topic": ""}, "press")},
		},
	})
}

func info(typ, label, desc string, in, out int, defaults map[string]any, keywords ...string) flow.NodeTypeInfo {
	d := map[string]any{"group": "", "width": 0.0, "order": 0.0, "label": ""}
	for k, v := range defaults {
		d[k] = v
	}
	return flow.NodeTypeInfo{
		Type: typ, Category: "dashboard", Label: label, Description: desc, Icon: typ,
		Keywords: append([]string{"dashboard", "ui"}, keywords...),
		Defaults: d, Inputs: in, Outputs: out,
	}
}
