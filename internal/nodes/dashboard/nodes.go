// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

// Package dashboard holds the dashboard nodes, named and behaving like
// Node-RED Dashboard 2.0: ui-gauge, ui-chart and ui-text show what they
// receive; ui-switch, ui-slider and ui-button send a message when someone
// uses them; ui-page and ui-group are where the widgets go.
package dashboard

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/dashboard"
	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/flow"
	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/nodes"
)

// hub is the dashboard the nodes register with; tests replace it.
var hub = dashboard.Default()

// ---- pages and groups -------------------------------------------------------

// PageConfig is a tab of the dashboard (config node ui-page).
type PageConfig struct{ page dashboard.Page }

// NewPageConfig parses a ui-page config node.
func NewPageConfig(cfg flow.ConfigNode) (flow.ConfigInstance, error) {
	name := cfg.Name
	if name == "" {
		name = nodes.StringVal(cfg.Config, "name", "Home")
	}
	return &PageConfig{page: dashboard.Page{ID: cfg.ID, Name: name, Order: nodes.IntVal(cfg.Config, "order", 0)}}, nil
}

func (c *PageConfig) Start() error             { hub.SetPage(c.page); return nil }
func (c *PageConfig) Stop() error              { hub.RemovePage(c.page.ID); return nil }
func (c *PageConfig) Status() (string, string) { return "green", "" }

// GroupConfig is a section of a page (config node ui-group).
type GroupConfig struct{ group dashboard.Group }

// NewGroupConfig parses a ui-group config node.
func NewGroupConfig(cfg flow.ConfigNode) (flow.ConfigInstance, error) {
	name := cfg.Name
	if name == "" {
		name = nodes.StringVal(cfg.Config, "name", "Group")
	}
	width := nodes.IntVal(cfg.Config, "width", 12)
	if width < 1 || width > 12 {
		width = 12
	}
	return &GroupConfig{group: dashboard.Group{
		ID: cfg.ID, Name: name, Page: nodes.StringVal(cfg.Config, "page", ""),
		Width: width, Order: nodes.IntVal(cfg.Config, "order", 0),
	}}, nil
}

func (c *GroupConfig) Start() error             { hub.SetGroup(c.group); return nil }
func (c *GroupConfig) Stop() error              { hub.RemoveGroup(c.group.ID); return nil }
func (c *GroupConfig) Status() (string, string) { return "green", "" }

// ---- widgets ----------------------------------------------------------------

// defaultLabel is the title of a widget whose node has no name.
var defaultLabel = map[string]string{
	"ui-gauge": "Gauge", "ui-chart": "Chart", "ui-text": "Text",
	"ui-switch": "Switch", "ui-slider": "Slider", "ui-button": "Button",
}

// widget is what every ui-* node shares: its place on the dashboard.
type widget struct {
	nodes.BaseNode
	cfg flow.NodeConfig
	w   *dashboard.Widget
}

func newWidget(cfg flow.NodeConfig, typ string, width int, props map[string]any) widget {
	label := cfg.Name
	if label == "" {
		label = nodes.StringVal(cfg.Properties, "label", "")
	}
	if label == "" {
		label = defaultLabel[typ]
	}
	wd := nodes.IntVal(cfg.Properties, "width", width)
	if wd < 1 || wd > 12 {
		wd = width
	}
	return widget{cfg: cfg, w: &dashboard.Widget{
		ID: cfg.ID, Type: typ, Label: label, Width: wd,
		Group: nodes.StringVal(cfg.Properties, "group", ""),
		Order: nodes.IntVal(cfg.Properties, "order", 0),
		Props: props,
	}}
}

func (b *widget) Init() error { return nil }
func (b *widget) Stop() error { hub.RemoveWidget(b.w.ID); return nil }

// GaugeNode (ui-gauge) shows a number on a dial.
type GaugeNode struct{ widget }

// NewGaugeNode is the NodeFactory for ui-gauge.
func NewGaugeNode(cfg flow.NodeConfig) (flow.NodeInstance, error) {
	p := cfg.Properties
	return &GaugeNode{newWidget(cfg, "ui-gauge", 4, map[string]any{
		"min":   nodes.FloatVal(p, "min", 0),
		"max":   nodes.FloatVal(p, "max", 100),
		"units": nodes.StringVal(p, "units", ""),
	})}, nil
}

func (n *GaugeNode) Start() error { hub.AddWidget(n.w, 0, 0); return nil }

// HandleMessage shows msg.payload.
func (n *GaugeNode) HandleMessage(msg *flow.Message) ([][]*flow.Message, error) {
	v, ok := number(msg.Payload())
	if !ok {
		return nil, fmt.Errorf("ui-gauge: msg.payload is not a number: %v", msg.Payload())
	}
	hub.SetValue(n.w.ID, v)
	return nil, nil
}

// ChartNode (ui-chart) draws the values it receives over time, one line
// (or set of bars) per msg.topic. msg.payload = [] empties it.
type ChartNode struct {
	widget
	keepFor   time.Duration
	maxPoints int
}

// NewChartNode is the NodeFactory for ui-chart.
func NewChartNode(cfg flow.NodeConfig) (flow.NodeInstance, error) {
	p := cfg.Properties
	n := &ChartNode{widget: newWidget(cfg, "ui-chart", 8, map[string]any{
		"chartType": nodes.StringVal(p, "chartType", "line"),
		"units":     nodes.StringVal(p, "units", ""),
	})}
	keep := nodes.FloatVal(p, "removeOlder", 1)
	n.keepFor = time.Duration(keep * float64(unitOf(nodes.StringVal(p, "removeOlderUnit", "3600"))))
	n.maxPoints = nodes.IntVal(p, "removeOlderPoints", 1000)
	n.w.Props["keepSeconds"] = n.keepFor.Seconds()
	return n, nil
}

// unitOf reads Node-RED's removeOlderUnit (seconds per unit, as text).
func unitOf(s string) time.Duration {
	sec, err := strconv.ParseFloat(s, 64)
	if err != nil || sec <= 0 {
		sec = 3600
	}
	return time.Duration(sec * float64(time.Second))
}

func (n *ChartNode) Start() error { hub.AddWidget(n.w, n.keepFor, n.maxPoints); return nil }

// HandleMessage adds msg.payload to the series named by msg.topic.
func (n *ChartNode) HandleMessage(msg *flow.Message) ([][]*flow.Message, error) {
	if arr, ok := msg.Payload().([]any); ok && len(arr) == 0 {
		hub.ClearChart(n.w.ID)
		return nil, nil
	}
	v, ok := number(msg.Payload())
	if !ok {
		return nil, fmt.Errorf("ui-chart: msg.payload is not a number: %v", msg.Payload())
	}
	series := msg.Topic()
	if series == "" {
		series = n.w.Label
	}
	hub.AddPoint(n.w.ID, series, v)
	return nil, nil
}

// TextNode (ui-text) shows msg.payload as text.
type TextNode struct{ widget }

// NewTextNode is the NodeFactory for ui-text.
func NewTextNode(cfg flow.NodeConfig) (flow.NodeInstance, error) {
	return &TextNode{newWidget(cfg, "ui-text", 4, map[string]any{
		"units": nodes.StringVal(cfg.Properties, "units", ""),
	})}, nil
}

func (n *TextNode) Start() error { hub.AddWidget(n.w, 0, 0); return nil }

// HandleMessage shows msg.payload.
func (n *TextNode) HandleMessage(msg *flow.Message) ([][]*flow.Message, error) {
	hub.SetValue(n.w.ID, text(msg.Payload()))
	return nil, nil
}

// SwitchNode (ui-switch) is an on/off switch: using it sends the on or
// off value; a message received sets its position (and is passed on when
// "pass through" is on).
type SwitchNode struct {
	widget
	on, off  any
	topic    string
	passthru bool
}

// NewSwitchNode is the NodeFactory for ui-switch.
func NewSwitchNode(cfg flow.NodeConfig) (flow.NodeInstance, error) {
	p := cfg.Properties
	on, err := typed(nodes.StringVal(p, "onvalueType", "bool"), nodes.StringVal(p, "onvalue", "true"))
	if err != nil {
		return nil, fmt.Errorf("ui-switch: on value: %w", err)
	}
	off, err := typed(nodes.StringVal(p, "offvalueType", "bool"), nodes.StringVal(p, "offvalue", "false"))
	if err != nil {
		return nil, fmt.Errorf("ui-switch: off value: %w", err)
	}
	return &SwitchNode{
		widget: newWidget(cfg, "ui-switch", 4, map[string]any{}),
		on:     on, off: off,
		topic:    nodes.StringVal(p, "topic", ""),
		passthru: nodes.BoolVal(p, "passthru", false),
	}, nil
}

func (n *SwitchNode) Start() error {
	n.w.OnAction = n.action
	hub.AddWidget(n.w, 0, 0)
	return nil
}

func (n *SwitchNode) action(v any) error {
	state, _ := v.(bool)
	hub.SetValue(n.w.ID, state)
	out := flow.NewMessage()
	if state {
		out.SetPayload(n.on)
	} else {
		out.SetPayload(n.off)
	}
	if n.topic != "" {
		out.SetTopic(n.topic)
	}
	if n.Send != nil {
		n.Send(0, out)
	}
	return nil
}

// HandleMessage sets the switch from msg.payload (the on value, or true).
func (n *SwitchNode) HandleMessage(msg *flow.Message) ([][]*flow.Message, error) {
	p := msg.Payload()
	state := sameValue(p, n.on) || (!sameValue(p, n.off) && p == true)
	hub.SetValue(n.w.ID, state)
	if n.passthru {
		return [][]*flow.Message{{msg}}, nil
	}
	return nil, nil
}

// SliderNode (ui-slider) sends its value when someone moves it; a message
// received sets it.
type SliderNode struct {
	widget
	topic    string
	passthru bool
}

// NewSliderNode is the NodeFactory for ui-slider.
func NewSliderNode(cfg flow.NodeConfig) (flow.NodeInstance, error) {
	p := cfg.Properties
	return &SliderNode{
		widget: newWidget(cfg, "ui-slider", 4, map[string]any{
			"min":  nodes.FloatVal(p, "min", 0),
			"max":  nodes.FloatVal(p, "max", 100),
			"step": nodes.FloatVal(p, "step", 1),
		}),
		topic:    nodes.StringVal(p, "topic", ""),
		passthru: nodes.BoolVal(p, "passthru", false),
	}, nil
}

func (n *SliderNode) Start() error {
	n.w.OnAction = n.action
	hub.AddWidget(n.w, 0, 0)
	return nil
}

func (n *SliderNode) action(v any) error {
	f, ok := number(v)
	if !ok {
		return fmt.Errorf("slider value is not a number")
	}
	hub.SetValue(n.w.ID, f)
	out := flow.NewMessage()
	out.SetPayload(f)
	if n.topic != "" {
		out.SetTopic(n.topic)
	}
	if n.Send != nil {
		n.Send(0, out)
	}
	return nil
}

// HandleMessage sets the slider from msg.payload.
func (n *SliderNode) HandleMessage(msg *flow.Message) ([][]*flow.Message, error) {
	f, ok := number(msg.Payload())
	if !ok {
		return nil, fmt.Errorf("ui-slider: msg.payload is not a number: %v", msg.Payload())
	}
	hub.SetValue(n.w.ID, f)
	if n.passthru {
		return [][]*flow.Message{{msg}}, nil
	}
	return nil, nil
}

// ButtonNode (ui-button) sends its message when pressed; a message
// received presses it.
type ButtonNode struct {
	widget
	payloadType string
	payload     string
	topic       string
}

// NewButtonNode is the NodeFactory for ui-button.
func NewButtonNode(cfg flow.NodeConfig) (flow.NodeInstance, error) {
	p := cfg.Properties
	n := &ButtonNode{
		widget:      newWidget(cfg, "ui-button", 4, map[string]any{}),
		payloadType: nodes.StringVal(p, "payloadType", "str"),
		payload:     nodes.StringVal(p, "payload", ""),
		topic:       nodes.StringVal(p, "topic", ""),
	}
	if _, err := typed(n.payloadType, n.payload); err != nil {
		return nil, fmt.Errorf("ui-button: payload: %w", err)
	}
	return n, nil
}

func (n *ButtonNode) Start() error {
	n.w.OnAction = func(any) error { n.press(nil); return nil }
	hub.AddWidget(n.w, 0, 0)
	return nil
}

func (n *ButtonNode) press(in *flow.Message) {
	out := flow.NewMessage()
	if in != nil {
		out = in.Clone()
	}
	v, _ := typed(n.payloadType, n.payload) // a date is taken at press time
	out.SetPayload(v)
	if n.topic != "" {
		out.SetTopic(n.topic)
	}
	if n.Send != nil {
		n.Send(0, out)
	}
}

// HandleMessage presses the button.
func (n *ButtonNode) HandleMessage(msg *flow.Message) ([][]*flow.Message, error) {
	n.press(msg)
	return nil, nil
}

// ---- helpers ----------------------------------------------------------------

func typed(typ, value string) (any, error) {
	switch typ {
	case "str", "num", "bool", "json", "date":
		return nodes.ResolveValue(typ, value, "", nil, nodes.ValueContext{})
	}
	return nil, fmt.Errorf("unknown value type %q", typ)
}

func number(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case int32:
		return float64(x), true
	case uint8:
		return float64(x), true
	case bool:
		if x {
			return 1, true
		}
		return 0, true
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
		return f, err == nil
	}
	return 0, false
}

func text(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case []byte:
		return string(x)
	case map[string]any, []any:
		b, err := json.Marshal(x)
		if err == nil {
			return string(b)
		}
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	}
	return fmt.Sprint(v)
}

func sameValue(a, b any) bool {
	ja, errA := json.Marshal(a)
	jb, errB := json.Marshal(b)
	return errA == nil && errB == nil && string(ja) == string(jb)
}
