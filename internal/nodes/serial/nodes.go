// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package serial

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/flow"
	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/nodes"
)

// portUser is what the three Serial nodes share: the Serial Port they name,
// resolved and held while the node runs.
type portUser struct {
	cfg flow.NodeConfig
	nodes.BaseNode
	configLookup flow.ConfigLookupFunc

	portID  string
	port    *SerialPortConfig
	unwatch func()
}

func (u *portUser) SetConfigLookup(fn flow.ConfigLookupFunc) { u.configLookup = fn }

func (u *portUser) initPort(kind string) error {
	u.portID = nodes.StringVal(u.cfg.Properties, "port", "")
	if u.portID == "" {
		return fmt.Errorf("%s %s: no Serial Port configured", kind, u.cfg.ID)
	}
	return nil
}

// acquire resolves the Serial Port, shows its state on the node and opens it.
func (u *portUser) acquire(kind string) error {
	p, err := nodes.ResolveConfigInstance[SerialPortConfig](u.configLookup, u.portID, u.Status, nodes.ResolveConfigParams{
		NodeKind:   kind,
		NodeID:     u.cfg.ID,
		ConfigKind: "serial port",
		TypeLabel:  "a Serial Port",
	})
	if err != nil {
		return err
	}
	u.port = p
	if u.Status != nil {
		u.unwatch = p.Watch(u.Status)
	}
	p.Acquire()
	return nil
}

func (u *portUser) release() {
	if u.unwatch != nil {
		u.unwatch()
		u.unwatch = nil
	}
	if u.port != nil {
		u.port.Release()
		u.port = nil
	}
}

func (u *portUser) fail(err error) error {
	if u.Status != nil {
		u.Status("red", err.Error())
	}
	return err
}

// output turns received bytes into msg.payload: text, or a byte array (a
// list of numbers, as the I²C and SPI nodes give).
func output(mode string, b []byte) any {
	if mode == "bytes" {
		return nodes.BytesToIntArray(b)
	}
	return string(b)
}

// payloadBytes turns msg.payload into the bytes to send, as Node-RED's
// serial out does: text as it is, a byte array (numbers 0-255) as bytes,
// other values as text (numbers, true/false) or JSON (objects).
func payloadBytes(v any) ([]byte, error) {
	switch x := v.(type) {
	case nil:
		return nil, nil
	case string:
		return []byte(x), nil
	case []byte:
		return x, nil
	case []any:
		if b, err := nodes.PayloadToBytes(x); err == nil {
			return b, nil
		}
	case []int:
		b := make([]byte, len(x))
		for i, n := range x {
			if n < 0 || n > 255 {
				return json.Marshal(x)
			}
			b[i] = byte(n)
		}
		return b, nil
	case float64:
		return []byte(strconv.FormatFloat(x, 'f', -1, 64)), nil
	case bool:
		return []byte(strconv.FormatBool(x)), nil
	}
	return json.Marshal(v)
}

// ─── Serial In ──────────────────────────────────────────────────────────────

// SerialInNode sends one message for each piece of data the port receives.
type SerialInNode struct {
	portUser
	split  splitConfig
	format string

	framer *framer
	stop   func()
}

// NewSerialInNode is the NodeFactory for serial-in.
func NewSerialInNode(cfg flow.NodeConfig) (flow.NodeInstance, error) {
	return &SerialInNode{portUser: portUser{cfg: cfg}}, nil
}

func (n *SerialInNode) Init() error {
	if err := n.initPort("serial-in"); err != nil {
		return err
	}
	s, err := parseSplit(n.cfg.Properties)
	if err != nil {
		return fmt.Errorf("serial-in %s: %w", n.cfg.ID, err)
	}
	n.split, n.format = s, nodes.StringVal(n.cfg.Properties, "output", "text")
	return nil
}

func (n *SerialInNode) Start() error {
	if err := n.acquire("serial-in"); err != nil {
		return err
	}
	portName := n.port.Port()
	n.framer = newFramer(n.split, func(b []byte) {
		msg := flow.NewMessage()
		msg.SetPayload(output(n.format, b))
		msg.Set("port", portName)
		if n.Send != nil {
			n.Send(0, msg)
		}
	})
	n.stop = n.port.Listen(n.framer.feed)
	slog.Info("serial-in started", "node_id", n.cfg.ID, "port", portName)
	return nil
}

func (n *SerialInNode) Stop() error {
	if n.stop != nil {
		n.stop()
		n.stop = nil
	}
	if n.framer != nil {
		n.framer.reset()
	}
	n.release()
	return nil
}

func (n *SerialInNode) HandleMessage(*flow.Message) ([][]*flow.Message, error) { return nil, nil }

// SerialInTypeInfo returns the palette metadata.
func SerialInTypeInfo() flow.NodeTypeInfo {
	return flow.NodeTypeInfo{
		Type:        "serial-in",
		Category:    "hardware",
		Label:       "Serial In",
		Description: "Sends a message for each piece of data received on a serial port",
		Icon:        "serial",
		Keywords:    keywords,
		Defaults: map[string]any{
			"port":        "",
			"split":       "char",
			"splitChar":   `\n`,
			"keepChar":    false,
			"splitGap":    50,
			"splitLength": 16,
			"output":      "text",
		},
		Inputs:  0,
		Outputs: 1,
	}
}

var keywords = []string{"uart", "rs232", "rs-232", "rs485", "rs-485", "usb serial", "tty", "com port"}

// ─── Serial Out ─────────────────────────────────────────────────────────────

// SerialOutNode writes msg.payload to the port.
type SerialOutNode struct {
	portUser
	end []byte // added after each payload
}

// NewSerialOutNode is the NodeFactory for serial-out.
func NewSerialOutNode(cfg flow.NodeConfig) (flow.NodeInstance, error) {
	return &SerialOutNode{portUser: portUser{cfg: cfg}}, nil
}

func (n *SerialOutNode) Init() error {
	if err := n.initPort("serial-out"); err != nil {
		return err
	}
	end, err := ParseChars(nodes.StringVal(n.cfg.Properties, "addChar", ""))
	if err != nil {
		return fmt.Errorf("serial-out %s: add at the end: %w", n.cfg.ID, err)
	}
	n.end = end
	return nil
}

func (n *SerialOutNode) Start() error { return n.acquire("serial-out") }

func (n *SerialOutNode) Stop() error {
	n.release()
	return nil
}

func (n *SerialOutNode) HandleMessage(msg *flow.Message) ([][]*flow.Message, error) {
	if msg == nil || n.port == nil {
		return nil, nil
	}
	data, err := payloadBytes(msg.Payload())
	if err != nil {
		return nil, n.fail(fmt.Errorf("serial-out: payload: %w", err))
	}
	if err := n.port.Write(append(data, n.end...)); err != nil {
		return nil, n.fail(fmt.Errorf("serial-out: %w", err))
	}
	return nil, nil
}

// SerialOutTypeInfo returns the palette metadata.
func SerialOutTypeInfo() flow.NodeTypeInfo {
	return flow.NodeTypeInfo{
		Type:        "serial-out",
		Category:    "hardware",
		Label:       "Serial Out",
		Description: "Writes msg.payload to a serial port",
		Icon:        "serial",
		Keywords:    keywords,
		Defaults: map[string]any{
			"port":    "",
			"addChar": "",
		},
		Inputs:  1,
		Outputs: 0,
	}
}

// ─── Serial Request ─────────────────────────────────────────────────────────

// SerialRequestNode writes msg.payload and waits for the reply, which
// replaces msg.payload. Messages that arrive while it waits are handled one
// after the other (the engine gives a node one message at a time), and
// requests from all nodes on the same port never overlap.
type SerialRequestNode struct {
	portUser
	end     []byte
	split   splitConfig
	format  string
	timeout time.Duration

	stopped chan struct{}
}

// NewSerialRequestNode is the NodeFactory for serial-request.
func NewSerialRequestNode(cfg flow.NodeConfig) (flow.NodeInstance, error) {
	return &SerialRequestNode{portUser: portUser{cfg: cfg}}, nil
}

func (n *SerialRequestNode) Init() error {
	if err := n.initPort("serial-request"); err != nil {
		return err
	}
	props := n.cfg.Properties
	end, err := ParseChars(nodes.StringVal(props, "addChar", ""))
	if err != nil {
		return fmt.Errorf("serial-request %s: add at the end: %w", n.cfg.ID, err)
	}
	s, err := parseSplit(props)
	if err != nil {
		return fmt.Errorf("serial-request %s: reply: %w", n.cfg.ID, err)
	}
	ms := nodes.IntVal(props, "timeout", 1000)
	if ms <= 0 {
		return fmt.Errorf("serial-request %s: timeout must be at least 1 ms", n.cfg.ID)
	}
	n.end, n.split, n.format = end, s, nodes.StringVal(props, "output", "text")
	n.timeout = time.Duration(ms) * time.Millisecond
	return nil
}

func (n *SerialRequestNode) Start() error {
	n.stopped = make(chan struct{})
	return n.acquire("serial-request")
}

func (n *SerialRequestNode) Stop() error {
	if n.stopped != nil {
		close(n.stopped)
	}
	n.release()
	return nil
}

func (n *SerialRequestNode) HandleMessage(msg *flow.Message) ([][]*flow.Message, error) {
	if msg == nil || n.port == nil {
		return nil, nil
	}
	data, err := payloadBytes(msg.Payload())
	if err != nil {
		return nil, n.fail(fmt.Errorf("serial-request: payload: %w", err))
	}
	timeout := n.timeout
	if v, ok := msg.Get("timeout").(float64); ok && v > 0 {
		timeout = time.Duration(v) * time.Millisecond
	}

	replies := make(chan []byte, 1)
	f := newFramer(n.split, func(b []byte) {
		select {
		case replies <- b:
		default: // only the first piece is the reply
		}
	})
	end, err := n.port.Request(append(data, n.end...), f.feed)
	if err != nil {
		return nil, n.fail(fmt.Errorf("serial-request: %w", err))
	}
	defer func() {
		end()
		f.reset()
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case b := <-replies:
		out := msg.Clone()
		out.SetPayload(output(n.format, b))
		out.Set("port", n.port.Port())
		if n.Status != nil {
			n.Status("green", fmt.Sprintf("reply · %d bytes", len(b)))
		}
		return [][]*flow.Message{{out}}, nil
	case <-timer.C:
		return nil, n.fail(fmt.Errorf("serial-request: no reply from %s within %d ms", n.port.Port(), timeout.Milliseconds()))
	case <-n.stopped:
		return nil, nil
	}
}

// SerialRequestTypeInfo returns the palette metadata.
func SerialRequestTypeInfo() flow.NodeTypeInfo {
	return flow.NodeTypeInfo{
		Type:        "serial-request",
		Category:    "hardware",
		Label:       "Serial Request",
		Description: "Writes msg.payload to a serial port and waits for the reply",
		Icon:        "serial",
		Keywords:    keywords,
		Defaults: map[string]any{
			"port":        "",
			"addChar":     `\r\n`,
			"split":       "char",
			"splitChar":   `\n`,
			"keepChar":    false,
			"splitGap":    50,
			"splitLength": 16,
			"timeout":     1000,
			"output":      "text",
		},
		Inputs:  1,
		Outputs: 1,
	}
}
