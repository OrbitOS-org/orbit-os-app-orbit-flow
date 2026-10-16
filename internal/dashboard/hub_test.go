// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package dashboard

import (
	"testing"
	"time"
)

func TestHub_LayoutOrderAndValues(t *testing.T) {
	h := New()
	h.SetPage(Page{ID: "p2", Name: "Energy", Order: 2})
	h.SetPage(Page{ID: "p1", Name: "Garage", Order: 1})
	h.SetGroup(Group{ID: "g1", Page: "p1", Name: "Status", Width: 12})
	h.AddWidget(&Widget{ID: "b", Type: "ui-text", Group: "g1", Order: 2}, 0, 0)
	h.AddWidget(&Widget{ID: "a", Type: "ui-gauge", Group: "g1", Order: 1}, 0, 0)
	h.SetValue("a", 62.0)

	l := h.Layout()
	if l.Pages[0].ID != "p1" || l.Pages[1].ID != "p2" {
		t.Fatalf("pages order: %v", l.Pages)
	}
	if l.Widgets[0].ID != "a" || l.Widgets[0].Value != 62.0 || l.Widgets[1].ID != "b" {
		t.Fatalf("widgets: %+v", l.Widgets)
	}
}

func TestHub_ValueSurvivesRedeployThenIsForgotten(t *testing.T) {
	h := New()
	now := time.Unix(1000, 0)
	h.now = func() time.Time { return now }
	w := &Widget{ID: "a", Type: "ui-gauge"}
	h.AddWidget(w, 0, 0)
	h.SetValue("a", 5.0)
	h.RemoveWidget("a")
	h.AddWidget(w, 0, 0) // a deploy restarted it
	if v, _ := h.Value("a"); v != 5.0 {
		t.Fatalf("value lost on redeploy: %v", v)
	}
	h.RemoveWidget("a")
	now = now.Add(forgetAfter + time.Second)
	h.Flush()
	if _, ok := h.Value("a"); ok {
		t.Fatal("value of a deleted widget was kept")
	}
}

func TestHub_ChartTrimsAndClears(t *testing.T) {
	h := New()
	now := time.Unix(1000, 0)
	h.now = func() time.Time { return now }
	h.AddWidget(&Widget{ID: "c", Type: "ui-chart"}, 10*time.Second, 3)
	for i := 0; i < 5; i++ {
		h.AddPoint("c", "power", float64(i))
	}
	if pts := h.Layout().Widgets[0].Series["power"]; len(pts) != 3 || pts[0].V != 2 {
		t.Fatalf("max points: %v", pts)
	}
	now = now.Add(11 * time.Second)
	h.AddPoint("c", "power", 9)
	if pts := h.Layout().Widgets[0].Series["power"]; len(pts) != 1 || pts[0].V != 9 {
		t.Fatalf("keep for: %v", pts)
	}
	h.ClearChart("c")
	if len(h.Layout().Widgets[0].Series) != 0 {
		t.Fatal("clear left points")
	}
}

func TestHub_FlushSendsOneUpdateWithTheLatestValues(t *testing.T) {
	h := New()
	var got []Update
	h.mu.Lock()
	h.broadcast = func(u Update) { got = append(got, u) }
	h.mu.Unlock()
	h.AddWidget(&Widget{ID: "a", Type: "ui-gauge"}, 0, 0)
	h.AddWidget(&Widget{ID: "c", Type: "ui-chart"}, 0, 0)
	for i := 0; i < 50; i++ {
		h.SetValue("a", float64(i))
		h.AddPoint("c", "s", float64(i))
	}
	h.Flush()
	h.Flush() // nothing new
	if len(got) != 1 {
		t.Fatalf("want 1 update, got %d", len(got))
	}
	u := got[0]
	if !u.Layout || u.Values["a"] != 49.0 || len(u.Points["c"]["s"]) != 50 {
		t.Fatalf("update: layout %v, value %v, points %d", u.Layout, u.Values["a"], len(u.Points["c"]["s"]))
	}
}

func TestHub_Action(t *testing.T) {
	h := New()
	var got any
	h.AddWidget(&Widget{ID: "s", Type: "ui-switch", OnAction: func(v any) error { got = v; return nil }}, 0, 0)
	h.AddWidget(&Widget{ID: "g", Type: "ui-gauge"}, 0, 0)
	if err := h.Action("s", true); err != nil || got != true {
		t.Fatalf("action: %v, %v", err, got)
	}
	if h.Action("g", 1) == nil || h.Action("nope", 1) == nil {
		t.Fatal("want errors for a display widget and an unknown one")
	}
}
