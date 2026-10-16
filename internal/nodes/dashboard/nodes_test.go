// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package dashboard

import (
	"sync"
	"testing"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/dashboard"
	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/flow"
)

type sink struct {
	mu   sync.Mutex
	msgs []*flow.Message
}

func (s *sink) send(_ int, m *flow.Message) { s.mu.Lock(); s.msgs = append(s.msgs, m); s.mu.Unlock() }

func start(t *testing.T, f flow.NodeFactory, props map[string]any) (flow.NodeInstance, *sink) {
	t.Helper()
	hub = dashboard.New()
	t.Cleanup(func() { hub = dashboard.Default() })
	n, err := f(flow.NodeConfig{ID: "w1", Name: "Light", Properties: props})
	if err != nil {
		t.Fatal(err)
	}
	s := &sink{}
	n.SetSend(s.send)
	n.SetStatus(func(string, string) {})
	n.SetDebug(func(flow.DebugMessage) {})
	if err := n.Init(); err != nil {
		t.Fatal(err)
	}
	if err := n.Start(); err != nil {
		t.Fatal(err)
	}
	return n, s
}

func msg(p any) *flow.Message { m := flow.NewMessage(); m.SetPayload(p); return m }

func TestSwitch_ActionAndInput(t *testing.T) {
	n, s := start(t, NewSwitchNode, map[string]any{"onvalue": "ON", "onvalueType": "str", "offvalue": "OFF", "offvalueType": "str", "topic": "light"})
	if err := hub.Action("w1", true); err != nil {
		t.Fatal(err)
	}
	if len(s.msgs) != 1 || s.msgs[0].Payload() != "ON" || s.msgs[0].Topic() != "light" {
		t.Fatalf("action sent %v", s.msgs)
	}
	out, _ := n.HandleMessage(msg("OFF"))
	if v, _ := hub.Value("w1"); v != false || out != nil {
		t.Fatalf("input: state %v, passed on %v", v, out)
	}
	_, _ = n.HandleMessage(msg("ON"))
	if v, _ := hub.Value("w1"); v != true {
		t.Fatalf("input on: %v", v)
	}
}

func TestSlider_ButtonGaugeChart(t *testing.T) {
	_, s := start(t, NewSliderNode, map[string]any{"min": 0.0, "max": 100.0})
	_ = hub.Action("w1", 40.0)
	if len(s.msgs) != 1 || s.msgs[0].Payload() != 40.0 {
		t.Fatalf("slider sent %v", s.msgs)
	}

	b, s := start(t, NewButtonNode, map[string]any{"payload": "run", "payloadType": "str", "topic": "chase"})
	_ = hub.Action("w1", nil)
	_, _ = b.HandleMessage(msg("x"))
	if len(s.msgs) != 2 || s.msgs[0].Payload() != "run" || s.msgs[1].Payload() != "run" || s.msgs[1].Topic() != "chase" {
		t.Fatalf("button sent %v", s.msgs)
	}

	g, _ := start(t, NewGaugeNode, nil)
	if _, err := g.HandleMessage(msg("21.5")); err != nil {
		t.Fatal(err)
	}
	if v, _ := hub.Value("w1"); v != 21.5 {
		t.Fatalf("gauge: %v", v)
	}
	if _, err := g.HandleMessage(msg("hot")); err == nil {
		t.Fatal("want an error for a gauge value that is not a number")
	}

	c, _ := start(t, NewChartNode, nil)
	m := msg(410.0)
	m.SetTopic("power")
	_, _ = c.HandleMessage(m)
	_, _ = c.HandleMessage(msg(3.0)) // no topic: the series is the label
	series := hub.Layout().Widgets[0].Series
	if len(series["power"]) != 1 || len(series["Light"]) != 1 {
		t.Fatalf("chart series: %v", series)
	}
	_, _ = c.HandleMessage(msg([]any{}))
	if len(hub.Layout().Widgets[0].Series) != 0 {
		t.Fatal("msg.payload [] should clear the chart")
	}
}

func TestWidget_StopRemovesIt(t *testing.T) {
	n, _ := start(t, NewTextNode, map[string]any{"group": "g1", "width": 6.0})
	l := hub.Layout()
	if len(l.Widgets) != 1 || l.Widgets[0].Group != "g1" || l.Widgets[0].Width != 6 || l.Widgets[0].Label != "Light" {
		t.Fatalf("layout %+v", l.Widgets)
	}
	_ = n.Stop()
	if len(hub.Layout().Widgets) != 0 {
		t.Fatal("stopped widget still on the dashboard")
	}
}
