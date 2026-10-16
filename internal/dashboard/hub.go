// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

// Package dashboard keeps the state of the dashboard page: its pages,
// groups and widgets (registered by the ui-* nodes while they run), the
// last value of each widget and the points of each chart. Changes go to
// the browsers in one "dashboard" WebSocket event every 100 ms, so a
// widget fed many times a second does not flood the connection.
package dashboard

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// flushEvery is how often pending changes are sent to the browsers.
const flushEvery = 100 * time.Millisecond

// forgetAfter is how long the value of a widget that is no longer running
// is kept: long enough for a deploy to restart it without losing it.
const forgetAfter = 30 * time.Second

// Page is a tab of the dashboard.
type Page struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Order int    `json:"order"`
}

// Group is a titled section of a page, Width columns wide out of 12.
type Group struct {
	ID    string `json:"id"`
	Page  string `json:"page"`
	Name  string `json:"name"`
	Width int    `json:"width"`
	Order int    `json:"order"`
}

// Widget is one control or display, placed in a group.
type Widget struct {
	ID    string         `json:"id"`
	Type  string         `json:"type"`
	Group string         `json:"group"`
	Label string         `json:"label"`
	Width int            `json:"width"`
	Order int            `json:"order"`
	Props map[string]any `json:"props"`

	// OnAction receives what the user did (switch, slider, button); nil
	// for widgets that only show values.
	OnAction func(value any) error `json:"-"`
}

// Point is one value of a chart series.
type Point struct {
	T int64   `json:"t"` // Unix milliseconds
	V float64 `json:"v"`
}

// chart keeps the recent points of each series (msg.topic) of a chart.
type chart struct {
	keepFor   time.Duration
	maxPoints int
	series    map[string][]Point
}

// WidgetState is a widget as the dashboard page receives it.
type WidgetState struct {
	Widget
	Value  any                `json:"value,omitempty"`
	Series map[string][]Point `json:"series,omitempty"`
}

// Layout is everything the dashboard page needs to draw itself.
type Layout struct {
	Pages   []Page        `json:"pages"`
	Groups  []Group       `json:"groups"`
	Widgets []WidgetState `json:"widgets"`
}

// Update is the content of a "dashboard" WebSocket event.
type Update struct {
	Layout bool                          `json:"layout,omitempty"` // fetch the layout again
	Values map[string]any                `json:"values,omitempty"`
	Points map[string]map[string][]Point `json:"points,omitempty"` // widget → series → new points
	Clears []string                      `json:"clears,omitempty"` // charts emptied
}

// Hub is the dashboard state. Use Default().
type Hub struct {
	mu        sync.Mutex
	pages     map[string]Page
	groups    map[string]Group
	widgets   map[string]*Widget
	values    map[string]any
	charts    map[string]*chart
	removedAt map[string]time.Time

	pending   Update
	dirty     bool
	broadcast func(Update)
	once      sync.Once
	now       func() time.Time
}

var defaultHub = New()

// Default returns the dashboard of the app.
func Default() *Hub { return defaultHub }

// New returns an empty dashboard (tests use their own).
func New() *Hub {
	return &Hub{
		pages:     map[string]Page{},
		groups:    map[string]Group{},
		widgets:   map[string]*Widget{},
		values:    map[string]any{},
		charts:    map[string]*chart{},
		removedAt: map[string]time.Time{},
		now:       time.Now,
	}
}

// SetBroadcast sets how updates reach the browsers and starts sending them.
func (h *Hub) SetBroadcast(fn func(Update)) {
	h.mu.Lock()
	h.broadcast = fn
	h.mu.Unlock()
	h.once.Do(func() {
		go func() {
			for range time.Tick(flushEvery) {
				h.Flush()
			}
		}()
	})
}

// Flush sends the pending changes now.
func (h *Hub) Flush() {
	h.mu.Lock()
	h.forgetRemoved()
	if !h.dirty || h.broadcast == nil {
		h.mu.Unlock()
		return
	}
	u := h.pending
	h.pending = Update{}
	h.dirty = false
	fn := h.broadcast
	h.mu.Unlock()
	fn(u)
}

// forgetRemoved drops the state of widgets gone for longer than
// forgetAfter. Caller holds mu.
func (h *Hub) forgetRemoved() {
	for id, at := range h.removedAt {
		if h.now().Sub(at) < forgetAfter {
			continue
		}
		delete(h.removedAt, id)
		delete(h.values, id)
		delete(h.charts, id)
	}
}

func (h *Hub) layoutChanged() {
	h.pending.Layout = true
	h.dirty = true
}

// SetPage adds or replaces a page.
func (h *Hub) SetPage(p Page) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.pages[p.ID] = p
	h.layoutChanged()
}

// RemovePage removes a page.
func (h *Hub) RemovePage(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.pages, id)
	h.layoutChanged()
}

// SetGroup adds or replaces a group.
func (h *Hub) SetGroup(g Group) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.groups[g.ID] = g
	h.layoutChanged()
}

// RemoveGroup removes a group.
func (h *Hub) RemoveGroup(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.groups, id)
	h.layoutChanged()
}

// AddWidget registers a running widget. A chart's points are kept for
// keepFor and at most maxPoints per series (0: the defaults, 1 h and 1000).
func (h *Hub) AddWidget(w *Widget, keepFor time.Duration, maxPoints int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.widgets[w.ID] = w
	delete(h.removedAt, w.ID)
	if w.Type == "ui-chart" {
		if keepFor <= 0 {
			keepFor = time.Hour
		}
		if maxPoints <= 0 {
			maxPoints = 1000
		}
		c := h.charts[w.ID]
		if c == nil {
			c = &chart{series: map[string][]Point{}}
			h.charts[w.ID] = c
		}
		c.keepFor, c.maxPoints = keepFor, maxPoints
	}
	h.layoutChanged()
}

// RemoveWidget unregisters a widget; its value is kept for a while in case
// it is started again by a deploy.
func (h *Hub) RemoveWidget(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.widgets, id)
	h.removedAt[id] = h.now()
	h.layoutChanged()
}

// SetValue sets the value a widget shows.
func (h *Hub) SetValue(id string, v any) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.values[id] = v
	if h.pending.Values == nil {
		h.pending.Values = map[string]any{}
	}
	h.pending.Values[id] = v
	h.dirty = true
}

// Value returns the value a widget shows.
func (h *Hub) Value(id string) (any, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	v, ok := h.values[id]
	return v, ok
}

// AddPoint adds a value to a chart series.
func (h *Hub) AddPoint(id, series string, v float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	c := h.charts[id]
	if c == nil {
		return
	}
	p := Point{T: h.now().UnixMilli(), V: v}
	pts := append(c.series[series], p)
	cutoff := p.T - c.keepFor.Milliseconds()
	start := 0
	for start < len(pts) && pts[start].T < cutoff {
		start++
	}
	if len(pts)-start > c.maxPoints {
		start = len(pts) - c.maxPoints
	}
	if start > 0 {
		pts = append([]Point(nil), pts[start:]...)
	}
	c.series[series] = pts

	if h.pending.Points == nil {
		h.pending.Points = map[string]map[string][]Point{}
	}
	if h.pending.Points[id] == nil {
		h.pending.Points[id] = map[string][]Point{}
	}
	h.pending.Points[id][series] = append(h.pending.Points[id][series], p)
	h.dirty = true
}

// ClearChart empties a chart.
func (h *Hub) ClearChart(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if c := h.charts[id]; c != nil {
		c.series = map[string][]Point{}
	}
	if h.pending.Points != nil {
		delete(h.pending.Points, id)
	}
	h.pending.Clears = append(h.pending.Clears, id)
	h.dirty = true
}

// Action passes what the user did on a widget to its node.
func (h *Hub) Action(id string, value any) error {
	h.mu.Lock()
	w := h.widgets[id]
	h.mu.Unlock()
	if w == nil {
		return fmt.Errorf("no widget %q on the dashboard", id)
	}
	if w.OnAction == nil {
		return fmt.Errorf("widget %q does not take input", id)
	}
	return w.OnAction(value)
}

// Layout returns the pages, groups and running widgets, in order, with
// their values.
func (h *Hub) Layout() Layout {
	h.mu.Lock()
	defer h.mu.Unlock()
	l := Layout{Pages: []Page{}, Groups: []Group{}, Widgets: []WidgetState{}}
	for _, p := range h.pages {
		l.Pages = append(l.Pages, p)
	}
	for _, g := range h.groups {
		l.Groups = append(l.Groups, g)
	}
	for _, w := range h.widgets {
		ws := WidgetState{Widget: *w, Value: h.values[w.ID]}
		if c := h.charts[w.ID]; c != nil {
			ws.Series = map[string][]Point{}
			for k, pts := range c.series {
				ws.Series[k] = append([]Point(nil), pts...)
			}
		}
		l.Widgets = append(l.Widgets, ws)
	}
	sort.Slice(l.Pages, func(i, j int) bool {
		return less(l.Pages[i].Order, l.Pages[j].Order, l.Pages[i].Name, l.Pages[j].Name)
	})
	sort.Slice(l.Groups, func(i, j int) bool {
		return less(l.Groups[i].Order, l.Groups[j].Order, l.Groups[i].Name, l.Groups[j].Name)
	})
	sort.Slice(l.Widgets, func(i, j int) bool {
		return less(l.Widgets[i].Order, l.Widgets[j].Order, l.Widgets[i].Label, l.Widgets[j].Label)
	})
	return l
}

func less(oa, ob int, na, nb string) bool {
	if oa != ob {
		return oa < ob
	}
	return na < nb
}
