// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package serial

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	orbitclient "github.com/OrbitOS-org/orbit-os-sdk-go/v26/client"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/flow"
	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/nodes"
	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/orbitsdk"
)

// retryEvery is how long the port waits before opening again after it
// failed to open or was lost.
const retryEvery = 5 * time.Second

// maxChunk is the most bytes one read from the UART API may return.
const maxChunk = 1024

// conn is an open serial port: what SerialPortConfig needs from the UART
// API, so tests can use a fake.
type conn interface {
	Write(data []byte) (int, error)
	// Chunks delivers the bytes read from the port. It is closed when the
	// port is lost (or closed).
	Chunks() <-chan []byte
	Close() error
}

// opener opens a port with the given settings.
type opener func(cfg orbitclient.UartConfig) (conn, error)

// SerialPortConfig is the Serial Port config node: one serial port and its
// settings, shared by every Serial In / Out / Request node that names it.
// They share one connection to the port.
//
// The port is not opened by Start: the engine starts every config node of
// the workspace, used or not. It opens when the first node that uses it
// starts (Acquire) and closes when the last one stops (Release). While it
// cannot be opened (another app holds it, an adapter is unplugged) or after
// it is lost, it tries again every retryEvery.
//
// Incoming bytes go to the active request when a Serial Request is waiting
// for its reply, and to every Serial In otherwise, as in Node-RED.
//
// Implements flow.ConfigInstance.
type SerialPortConfig struct {
	id   string
	cfg  orbitclient.UartConfig
	open opener

	mu         sync.Mutex // guards everything below
	refs       int
	cancel     context.CancelFunc
	done       chan struct{}
	conn       conn
	listeners  map[*func([]byte)]struct{}
	active     func([]byte)
	watchers   map[*flow.StatusFunc]struct{}
	fill, text string

	writeMu   sync.Mutex // one write at a time on the wire
	requestMu sync.Mutex // one request at a time, across nodes
}

// NewSerialPortConfig parses a Serial Port config node.
func NewSerialPortConfig(c flow.ConfigNode) (flow.ConfigInstance, error) {
	props := c.Config
	port := PortName(nodes.StringVal(props, "port", ""))
	if port == "" {
		return nil, fmt.Errorf("serial-port %s: port is required", c.ID)
	}
	baud := nodes.IntVal(props, "baudRate", 9600)
	if baud <= 0 {
		return nil, fmt.Errorf("serial-port %s: baudRate must be positive", c.ID)
	}
	dataBits := nodes.IntVal(props, "dataBits", 8)
	if dataBits < 5 || dataBits > 8 {
		return nil, fmt.Errorf("serial-port %s: dataBits must be 5 to 8", c.ID)
	}
	return &SerialPortConfig{
		id: c.ID,
		cfg: orbitclient.UartConfig{
			Port:        port,
			Baudrate:    baud,
			DataBits:    dataBits,
			Parity:      parity(nodes.StringVal(props, "parity", "none")),
			StopBits:    stopBits(nodes.IntVal(props, "stopBits", 1)),
			FlowControl: orbitclient.UartFlowNone,
		},
		open:      openOrbit,
		listeners: map[*func([]byte)]struct{}{},
		watchers:  map[*flow.StatusFunc]struct{}{},
		fill:      "grey",
		text:      "not in use",
	}, nil
}

// PortName is the name the UART API uses for a port: "ttyS3", not "/dev/ttyS3".
func PortName(s string) string {
	return strings.TrimPrefix(strings.TrimSpace(s), "/dev/")
}

// Start does nothing: the port opens when a node acquires it.
func (p *SerialPortConfig) Start() error { return nil }

// Stop closes the port if a node still holds it.
func (p *SerialPortConfig) Stop() error {
	p.mu.Lock()
	p.refs = 0
	p.mu.Unlock()
	p.stopRunning()
	return nil
}

// Status returns the port's state.
func (p *SerialPortConfig) Status() (string, string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.fill, p.text
}

// Watch lets a node show the port's state: fn gets the current state at once
// and every change until the returned function is called.
func (p *SerialPortConfig) Watch(fn flow.StatusFunc) (stop func()) {
	key := &fn
	p.mu.Lock()
	p.watchers[key] = struct{}{}
	fill, text := p.fill, p.text
	p.mu.Unlock()
	fn(fill, text)
	return func() {
		p.mu.Lock()
		delete(p.watchers, key)
		p.mu.Unlock()
	}
}

// Port is the port name, for messages and status texts.
func (p *SerialPortConfig) Port() string { return p.cfg.Port }

// Acquire is called by a node when it starts; the first one opens the port.
func (p *SerialPortConfig) Acquire() {
	p.mu.Lock()
	p.refs++
	if p.refs != 1 || p.cancel != nil {
		p.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	p.cancel, p.done = cancel, done
	p.mu.Unlock()
	// "opening" goes out before run can report "connected".
	p.setStatus("yellow", "opening")
	go p.run(ctx, done)
}

// Release is called by a node when it stops; the last one closes the port.
func (p *SerialPortConfig) Release() {
	p.mu.Lock()
	if p.refs > 0 {
		p.refs--
	}
	last := p.refs == 0
	p.mu.Unlock()
	if last {
		p.stopRunning()
	}
}

func (p *SerialPortConfig) stopRunning() {
	p.mu.Lock()
	cancel, done := p.cancel, p.done
	p.cancel, p.done = nil, nil
	p.mu.Unlock()
	if cancel == nil {
		return
	}
	cancel()
	<-done
	p.setStatus("grey", "not in use")
}

// Listen sends every byte read from the port to fn while no request is
// waiting. The returned function stops it.
func (p *SerialPortConfig) Listen(fn func([]byte)) (stop func()) {
	key := &fn
	p.mu.Lock()
	p.listeners[key] = struct{}{}
	p.mu.Unlock()
	return func() {
		p.mu.Lock()
		delete(p.listeners, key)
		p.mu.Unlock()
	}
}

// Write writes data to the port.
func (p *SerialPortConfig) Write(data []byte) error {
	p.mu.Lock()
	c := p.conn
	p.mu.Unlock()
	if c == nil {
		return p.notOpen()
	}
	p.writeMu.Lock()
	defer p.writeMu.Unlock()
	if _, err := c.Write(data); err != nil {
		return fmt.Errorf("write to %s: %w", p.cfg.Port, err)
	}
	return nil
}

// Request writes data and sends the bytes that arrive afterwards to reply,
// instead of to the Serial In nodes, until end is called. Requests from all
// nodes on this port run one at a time.
func (p *SerialPortConfig) Request(data []byte, reply func([]byte)) (end func(), err error) {
	p.requestMu.Lock()
	p.mu.Lock()
	p.active = reply
	p.mu.Unlock()
	end = func() {
		p.mu.Lock()
		p.active = nil
		p.mu.Unlock()
		p.requestMu.Unlock()
	}
	if err := p.Write(data); err != nil {
		end()
		return nil, err
	}
	return end, nil
}

func (p *SerialPortConfig) notOpen() error {
	p.mu.Lock()
	text := p.text
	p.mu.Unlock()
	return fmt.Errorf("serial port %s is not open: %s", p.cfg.Port, text)
}

// run keeps the port open until ctx ends.
func (p *SerialPortConfig) run(ctx context.Context, done chan struct{}) {
	defer close(done)
	for {
		c, err := p.open(p.cfg)
		if err != nil {
			p.setStatus("red", openError(err))
			slog.Warn("serial port did not open", "id", p.id, "port", p.cfg.Port, "error", err)
		} else {
			p.mu.Lock()
			p.conn = c
			p.mu.Unlock()
			p.setStatus("green", "connected")
			slog.Info("serial port open", "id", p.id, "port", p.cfg.Port, "settings", p.Settings())

			lost := p.read(ctx, c)

			p.mu.Lock()
			p.conn = nil
			p.mu.Unlock()
			_ = c.Close()
			if !lost {
				return
			}
			p.setStatus("red", "port lost, reopening")
			slog.Warn("serial port lost", "id", p.id, "port", p.cfg.Port)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(retryEvery):
		}
	}
}

// read hands incoming bytes out until ctx ends (false) or the port is lost (true).
func (p *SerialPortConfig) read(ctx context.Context, c conn) (lost bool) {
	chunks := c.Chunks()
	for {
		select {
		case <-ctx.Done():
			return false
		case chunk, ok := <-chunks:
			if !ok {
				return ctx.Err() == nil
			}
			p.dispatch(chunk)
		}
	}
}

func (p *SerialPortConfig) dispatch(chunk []byte) {
	p.mu.Lock()
	if p.active != nil {
		fn := p.active
		p.mu.Unlock()
		fn(chunk)
		return
	}
	fns := make([]func([]byte), 0, len(p.listeners))
	for k := range p.listeners {
		fns = append(fns, *k)
	}
	p.mu.Unlock()
	for _, fn := range fns {
		fn(chunk)
	}
}

// Settings is the short form of the port settings: "9600 8N1".
func (p *SerialPortConfig) Settings() string {
	par := map[orbitclient.UartParity]string{orbitclient.UartParityEven: "E", orbitclient.UartParityOdd: "O"}[p.cfg.Parity]
	if par == "" {
		par = "N"
	}
	stop := "1"
	if p.cfg.StopBits == orbitclient.UartStopBits2 {
		stop = "2"
	}
	return fmt.Sprintf("%d %d%s%s", p.cfg.Baudrate, p.cfg.DataBits, par, stop)
}

func (p *SerialPortConfig) setStatus(fill, text string) {
	p.mu.Lock()
	if p.fill == fill && p.text == text {
		p.mu.Unlock()
		return
	}
	p.fill, p.text = fill, text
	fns := make([]flow.StatusFunc, 0, len(p.watchers))
	for k := range p.watchers {
		fns = append(fns, *k)
	}
	p.mu.Unlock()
	for _, fn := range fns {
		fn(fill, text)
	}
}

// openError says in a few words why the port did not open.
func openError(err error) string {
	switch msg := err.Error(); {
	case errors.Is(err, errNoSDK):
		return "Orbit OS not connected"
	case strings.Contains(msg, "already open"):
		return "in use by another program"
	case strings.Contains(msg, "no such file"), strings.Contains(msg, "not found"):
		return "port not found"
	default:
		return "cannot open: " + msg
	}
}

var errNoSDK = errors.New("orbit os sdk client not connected")

// openOrbit opens a port through the Orbit OS UART API.
func openOrbit(cfg orbitclient.UartConfig) (conn, error) {
	orbit := orbitsdk.Client()
	if orbit == nil {
		return nil, errNoSDK
	}
	port, err := orbit.UartManager.Open(cfg)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	chunks, err := port.ListenAsync(ctx, maxChunk)
	if err != nil {
		cancel()
		_ = port.Close()
		return nil, err
	}
	return &orbitConn{port: port, chunks: chunks, cancel: cancel}, nil
}

type orbitConn struct {
	port   *orbitclient.UartPort
	chunks <-chan []byte
	cancel context.CancelFunc
}

func (c *orbitConn) Write(data []byte) (int, error) { return c.port.Write(data) }
func (c *orbitConn) Chunks() <-chan []byte          { return c.chunks }
func (c *orbitConn) Close() error {
	c.cancel()
	return c.port.Close()
}

func parity(s string) orbitclient.UartParity {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "even", "e":
		return orbitclient.UartParityEven
	case "odd", "o":
		return orbitclient.UartParityOdd
	default:
		return orbitclient.UartParityNone
	}
}

func stopBits(n int) orbitclient.UartStopBits {
	if n == 2 {
		return orbitclient.UartStopBits2
	}
	return orbitclient.UartStopBits1
}

// SerialPortConfigTypeInfo returns the config type metadata for the editor.
func SerialPortConfigTypeInfo() flow.ConfigTypeInfo {
	return flow.ConfigTypeInfo{
		Type:        "serial-port",
		Label:       "Serial Port",
		Description: "A serial port and its settings, shared by the Serial nodes",
		Defaults: map[string]any{
			"name":     "",
			"port":     "",
			"baudRate": 9600,
			"dataBits": 8,
			"parity":   "none",
			"stopBits": 1,
		},
	}
}
