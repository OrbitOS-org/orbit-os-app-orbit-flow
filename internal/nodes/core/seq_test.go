// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package core

import (
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/flow"
)

// portCollector records what a node sends asynchronously, with the port.
type portCollector struct {
	mu    sync.Mutex
	ports []int
	msgs  []*flow.Message
}

func (c *portCollector) send(port int, msg *flow.Message) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ports = append(c.ports, port)
	c.msgs = append(c.msgs, msg)
}

func (c *portCollector) snapshot() ([]int, []*flow.Message) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]int(nil), c.ports...), append([]*flow.Message(nil), c.msgs...)
}

func startNode(t *testing.T, factory flow.NodeFactory, typ string, props map[string]any) (flow.NodeInstance, *portCollector) {
	t.Helper()
	n, err := factory(flow.NodeConfig{ID: t.Name(), Type: typ, Properties: props})
	if err != nil {
		t.Fatal(err)
	}
	if err := n.Init(); err != nil {
		t.Fatal(err)
	}
	c := &portCollector{}
	n.SetSend(c.send)
	n.SetStatus(noopStatus)
	n.SetDebug(noopDebug)
	if err := n.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = n.Stop() })
	return n, c
}

// feed sends each message and returns the payloads that came out of
// HandleMessage directly (port 0).
func feed(t *testing.T, n flow.NodeInstance, msgs ...*flow.Message) []*flow.Message {
	t.Helper()
	var out []*flow.Message
	for _, m := range msgs {
		res, err := n.HandleMessage(m)
		if err != nil {
			t.Fatal(err)
		}
		if len(res) > 0 {
			out = append(out, res[0]...)
		}
	}
	return out
}

func payloads(msgs []*flow.Message) []any {
	out := make([]any, len(msgs))
	for i, m := range msgs {
		out[i] = m.Payload()
	}
	return out
}

func eventually(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not met in time")
}

// ---- filter -----------------------------------------------------------------

func TestFilter_PassesOnlyChanges(t *testing.T) {
	n, _ := startNode(t, NewFilterNode, "filter", map[string]any{"func": "rbe"})
	in := []any{0.0, 0.0, 1.0, 1.0, 0.0, map[string]any{"a": 1.0}, map[string]any{"a": 1.0}}
	var msgs []*flow.Message
	for _, v := range in {
		msgs = append(msgs, makeMsg(v))
	}
	got := payloads(feed(t, n, msgs...))
	want := []any{0.0, 1.0, 0.0, map[string]any{"a": 1.0}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestFilter_IgnoreInitialAndTopics(t *testing.T) {
	n, _ := startNode(t, NewFilterNode, "filter", map[string]any{"func": "rbei"})
	got := payloads(feed(t, n,
		makeMsg(1.0, map[string]any{"topic": "a"}),
		makeMsg(1.0, map[string]any{"topic": "b"}),
		makeMsg(2.0, map[string]any{"topic": "a"}),
		makeMsg(1.0, map[string]any{"topic": "b"}),
	))
	if !reflect.DeepEqual(got, []any{2.0}) {
		t.Fatalf("got %v, want [2]", got)
	}
}

func TestFilter_Reset(t *testing.T) {
	n, _ := startNode(t, NewFilterNode, "filter", map[string]any{"func": "rbe"})
	got := payloads(feed(t, n, makeMsg(5.0), makeMsg(5.0), makeMsg(5.0, map[string]any{"reset": true})))
	if !reflect.DeepEqual(got, []any{5.0, 5.0}) {
		t.Fatalf("got %v, want [5 5]", got)
	}
}

func TestFilter_Deadband(t *testing.T) {
	n, _ := startNode(t, NewFilterNode, "filter", map[string]any{"func": "deadband", "gap": "5"})
	got := payloads(feed(t, n, makeMsg(20.0), makeMsg(23.0), makeMsg(26.0), makeMsg(27.0), makeMsg(31.5)))
	// Compared with the last value sent: 20 passes, 23 and 26? 26-20=6 > 5 passes, 27 no, 31.5 passes.
	if !reflect.DeepEqual(got, []any{20.0, 26.0, 31.5}) {
		t.Fatalf("got %v", got)
	}
}

func TestFilter_DeadbandPercentAndStrings(t *testing.T) {
	n, _ := startNode(t, NewFilterNode, "filter", map[string]any{"func": "deadbandEq", "gap": "10%"})
	got := payloads(feed(t, n, makeMsg("100 W"), makeMsg("105 W"), makeMsg("110 W"), makeMsg("abc")))
	if !reflect.DeepEqual(got, []any{"100 W", "110 W"}) {
		t.Fatalf("got %v", got)
	}
}

func TestFilter_NarrowbandBlocksSpikes(t *testing.T) {
	n, _ := startNode(t, NewFilterNode, "filter", map[string]any{"func": "narrowband", "gap": "10", "inout": "in"})
	got := payloads(feed(t, n, makeMsg(20.0), makeMsg(22.0), makeMsg(80.0), makeMsg(85.0)))
	if !reflect.DeepEqual(got, []any{20.0, 22.0, 85.0}) {
		t.Fatalf("got %v", got)
	}
}

// ---- range ------------------------------------------------------------------

func TestRange_Actions(t *testing.T) {
	cases := []struct {
		action string
		round  bool
		in     any
		want   []any
	}{
		{"scale", true, 2048.0, []any{50.0}},
		{"scale", false, 5000.0, []any{5000.0 / 4095 * 100}},
		{"clamp", false, 5000.0, []any{100.0}},
		{"clamp", false, -10.0, []any{0.0}},
		{"roll", true, 4095.0 + 2048.0, []any{50.0}},
		{"drop", false, 5000.0, []any{}},
		{"scale", true, "1023.75", []any{25.0}},
	}
	for _, tc := range cases {
		n, _ := startNode(t, NewRangeNode, "range", map[string]any{
			"action": tc.action, "minin": 0.0, "maxin": 4095.0, "minout": 0.0, "maxout": 100.0, "round": tc.round,
		})
		got := payloads(feed(t, n, makeMsg(tc.in)))
		if len(got) == 0 && len(tc.want) == 0 {
			continue
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s %v: got %v, want %v", tc.action, tc.in, got, tc.want)
		}
	}
}

func TestRange_RejectsEmptyInputRange(t *testing.T) {
	n, _ := NewRangeNode(flow.NodeConfig{Properties: map[string]any{"minin": 1.0, "maxin": 1.0, "minout": 0.0, "maxout": 1.0}})
	if err := n.Init(); err == nil {
		t.Fatal("want an error for an empty input range")
	}
}

// ---- trigger ----------------------------------------------------------------

func TestTrigger_SendsThenWaitsThenSends(t *testing.T) {
	n, c := startNode(t, NewTriggerNode, "trigger", map[string]any{
		"op1": "on", "op1type": "str", "op2": "off", "op2type": "str", "duration": 60.0, "units": "ms",
	})
	first := feed(t, n, makeMsg("motion"), makeMsg("motion again"))
	if !reflect.DeepEqual(payloads(first), []any{"on"}) {
		t.Fatalf("first: got %v", payloads(first))
	}
	eventually(t, func() bool { _, m := c.snapshot(); return len(m) == 1 })
	ports, msgs := c.snapshot()
	if msgs[0].Payload() != "off" || ports[0] != 0 {
		t.Fatalf("second: got %v on port %d", msgs[0].Payload(), ports[0])
	}
	// After the wait it can start again.
	if got := payloads(feed(t, n, makeMsg("x"))); !reflect.DeepEqual(got, []any{"on"}) {
		t.Fatalf("restart: got %v", got)
	}
}

func TestTrigger_ExtendKeepsItOn(t *testing.T) {
	n, c := startNode(t, NewTriggerNode, "trigger", map[string]any{
		"op1": "on", "op2": "off", "duration": 80.0, "units": "ms", "extend": true,
	})
	feed(t, n, makeMsg(1.0))
	for i := 0; i < 4; i++ {
		time.Sleep(40 * time.Millisecond)
		feed(t, n, makeMsg(1.0))
	}
	if _, m := c.snapshot(); len(m) != 0 {
		t.Fatalf("extended trigger fired early: %d messages", len(m))
	}
	eventually(t, func() bool { _, m := c.snapshot(); return len(m) == 1 })
}

func TestTrigger_ResetCancels(t *testing.T) {
	n, c := startNode(t, NewTriggerNode, "trigger", map[string]any{
		"op1": "on", "op2": "off", "duration": 50.0, "units": "ms", "reset": "stop",
	})
	feed(t, n, makeMsg(1.0), makeMsg("stop"))
	time.Sleep(100 * time.Millisecond)
	if _, m := c.snapshot(); len(m) != 0 {
		t.Fatalf("reset trigger still sent %d messages", len(m))
	}
}

func TestTrigger_SecondOutputAndOriginalMessage(t *testing.T) {
	n, c := startNode(t, NewTriggerNode, "trigger", map[string]any{
		"op1type": "pay", "op2type": "payl", "duration": 40.0, "units": "ms", "second": true,
	})
	first := feed(t, n, makeMsg("a"), makeMsg("b"))
	if !reflect.DeepEqual(payloads(first), []any{"a"}) {
		t.Fatalf("first: %v", payloads(first))
	}
	eventually(t, func() bool { _, m := c.snapshot(); return len(m) == 1 })
	ports, msgs := c.snapshot()
	if ports[0] != 1 || msgs[0].Payload() != "b" {
		t.Fatalf("second: %v on port %d, want b on port 1", msgs[0].Payload(), ports[0])
	}
}

func TestTrigger_NothingAndResend(t *testing.T) {
	n, c := startNode(t, NewTriggerNode, "trigger", map[string]any{
		"op1": "ping", "then": "resend", "duration": 30.0, "units": "ms",
	})
	feed(t, n, makeMsg(1.0))
	eventually(t, func() bool { _, m := c.snapshot(); return len(m) >= 3 })
	feed(t, n, makeMsg(nil, map[string]any{"reset": true}))
	_, before := c.snapshot()
	time.Sleep(90 * time.Millisecond)
	if _, after := c.snapshot(); len(after) > len(before)+1 {
		t.Fatalf("resend continued after reset: %d → %d", len(before), len(after))
	}

	n2, _ := startNode(t, NewTriggerNode, "trigger", map[string]any{"op1type": "nul", "then": "reset"})
	if got := feed(t, n2, makeMsg(1.0)); len(got) != 0 {
		t.Fatalf("op1 nothing still sent %v", payloads(got))
	}
}

func TestTrigger_ByTopic(t *testing.T) {
	n, _ := startNode(t, NewTriggerNode, "trigger", map[string]any{
		"op1": "on", "then": "reset", "bytopic": "topic",
	})
	got := feed(t, n,
		makeMsg(1.0, map[string]any{"topic": "a"}),
		makeMsg(1.0, map[string]any{"topic": "b"}),
		makeMsg(1.0, map[string]any{"topic": "a"}),
	)
	if len(got) != 2 {
		t.Fatalf("want one run per topic, got %d messages", len(got))
	}
}

// ---- split / join -----------------------------------------------------------

func TestSplitJoin_RoundTrips(t *testing.T) {
	inputs := []any{
		"line 1\nline 2\nline 3",
		[]any{1.0, 2.0, 3.0, 4.0, 5.0},
		map[string]any{"voltage": 231.0, "current": 1.2},
		[]byte("ab,cd,ef"),
	}
	for _, in := range inputs {
		props := map[string]any{}
		if _, ok := in.([]byte); ok {
			props["splt"] = ","
		}
		s, _ := startNode(t, NewSplitNode, "split", props)
		j, _ := startNode(t, NewJoinNode, "join", map[string]any{"mode": "auto"})
		pieces := feed(t, s, makeMsg(in, map[string]any{"topic": "t"}))
		if len(pieces) < 2 {
			t.Fatalf("%v: split made %d messages", in, len(pieces))
		}
		joined := feed(t, j, pieces...)
		if len(joined) != 1 {
			t.Fatalf("%v: join sent %d messages", in, len(joined))
		}
		if !reflect.DeepEqual(joined[0].Payload(), in) {
			t.Errorf("round trip: got %#v, want %#v", joined[0].Payload(), in)
		}
		if joined[0].Get("parts") != nil || joined[0].Topic() != "t" {
			t.Errorf("join output: parts %v, topic %q", joined[0].Get("parts"), joined[0].Topic())
		}
	}
}

func TestSplit_PartsAndChunks(t *testing.T) {
	s, _ := startNode(t, NewSplitNode, "split", map[string]any{"arraySplt": 2.0})
	out := feed(t, s, makeMsg([]any{1.0, 2.0, 3.0}))
	if !reflect.DeepEqual(payloads(out), []any{[]any{1.0, 2.0}, []any{3.0}}) {
		t.Fatalf("chunks: %v", payloads(out))
	}
	p := out[1].Get("parts").(map[string]any)
	if p["index"] != 1.0 || p["count"] != 2.0 || p["type"] != "array" {
		t.Fatalf("parts: %v", p)
	}
	j, _ := startNode(t, NewJoinNode, "join", map[string]any{"mode": "auto"})
	// Out of order still joins in order.
	joined := feed(t, j, out[1], out[0])
	if !reflect.DeepEqual(joined[0].Payload(), []any{1.0, 2.0, 3.0}) {
		t.Fatalf("join chunks: %v", joined[0].Payload())
	}

	s2, _ := startNode(t, NewSplitNode, "split", map[string]any{"splt": "3", "spltType": "len"})
	if got := payloads(feed(t, s2, makeMsg("abcdefg"))); !reflect.DeepEqual(got, []any{"abc", "def", "g"}) {
		t.Fatalf("fixed length: %v", got)
	}

	s3, _ := startNode(t, NewSplitNode, "split", map[string]any{"addname": "topic"})
	obj := feed(t, s3, makeMsg(map[string]any{"b": 2.0, "a": 1.0}))
	if obj[0].Topic() != "a" || obj[1].Topic() != "b" {
		t.Fatalf("object keys to topic: %q %q", obj[0].Topic(), obj[1].Topic())
	}
}

func TestSplit_NestedSequences(t *testing.T) {
	s, _ := startNode(t, NewSplitNode, "split", nil)
	j, _ := startNode(t, NewJoinNode, "join", map[string]any{"mode": "auto"})
	outer := feed(t, s, makeMsg([]any{[]any{1.0, 2.0}, []any{3.0}}))
	var inner []*flow.Message
	for _, m := range outer {
		inner = append(inner, feed(t, s, m)...)
	}
	rows := feed(t, j, inner...)
	if len(rows) != 2 {
		t.Fatalf("inner join: %d", len(rows))
	}
	all := feed(t, j, rows...)
	if len(all) != 1 || !reflect.DeepEqual(all[0].Payload(), []any{[]any{1.0, 2.0}, []any{3.0}}) {
		t.Fatalf("outer join: %v", payloads(all))
	}
}

func TestJoin_ManualCountObjectAndComplete(t *testing.T) {
	j, _ := startNode(t, NewJoinNode, "join", map[string]any{"mode": "custom", "build": "object", "count": 2.0})
	out := feed(t, j,
		makeMsg(21.5, map[string]any{"topic": "temp"}),
		makeMsg(60.0, map[string]any{"topic": "humidity"}),
	)
	if len(out) != 1 || !reflect.DeepEqual(out[0].Payload(), map[string]any{"temp": 21.5, "humidity": 60.0}) {
		t.Fatalf("object: %v", payloads(out))
	}

	j2, _ := startNode(t, NewJoinNode, "join", map[string]any{"mode": "custom", "build": "string", "joiner": ", "})
	out = feed(t, j2, makeMsg("a"), makeMsg("b"), makeMsg("c", map[string]any{"complete": true}))
	if len(out) != 1 || out[0].Payload() != "a, b, c" || out[0].Get("complete") != nil {
		t.Fatalf("complete: %v", payloads(out))
	}

	j3, _ := startNode(t, NewJoinNode, "join", map[string]any{"mode": "custom", "build": "array", "count": 3.0})
	out = feed(t, j3, makeMsg(1.0), makeMsg(2.0), makeMsg(nil, map[string]any{"reset": true}), makeMsg(3.0), makeMsg(4.0))
	if len(out) != 0 {
		t.Fatalf("reset should discard, got %v", payloads(out))
	}
}

func TestJoin_ManualTimeout(t *testing.T) {
	j, c := startNode(t, NewJoinNode, "join", map[string]any{"mode": "custom", "build": "array", "timeout": 0.05})
	feed(t, j, makeMsg(1.0), makeMsg(2.0))
	eventually(t, func() bool { _, m := c.snapshot(); return len(m) == 1 })
	_, msgs := c.snapshot()
	if !reflect.DeepEqual(msgs[0].Payload(), []any{1.0, 2.0}) {
		t.Fatalf("timeout: %v", msgs[0].Payload())
	}
}

// ---- batch ------------------------------------------------------------------

func TestBatch_CountWithOverlapThenJoin(t *testing.T) {
	b, _ := startNode(t, NewBatchNode, "batch", map[string]any{"mode": "count", "count": 3.0, "overlap": 1.0})
	j, _ := startNode(t, NewJoinNode, "join", map[string]any{"mode": "auto"})
	var seqs []any
	for i := 1; i <= 5; i++ {
		for _, m := range feed(t, j, feed(t, b, makeMsg(float64(i)))...) {
			seqs = append(seqs, m.Payload())
		}
	}
	want := []any{[]any{1.0, 2.0, 3.0}, []any{3.0, 4.0, 5.0}}
	if !reflect.DeepEqual(seqs, want) {
		t.Fatalf("got %v, want %v", seqs, want)
	}
}

func TestBatch_Interval(t *testing.T) {
	b, c := startNode(t, NewBatchNode, "batch", map[string]any{"mode": "interval", "interval": 0.05})
	feed(t, b, makeMsg(1.0), makeMsg(2.0))
	eventually(t, func() bool { _, m := c.snapshot(); return len(m) == 2 })
	_, msgs := c.snapshot()
	p := msgs[1].Get("parts").(map[string]any)
	if p["count"] != 2.0 || p["index"] != 1.0 {
		t.Fatalf("parts: %v", p)
	}
}
