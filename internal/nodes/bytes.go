// Copyright (C) 2026 Dennis Bleul
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package nodes

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseHexBytes parses a whitespace/comma-separated list of hex byte
// values, with or without a leading "0x" per token (e.g. "9F 00 3C" or
// "0x9F,0x00,0x3C"). Used by config fields that let an operator type a
// fixed byte sequence (e.g. i2c-transfer / spi-transfer's writeData).
func ParseHexBytes(s string) ([]byte, error) {
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return r == ' ' || r == ',' || r == '\t' || r == '\n'
	})
	out := make([]byte, 0, len(fields))
	for _, f := range fields {
		f = strings.TrimPrefix(strings.TrimPrefix(f, "0x"), "0X")
		v, err := strconv.ParseUint(f, 16, 8)
		if err != nil {
			return nil, fmt.Errorf("invalid hex byte %q: %w", f, err)
		}
		out = append(out, byte(v))
	}
	return out, nil
}

// PayloadToBytes converts a message payload into bytes to write over a
// hardware bus (I2C, SPI, ...). Accepts a raw []byte (e.g. set by a
// Function node), a JSON array of numbers (how a byte array survives the
// wire — see BytesToIntArray for the matching encode side), or a single
// number as a 1-byte write.
func PayloadToBytes(v any) ([]byte, error) {
	switch x := v.(type) {
	case []byte:
		return x, nil
	case []any:
		out := make([]byte, len(x))
		for i, e := range x {
			n, ok := e.(float64)
			if !ok || n < 0 || n > 255 {
				return nil, fmt.Errorf("payload[%d] is not a byte (0-255)", i)
			}
			out[i] = byte(n)
		}
		return out, nil
	case float64:
		if x < 0 || x > 255 {
			return nil, fmt.Errorf("payload is not a byte (0-255)")
		}
		return []byte{byte(x)}, nil
	case nil:
		return nil, nil
	default:
		return nil, fmt.Errorf("payload must be a byte array, a number, or empty (got %T)", v)
	}
}

// BytesToIntArray converts raw bytes to []int so the value survives JSON
// round-trips without becoming base64 — same convention as
// internal/nodes/network/http_in.go's bytesToNumberArray.
func BytesToIntArray(b []byte) []int {
	out := make([]int, len(b))
	for i, x := range b {
		out[i] = int(x)
	}
	return out
}
