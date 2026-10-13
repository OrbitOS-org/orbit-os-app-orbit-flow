// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package modbus

import (
	"testing"

	orbitclient "github.com/OrbitOS-org/orbit-os-sdk-go/v26/client"
	gomodbus "github.com/goburrow/modbus"
)

// crc16Modbus must match the textbook Modbus RTU CRC example: slave 1,
// function 3 (read holding registers), address 0, quantity 10 → CRC bytes
// C5 CD (low byte first).
func TestCRC16Modbus(t *testing.T) {
	got := crc16Modbus([]byte{0x01, 0x03, 0x00, 0x00, 0x00, 0x0A})
	if low, high := byte(got), byte(got>>8); low != 0xC5 || high != 0xCD {
		t.Fatalf("crc16Modbus = %04X (low=%02X high=%02X), want low=C5 high=CD", got, low, high)
	}
}

func TestOrbitRTUHandlerEncodeMatchesReferenceFrame(t *testing.T) {
	h := &orbitRTUHandler{SlaveId: 1}

	adu, err := h.Encode(&gomodbus.ProtocolDataUnit{
		FunctionCode: gomodbus.FuncCodeReadHoldingRegisters,
		Data:         []byte{0x00, 0x00, 0x00, 0x0A},
	})
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	want := []byte{0x01, 0x03, 0x00, 0x00, 0x00, 0x0A, 0xC5, 0xCD}
	if len(adu) != len(want) {
		t.Fatalf("Encode length = %d, want %d (% x)", len(adu), len(want), adu)
	}
	for i := range want {
		if adu[i] != want[i] {
			t.Fatalf("Encode = % x, want % x", adu, want)
		}
	}
}

func TestOrbitRTUHandlerEncodeDecodeRoundTrip(t *testing.T) {
	h := &orbitRTUHandler{SlaveId: 17}

	pdu := &gomodbus.ProtocolDataUnit{
		FunctionCode: gomodbus.FuncCodeWriteSingleRegister,
		Data:         []byte{0x00, 0x01, 0x00, 0x2A},
	}
	adu, err := h.Encode(pdu)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	if err := h.Verify(adu, adu); err != nil {
		t.Fatalf("Verify: %v", err)
	}

	decoded, err := h.Decode(adu)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if decoded.FunctionCode != pdu.FunctionCode {
		t.Fatalf("Decode FunctionCode = %v, want %v", decoded.FunctionCode, pdu.FunctionCode)
	}
	if string(decoded.Data) != string(pdu.Data) {
		t.Fatalf("Decode Data = % x, want % x", decoded.Data, pdu.Data)
	}
}

func TestOrbitRTUHandlerDecodeRejectsBadCRC(t *testing.T) {
	h := &orbitRTUHandler{SlaveId: 1}
	adu, err := h.Encode(&gomodbus.ProtocolDataUnit{FunctionCode: gomodbus.FuncCodeReadCoils, Data: []byte{0, 0, 0, 1}})
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	adu[len(adu)-1] ^= 0xFF // corrupt CRC high byte

	if _, err := h.Decode(adu); err == nil {
		t.Fatal("Decode: expected error for corrupted CRC, got nil")
	}
}

func TestOrbitRTUHandlerVerifyRejectsMismatchedSlaveID(t *testing.T) {
	h := &orbitRTUHandler{SlaveId: 1}
	req, _ := h.Encode(&gomodbus.ProtocolDataUnit{FunctionCode: gomodbus.FuncCodeReadCoils, Data: []byte{0, 0, 0, 1}})

	resp := append([]byte(nil), req...)
	resp[0] = 2 // different slave id in the "response"

	if err := h.Verify(req, resp); err == nil {
		t.Fatal("Verify: expected error for mismatched slave id, got nil")
	}
}

func TestNormalizeUartPortName(t *testing.T) {
	cases := map[string]string{
		"/dev/ttyUSB0": "ttyUSB0",
		"ttyAMA0":      "ttyAMA0",
		"  ttyS0  ":    "ttyS0",
	}
	for in, want := range cases {
		if got := normalizeUartPortName(in); got != want {
			t.Errorf("normalizeUartPortName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestUartParityAndStopBitsMapping(t *testing.T) {
	if uartParityFromString("even") != orbitclient.UartParityEven {
		t.Error("expected even parity")
	}
	if uartParityFromString("ODD") != orbitclient.UartParityOdd {
		t.Error("expected odd parity")
	}
	if uartParityFromString("") != orbitclient.UartParityNone {
		t.Error("expected none parity as default")
	}
	if uartStopBitsFromInt(2) != orbitclient.UartStopBits2 {
		t.Error("expected 2 stop bits")
	}
	if uartStopBitsFromInt(1) != orbitclient.UartStopBits1 {
		t.Error("expected 1 stop bit")
	}
}
