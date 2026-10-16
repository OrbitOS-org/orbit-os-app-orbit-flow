// Copyright (C) 2026 Dennis Bleul
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package core

import "github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/nodes"

// init registers the core node group — the protocol-agnostic building
// blocks every flow uses (inject, debug, function, change, switch, …).
// This group is the only one that should not be disabled on a typical
// Orbit Flow installation.
func init() {
	nodes.RegisterGroup(nodes.Group{
		Name:        "core",
		Description: "Core nodes: inject, debug, function, change, switch, link, delay, trigger, filter, range, split, join, batch, template, status, catch, statemachine, json, xml, csv, context-watch",
		Nodes: []nodes.FlowNodeRegistration{
			{Type: "inject", Factory: NewInjectNode, Info: InjectTypeInfo()},
			{Type: "debug", Factory: NewDebugNode, Info: DebugTypeInfo()},
			{Type: "function", Factory: NewFunctionNode, Info: FunctionTypeInfo()},
			{Type: "function-expr", Factory: NewFunctionExprNode, Info: FunctionExprTypeInfo()},
			{Type: "function-go", Factory: NewFunctionGoNode, Info: FunctionGoTypeInfo()},
			{Type: "json", Factory: NewJSONParserNode, Info: JSONParserTypeInfo()},
			{Type: "xml", Factory: NewXMLParserNode, Info: XMLParserTypeInfo()},
			{Type: "csv", Factory: NewCSVParserNode, Info: CSVParserTypeInfo()},
			{Type: "context-watch", Factory: NewContextWatchNode, Info: ContextWatchTypeInfo()},
			{Type: "catch", Factory: NewCatchNode, Info: CatchTypeInfo()},
			{Type: "change", Factory: NewChangeNode, Info: ChangeTypeInfo()},
			{Type: "delay", Factory: NewDelayNode, Info: DelayTypeInfo()},
			{Type: "trigger", Factory: NewTriggerNode, Info: TriggerTypeInfo()},
			{Type: "filter", Factory: NewFilterNode, Info: FilterTypeInfo()},
			{Type: "range", Factory: NewRangeNode, Info: RangeTypeInfo()},
			{Type: "split", Factory: NewSplitNode, Info: SplitTypeInfo()},
			{Type: "join", Factory: NewJoinNode, Info: JoinTypeInfo()},
			{Type: "batch", Factory: NewBatchNode, Info: BatchTypeInfo()},
			{Type: "link-in", Factory: NewLinkInNode, Info: LinkInTypeInfo()},
			{Type: "link-out", Factory: NewLinkOutNode, Info: LinkOutTypeInfo()},
			{Type: "link-call", Factory: NewLinkCallNode, Info: LinkCallTypeInfo()},
			{Type: "statemachine", Factory: NewStateMachineNode, Info: StateMachineTypeInfo()},
			{Type: "status", Factory: NewStatusNode, Info: StatusTypeInfo()},
			{Type: "switch", Factory: NewSwitchNode, Info: SwitchTypeInfo()},
			{Type: "template", Factory: NewTemplateNode, Info: TemplateTypeInfo()},
		},
	})
}
