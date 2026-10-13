// Copyright (C) 2026 Dennis Bleul
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package nodes

import (
	"reflect"
	"testing"
)

func TestParseHexBytes(t *testing.T) {
	cases := []struct {
		in   string
		want []byte
	}{
		{"9F 00 3C", []byte{0x9F, 0x00, 0x3C}},
		{"0x9F,0x00,0x3C", []byte{0x9F, 0x00, 0x3C}},
		{"9f,00 3c", []byte{0x9F, 0x00, 0x3C}},
		{"FF", []byte{0xFF}},
		{"", []byte{}},
	}
	for _, c := range cases {
		got, err := ParseHexBytes(c.in)
		if err != nil {
			t.Errorf("ParseHexBytes(%q): unexpected error: %v", c.in, err)
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("ParseHexBytes(%q) = % X, want % X", c.in, got, c.want)
		}
	}
}

func TestParseHexBytes_InvalidToken(t *testing.T) {
	if _, err := ParseHexBytes("9F ZZ"); err == nil {
		t.Fatal("expected error for invalid hex token, got nil")
	}
}

func TestParseHexBytes_OutOfByteRange(t *testing.T) {
	if _, err := ParseHexBytes("1FF"); err == nil {
		t.Fatal("expected error for a value that doesn't fit in a byte, got nil")
	}
}

func TestPayloadToBytes_NumberArray(t *testing.T) {
	got, err := PayloadToBytes([]any{float64(0x9F), float64(0x00), float64(255)})
	if err != nil {
		t.Fatalf("PayloadToBytes: %v", err)
	}
	want := []byte{0x9F, 0x00, 0xFF}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("PayloadToBytes = % X, want % X", got, want)
	}
}

func TestPayloadToBytes_RawBytes(t *testing.T) {
	in := []byte{1, 2, 3}
	got, err := PayloadToBytes(in)
	if err != nil {
		t.Fatalf("PayloadToBytes: %v", err)
	}
	if !reflect.DeepEqual(got, in) {
		t.Errorf("PayloadToBytes = % X, want % X", got, in)
	}
}

func TestPayloadToBytes_SingleNumber(t *testing.T) {
	got, err := PayloadToBytes(float64(42))
	if err != nil {
		t.Fatalf("PayloadToBytes: %v", err)
	}
	if !reflect.DeepEqual(got, []byte{42}) {
		t.Errorf("PayloadToBytes = % X, want [2A]", got)
	}
}

func TestPayloadToBytes_Nil(t *testing.T) {
	got, err := PayloadToBytes(nil)
	if err != nil || got != nil {
		t.Errorf("PayloadToBytes(nil) = (% X, %v), want (nil, nil)", got, err)
	}
}

func TestPayloadToBytes_RejectsOutOfRangeNumber(t *testing.T) {
	if _, err := PayloadToBytes([]any{float64(300)}); err == nil {
		t.Fatal("expected error for a number outside 0-255, got nil")
	}
	if _, err := PayloadToBytes(float64(-1)); err == nil {
		t.Fatal("expected error for a negative number, got nil")
	}
}

func TestPayloadToBytes_RejectsUnsupportedType(t *testing.T) {
	if _, err := PayloadToBytes("not bytes"); err == nil {
		t.Fatal("expected error for a string payload, got nil")
	}
	if _, err := PayloadToBytes(map[string]any{}); err == nil {
		t.Fatal("expected error for an object payload, got nil")
	}
}

func TestBytesToIntArray(t *testing.T) {
	got := BytesToIntArray([]byte{0x00, 0x7F, 0xFF})
	want := []int{0, 127, 255}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("BytesToIntArray = %v, want %v", got, want)
	}
}

func TestBytesToIntArray_Empty(t *testing.T) {
	got := BytesToIntArray(nil)
	if len(got) != 0 {
		t.Errorf("BytesToIntArray(nil) = %v, want empty", got)
	}
}
