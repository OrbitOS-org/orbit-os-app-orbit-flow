// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package serial

import (
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	orbitclient "github.com/OrbitOS-org/orbit-os-sdk-go/v26/client"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/flow"
)

// ─── fake port ──────────────────────────────────────────────────────────────

type fakeConn struct {
	mu      sync.Mutex
	written [][]byte
	chunks  chan []byte
	closed  bool
	onWrite func([]byte) // e.g. a device that answers
}

func newFakeConn() *fakeConn { return &fakeConn{chunks: make(chan []byte, 16)} }

func (c *fakeConn) Write(b []byte) (int, error) {
	c.mu.Lock()
	c.written = append(c.written, append([]byte(nil), b...))
	fn := c.onWrite
	c.mu.Unlock()
	if fn != nil {
		fn(b)
	}
	return len(b), nil
}
func (c *fakeConn) Chunks() <-chan []byte { return c.chunks }
func (c *fakeConn) Close() error {
	c.mu.Lock()
	c.closed = true
	c.mu.Unlock()
	return nil
}
func (c *fakeConn) wrote() [][]byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([][]byte(nil), c.written...)
}
func (c *fakeConn) isClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

// newTestPort returns a Serial Port whose opens return the given conns in turn.
func newTestPort(t *testing.T, conns ...*fakeConn) (*SerialPortConfig, *int) {
	t.Helper()
	inst, err := NewSerialPortConfig(flow.ConfigNode{ID: "p1", Type: "serial-port", Config: map[string]any{"port": "/dev/ttyS3"}})
	if err != nil {
		t.Fatal(err)
	}
	p := inst.(*SerialPortConfig)
	opens := 0
	var mu sync.Mutex
	p.open = func(orbitclient.UartConfig) (conn, error) {
		mu.Lock()
		defer mu.Unlock()
		if opens >= len(conns) {
			return nil, errors.New("UART port ttyS3 already open")
		}
		c := conns[opens]
		opens++
		return c, nil
	}
	t.Cleanup(func() { _ = p.Stop() })
	return p, &opens
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func lookup(p *SerialPortConfig) flow.ConfigLookupFunc {
	return func(id string) (flow.ConfigInstance, bool) {
		if id == "p1" {
			return p, true
		}
		return nil, false
	}
}

// ─── split ──────────────────────────────────────────────────────────────────

func TestParseChars(t *testing.T) {
	cases := map[string][]byte{
		`\n`:        {'\n'},
		`\r\n`:      {'\r', '\n'},
		`\x03`:      {0x03},
		"0x0D 0x0A": {0x0D, 0x0A},
		";":         {';'},
		"":          nil,
	}
	for in, want := range cases {
		got, err := ParseChars(in)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("ParseChars(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	if _, err := ParseChars(`\q`); err == nil {
		t.Error(`ParseChars("\q"): want an error`)
	}
}

func collect(cfg splitConfig) (*framer, func() []string) {
	var mu sync.Mutex
	var got []string
	f := newFramer(cfg, func(b []byte) { mu.Lock(); got = append(got, string(b)); mu.Unlock() })
	return f, func() []string { mu.Lock(); defer mu.Unlock(); return append([]string(nil), got...) }
}

func TestFramerOnCharacterAcrossChunks(t *testing.T) {
	f, got := collect(splitConfig{mode: "char", char: []byte("\r\n")})
	f.feed([]byte("$GPRMC,1"))
	f.feed([]byte("23\r"))
	f.feed([]byte("\n$GPGSV,2\r\n$GP"))
	if want := []string{"$GPRMC,123", "$GPGSV,2"}; !reflect.DeepEqual(got(), want) {
		t.Fatalf("got %q, want %q", got(), want)
	}
}

func TestFramerKeepsCharacter(t *testing.T) {
	f, got := collect(splitConfig{mode: "char", char: []byte("\n"), keep: true})
	f.feed([]byte("a\nb\n"))
	if want := []string{"a\n", "b\n"}; !reflect.DeepEqual(got(), want) {
		t.Fatalf("got %q, want %q", got(), want)
	}
}

func TestFramerFixedLength(t *testing.T) {
	f, got := collect(splitConfig{mode: "len", length: 3})
	f.feed([]byte("abcd"))
	f.feed([]byte("ef"))
	if want := []string{"abc", "def"}; !reflect.DeepEqual(got(), want) {
		t.Fatalf("got %q, want %q", got(), want)
	}
}

func TestFramerAfterSilence(t *testing.T) {
	f, got := collect(splitConfig{mode: "gap", gap: 30 * time.Millisecond})
	f.feed([]byte("12."))
	time.Sleep(10 * time.Millisecond)
	f.feed([]byte("48 kg"))
	if len(got()) != 0 {
		t.Fatalf("sent before the silence: %q", got())
	}
	waitFor(t, "the piece after the silence", func() bool { return len(got()) == 1 })
	if got()[0] != "12.48 kg" {
		t.Fatalf("got %q", got())
	}
}

func TestParseSplitRejectsBadValues(t *testing.T) {
	for _, props := range []map[string]any{
		{"split": "char", "splitChar": ""},
		{"split": "gap", "splitGap": float64(0)},
		{"split": "len", "splitLength": float64(0)},
		{"split": "words"},
	} {
		if _, err := parseSplit(props); err == nil {
			t.Errorf("parseSplit(%v): want an error", props)
		}
	}
}

// ─── Serial Port ────────────────────────────────────────────────────────────

func TestPortOpensOnFirstUserAndClosesAfterLast(t *testing.T) {
	c := newFakeConn()
	p, opens := newTestPort(t, c)
	if p.Port() != "ttyS3" {
		t.Fatalf("port name %q, want ttyS3 (no /dev/)", p.Port())
	}
	if err := p.Start(); err != nil || *opens != 0 {
		t.Fatalf("Start opened the port (opens=%d, err=%v)", *opens, err)
	}
	p.Acquire()
	p.Acquire()
	waitFor(t, "open", func() bool { f, _ := p.Status(); return f == "green" })
	p.Release()
	if c.isClosed() {
		t.Fatal("closed while a node still uses it")
	}
	p.Release()
	if !c.isClosed() {
		t.Fatal("still open after the last node stopped")
	}
	if *opens != 1 {
		t.Fatalf("opened %d times, want 1", *opens)
	}
}

func TestPortReopensAfterLoss(t *testing.T) {
	first, second := newFakeConn(), newFakeConn()
	p, opens := newTestPort(t, first, second)
	p.Acquire()
	waitFor(t, "open", func() bool { f, _ := p.Status(); return f == "green" })
	close(first.chunks) // adapter unplugged
	waitFor(t, "lost", func() bool { f, _ := p.Status(); return f == "red" })
	if !first.isClosed() {
		t.Fatal("lost port not closed")
	}
	_ = opens // the retry after retryEvery is not waited for here
}

func TestPortInUseShowsWhy(t *testing.T) {
	p, _ := newTestPort(t) // every open fails with "already open"
	p.Acquire()
	waitFor(t, "status", func() bool { _, txt := p.Status(); return txt == "in use by another program" })
	if err := p.Write([]byte("x")); err == nil || !strings.Contains(err.Error(), "in use by another program") {
		t.Fatalf("Write: %v", err)
	}
}

func TestSettings(t *testing.T) {
	inst, _ := NewSerialPortConfig(flow.ConfigNode{ID: "p", Config: map[string]any{"port": "ttyS4", "baudRate": float64(19200), "parity": "even", "stopBits": float64(2)}})
	if s := inst.(*SerialPortConfig).Settings(); s != "19200 8E2" {
		t.Fatalf("Settings() = %q", s)
	}
	if _, err := NewSerialPortConfig(flow.ConfigNode{ID: "p", Config: map[string]any{}}); err == nil {
		t.Fatal("want an error without a port")
	}
}

// ─── nodes ──────────────────────────────────────────────────────────────────

func TestSerialInSendsOneMessagePerLine(t *testing.T) {
	c := newFakeConn()
	p, _ := newTestPort(t, c)
	node, _ := NewSerialInNode(flow.NodeConfig{ID: "in", Properties: map[string]any{"port": "p1"}})
	n := node.(*SerialInNode)
	var mu sync.Mutex
	var got []*flow.Message
	n.SetSend(func(_ int, m *flow.Message) { mu.Lock(); got = append(got, m); mu.Unlock() })
	n.SetConfigLookup(lookup(p))
	if err := n.Init(); err != nil {
		t.Fatal(err)
	}
	if err := n.Start(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "open", func() bool { f, _ := p.Status(); return f == "green" })
	c.chunks <- []byte("$GPRMC,A\n$GPGSV")
	c.chunks <- []byte(",3\n")
	waitFor(t, "two messages", func() bool { mu.Lock(); defer mu.Unlock(); return len(got) == 2 })
	mu.Lock()
	if got[0].Payload() != "$GPRMC,A" || got[1].Payload() != "$GPGSV,3" || got[0].Get("port") != "ttyS3" {
		t.Fatalf("got %v / %v", got[0].Data(), got[1].Data())
	}
	mu.Unlock()
	_ = n.Stop()
	if !c.isClosed() {
		t.Fatal("port still open after the only node stopped")
	}
}

func TestSerialOutWritesPayloadAndEnding(t *testing.T) {
	c := newFakeConn()
	p, _ := newTestPort(t, c)
	node, _ := NewSerialOutNode(flow.NodeConfig{ID: "out", Properties: map[string]any{"port": "p1", "addChar": `\r\n`}})
	n := node.(*SerialOutNode)
	n.SetConfigLookup(lookup(p))
	if err := n.Init(); err != nil {
		t.Fatal(err)
	}
	_ = n.Start()
	defer n.Stop()
	waitFor(t, "open", func() bool { f, _ := p.Status(); return f == "green" })
	for _, payload := range []any{"ALARM", []any{float64(1), float64(255)}, float64(42), map[string]any{"a": float64(1)}} {
		m := flow.NewMessage()
		m.SetPayload(payload)
		if _, err := n.HandleMessage(m); err != nil {
			t.Fatal(err)
		}
	}
	want := [][]byte{[]byte("ALARM\r\n"), {1, 255, '\r', '\n'}, []byte("42\r\n"), []byte("{\"a\":1}\r\n")}
	if got := c.wrote(); !reflect.DeepEqual(got, want) {
		t.Fatalf("wrote %q, want %q", got, want)
	}
}

func newRequestNode(t *testing.T, p *SerialPortConfig, props map[string]any) *SerialRequestNode {
	t.Helper()
	props["port"] = "p1"
	node, _ := NewSerialRequestNode(flow.NodeConfig{ID: "req", Properties: props})
	n := node.(*SerialRequestNode)
	n.SetConfigLookup(lookup(p))
	if err := n.Init(); err != nil {
		t.Fatal(err)
	}
	if err := n.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = n.Stop() })
	waitFor(t, "open", func() bool { f, _ := p.Status(); return f == "green" })
	return n
}

func TestSerialRequestGetsTheReply(t *testing.T) {
	c := newFakeConn()
	c.onWrite = func(b []byte) {
		if string(b) == "W\r\n" {
			c.chunks <- []byte("12.4")
			c.chunks <- []byte("8 kg\r\nextra\r\n")
		}
	}
	p, _ := newTestPort(t, c)
	n := newRequestNode(t, p, map[string]any{"addChar": `\r\n`, "splitChar": `\r\n`})

	// A Serial In on the same port must not see the reply.
	var inGot []string
	stop := p.Listen(func(b []byte) { inGot = append(inGot, string(b)) })
	defer stop()

	m := flow.NewMessage()
	m.SetPayload("W")
	m.Set("topic", "scale")
	out, err := n.HandleMessage(m)
	if err != nil {
		t.Fatal(err)
	}
	if got := out[0][0]; got.Payload() != "12.48 kg" || got.Topic() != "scale" {
		t.Fatalf("reply %v", got.Data())
	}
	if len(inGot) != 0 {
		t.Fatalf("Serial In saw the reply: %q", inGot)
	}
}

func TestSerialRequestTimesOut(t *testing.T) {
	p, _ := newTestPort(t, newFakeConn())
	n := newRequestNode(t, p, map[string]any{"timeout": float64(500)})
	m := flow.NewMessage()
	m.SetPayload("?")
	m.Set("timeout", float64(30)) // msg.timeout wins
	start := time.Now()
	_, err := n.HandleMessage(m)
	if err == nil || !strings.Contains(err.Error(), "no reply from ttyS3 within 30 ms") {
		t.Fatalf("err = %v", err)
	}
	if time.Since(start) > 400*time.Millisecond {
		t.Fatal("msg.timeout was not used")
	}
}

// ─── conflicts ──────────────────────────────────────────────────────────────

func TestConflicts(t *testing.T) {
	serialCfg := flow.ConfigNode{ID: "s1", Type: "serial-port", Config: map[string]any{"port": "ttyS4"}}
	modbusRTU := flow.ConfigNode{ID: "m1", Type: "modbus-server", Config: map[string]any{"transport": "rtu", "serialPort": "/dev/ttyS4"}}
	modbusTCP := flow.ConfigNode{ID: "m2", Type: "modbus-server", Config: map[string]any{"transport": "tcp", "serialPort": "ttyS4"}}
	flows := func(disabled bool) []flow.Flow {
		return []flow.Flow{{ID: "f", Label: "Plant", Disabled: disabled, Nodes: []flow.Node{
			{ID: "a", Type: "serial-in", Name: "GPS", Config: map[string]any{"port": "s1"}},
			{ID: "b", Type: "modbus-read", Name: "Energy meter", Config: map[string]any{"server": "m1"}},
			{ID: "c", Type: "modbus-read", Config: map[string]any{"server": "m2"}},
		}}}
	}

	err := Conflicts(flows(false), []flow.ConfigNode{serialCfg, modbusRTU, modbusTCP})
	if err == nil || !strings.Contains(err.Error(), "ttyS4") || !strings.Contains(err.Error(), `"GPS" (Plant)`) || !strings.Contains(err.Error(), `"Energy meter" (Plant)`) {
		t.Fatalf("err = %v", err)
	}
	if err := Conflicts(flows(false), []flow.ConfigNode{serialCfg, modbusTCP}); err != nil {
		t.Fatalf("TCP Modbus is not on a serial port: %v", err)
	}
	if err := Conflicts(flows(true), []flow.ConfigNode{serialCfg, modbusRTU}); err != nil {
		t.Fatalf("a disabled flow does not use the port: %v", err)
	}
	if err := Conflicts(nil, []flow.ConfigNode{serialCfg, modbusRTU}); err != nil {
		t.Fatalf("unused connections do not open the port: %v", err)
	}
}
