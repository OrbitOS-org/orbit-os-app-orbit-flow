// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package modbus

import (
	"context"
	"encoding/binary"
	"fmt"
	"strings"
	"sync"
	"time"

	orbitclient "github.com/OrbitOS-org/orbit-os-sdk-go/v26/client"
	gomodbus "github.com/goburrow/modbus"
)

// Modbus RTU framing constants, mirroring github.com/goburrow/modbus's
// unexported rtuclient.go.
const (
	rtuMinSize       = 4
	rtuMaxSize       = 256
	rtuExceptionSize = 5
)

// orbitRTUHandler implements gomodbus.Packager and gomodbus.Transporter
// (together, gomodbus.ClientHandler) over an OrbitOS UartManager handle
// instead of a directly-opened /dev/tty* file.
//
// goburrow/modbus's own RTUClientHandler always calls goburrow/serial.Open
// on a raw device path with no way to substitute a different transport —
// which Orbit OS apps do not use: an app reaches the serial ports through
// the UART API (open, then calls on a handle), not through the device files.
// This type reimplements the RTU framing (address + function
// + data + CRC16) and read/write pump against client.UartPort instead.
type orbitRTUHandler struct {
	SlaveId byte

	uartMgr *orbitclient.UartManager
	cfg     orbitclient.UartConfig
	timeout time.Duration

	mu      sync.Mutex
	port    *orbitclient.UartPort
	cancel  context.CancelFunc
	chunks  <-chan []byte
	pending []byte
}

// newOrbitRTUHandler builds a handler bound to the given OrbitOS UART
// manager and port configuration. Connect() opens the port lazily.
func newOrbitRTUHandler(uartMgr *orbitclient.UartManager, cfg orbitclient.UartConfig, timeout time.Duration) *orbitRTUHandler {
	return &orbitRTUHandler{uartMgr: uartMgr, cfg: cfg, timeout: timeout}
}

// normalizeUartPortName strips a "/dev/" prefix, if present, since
// UartManager identifies ports by bare name (e.g. "ttyAMA0", as returned by
// UartManager.ListPorts), not by device path.
func normalizeUartPortName(name string) string {
	return strings.TrimPrefix(strings.TrimSpace(name), "/dev/")
}

// Connect opens the UART port (idempotent) and starts the background chunk
// reader that Send's read loop drains from.
func (h *orbitRTUHandler) Connect() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.port != nil {
		return nil
	}

	port, err := h.uartMgr.Open(h.cfg)
	if err != nil {
		return fmt.Errorf("modbus: open uart %s: %w", h.cfg.Port, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	chunks, err := port.ListenAsync(ctx, rtuMaxSize)
	if err != nil {
		cancel()
		_ = port.Close()
		return fmt.Errorf("modbus: listen uart %s: %w", h.cfg.Port, err)
	}

	h.port = port
	h.cancel = cancel
	h.chunks = chunks
	return nil
}

// Close closes the UART port, if open.
func (h *orbitRTUHandler) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.port == nil {
		return nil
	}
	h.cancel()
	err := h.port.Close()
	h.port, h.cancel, h.chunks, h.pending = nil, nil, nil, nil
	return err
}

// ─── gomodbus.Packager ──────────────────────────────────────────────────────
// Identical wire framing to goburrow/modbus's rtuPackager: 1-byte slave
// address, 1-byte function code, data, 2-byte little-endian CRC16.

func (h *orbitRTUHandler) Encode(pdu *gomodbus.ProtocolDataUnit) ([]byte, error) {
	length := len(pdu.Data) + 4
	if length > rtuMaxSize {
		return nil, fmt.Errorf("modbus: length of data '%v' must not be bigger than '%v'", length, rtuMaxSize)
	}
	adu := make([]byte, length)
	adu[0] = h.SlaveId
	adu[1] = pdu.FunctionCode
	copy(adu[2:], pdu.Data)

	checksum := crc16Modbus(adu[:length-2])
	adu[length-2] = byte(checksum)
	adu[length-1] = byte(checksum >> 8)
	return adu, nil
}

func (h *orbitRTUHandler) Verify(aduRequest, aduResponse []byte) error {
	if len(aduResponse) < rtuMinSize {
		return fmt.Errorf("modbus: response length '%v' does not meet minimum '%v'", len(aduResponse), rtuMinSize)
	}
	if aduResponse[0] != aduRequest[0] {
		return fmt.Errorf("modbus: response slave id '%v' does not match request '%v'", aduResponse[0], aduRequest[0])
	}
	return nil
}

func (h *orbitRTUHandler) Decode(adu []byte) (*gomodbus.ProtocolDataUnit, error) {
	length := len(adu)
	want := crc16Modbus(adu[:length-2])
	got := uint16(adu[length-1])<<8 | uint16(adu[length-2])
	if want != got {
		return nil, fmt.Errorf("modbus: response crc '%v' does not match expected '%v'", got, want)
	}
	return &gomodbus.ProtocolDataUnit{
		FunctionCode: adu[1],
		Data:         adu[2 : length-2],
	}, nil
}

// ─── gomodbus.Transporter ───────────────────────────────────────────────────

// Send writes the RTU request ADU to the UART port and reads back the
// response, mirroring goburrow/modbus's rtuSerialTransporter.Send but
// sourcing bytes from the background UartPort.ListenAsync channel instead
// of a local io.ReadWriteCloser.
func (h *orbitRTUHandler) Send(aduRequest []byte) ([]byte, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.port == nil {
		return nil, fmt.Errorf("modbus: uart port %s not connected", h.cfg.Port)
	}
	if _, err := h.port.Write(aduRequest); err != nil {
		return nil, fmt.Errorf("modbus: uart write %s: %w", h.cfg.Port, err)
	}

	deadline := time.Now().Add(h.timeout)
	function := aduRequest[1]
	functionFail := aduRequest[1] & 0x80
	bytesToRead := calculateRTUResponseLength(aduRequest)

	var data [rtuMaxSize]byte
	n, err := h.readAtLeast(data[:], rtuMinSize, deadline)
	if err != nil {
		return nil, err
	}

	// As in upstream: once we know the function code echoed back, read the
	// remaining bytes of a full response, or of the shorter exception
	// frame, whichever applies.
	switch {
	case data[1] == function:
		if n < bytesToRead && bytesToRead > rtuMinSize && bytesToRead <= rtuMaxSize {
			n2, err := h.readAtLeast(data[n:bytesToRead], bytesToRead-n, deadline)
			n += n2
			if err != nil {
				return nil, err
			}
		}
	case data[1] == functionFail:
		if n < rtuExceptionSize {
			n2, err := h.readAtLeast(data[n:rtuExceptionSize], rtuExceptionSize-n, deadline)
			n += n2
			if err != nil {
				return nil, err
			}
		}
	}

	return append([]byte(nil), data[:n]...), nil
}

// readAtLeast blocks on the background ListenAsync channel, draining into p,
// until at least min bytes have been copied or deadline passes.
func (h *orbitRTUHandler) readAtLeast(p []byte, min int, deadline time.Time) (int, error) {
	var n int
	for n < min {
		if len(h.pending) > 0 {
			c := copy(p[n:], h.pending)
			h.pending = h.pending[c:]
			n += c
			continue
		}

		remaining := time.Until(deadline)
		if remaining <= 0 {
			return n, fmt.Errorf("modbus: uart read timeout on %s", h.cfg.Port)
		}
		timer := time.NewTimer(remaining)
		select {
		case chunk, ok := <-h.chunks:
			timer.Stop()
			if !ok {
				return n, fmt.Errorf("modbus: uart %s closed", h.cfg.Port)
			}
			c := copy(p[n:], chunk)
			n += c
			if c < len(chunk) {
				h.pending = append(h.pending, chunk[c:]...)
			}
		case <-timer.C:
			return n, fmt.Errorf("modbus: uart read timeout on %s", h.cfg.Port)
		}
	}
	return n, nil
}

// calculateRTUResponseLength mirrors goburrow/modbus's unexported
// calculateResponseLength using only exported function-code constants.
func calculateRTUResponseLength(adu []byte) int {
	length := rtuMinSize
	switch adu[1] {
	case gomodbus.FuncCodeReadDiscreteInputs, gomodbus.FuncCodeReadCoils:
		count := int(binary.BigEndian.Uint16(adu[4:]))
		length += 1 + count/8
		if count%8 != 0 {
			length++
		}
	case gomodbus.FuncCodeReadInputRegisters, gomodbus.FuncCodeReadHoldingRegisters, gomodbus.FuncCodeReadWriteMultipleRegisters:
		count := int(binary.BigEndian.Uint16(adu[4:]))
		length += 1 + count*2
	case gomodbus.FuncCodeWriteSingleCoil, gomodbus.FuncCodeWriteMultipleCoils, gomodbus.FuncCodeWriteSingleRegister, gomodbus.FuncCodeWriteMultipleRegisters:
		length += 4
	case gomodbus.FuncCodeMaskWriteRegister:
		length += 6
	}
	return length
}

// crc16Modbus computes the standard CRC-16/MODBUS checksum (poly 0xA001,
// init 0xFFFF), independent of goburrow/modbus's private lookup tables.
func crc16Modbus(data []byte) uint16 {
	crc := uint16(0xFFFF)
	for _, b := range data {
		crc ^= uint16(b)
		for i := 0; i < 8; i++ {
			if crc&1 != 0 {
				crc = (crc >> 1) ^ 0xA001
			} else {
				crc >>= 1
			}
		}
	}
	return crc
}

// uartParityFromString maps LOOPZE's "none|even|odd" config value to the
// SDK's UartParity enum (replaces the old single-char goburrow/serial code).
func uartParityFromString(s string) orbitclient.UartParity {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "even", "e":
		return orbitclient.UartParityEven
	case "odd", "o":
		return orbitclient.UartParityOdd
	default:
		return orbitclient.UartParityNone
	}
}

// uartStopBitsFromInt maps LOOPZE's 1|2 config value to the SDK's
// UartStopBits enum.
func uartStopBitsFromInt(n int) orbitclient.UartStopBits {
	if n == 2 {
		return orbitclient.UartStopBits2
	}
	return orbitclient.UartStopBits1
}
