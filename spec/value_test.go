package spec

import (
	"testing"
)

func TestValue_ASCII(t *testing.T) {
	v := ASCII
	if v.String() != "ASCII" {
		t.Errorf("String() = %q, want %q", v.String(), "ASCII")
	}

	// Round-trip.
	enc, err := v.Encode("hello", 0)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if string(enc) != "hello" {
		t.Errorf("encoded = %q, want %q", enc, "hello")
	}
	dec, err := v.Decode(enc)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if dec != "hello" {
		t.Errorf("decoded = %q, want %q", dec, "hello")
	}
}

func TestValue_BCD(t *testing.T) {
	v := BCD
	if v.String() != "BCD" {
		t.Errorf("String() = %q, want %q", v.String(), "BCD")
	}

	// Round-trip: even-length.
	enc, err := v.Encode("1234", 0)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if len(enc) != 2 {
		t.Errorf("len(encoded) = %d, want 2", len(enc))
	}
	dec, err := v.Decode(enc)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if dec != "1234" {
		t.Errorf("decoded = %q, want %q", dec, "1234")
	}

	// Odd-length: left-padded with zero nibble by encoding.BCD.
	enc, err = v.Encode("123", 0)
	if err != nil {
		t.Fatalf("Encode odd: %v", err)
	}
	dec, err = v.Decode(enc)
	if err != nil {
		t.Fatalf("Decode odd: %v", err)
	}
	if dec != "0123" {
		t.Errorf("odd-length decoded = %q, want %q", dec, "0123")
	}
}

func TestValue_RBCD(t *testing.T) {
	// RBCD with fixed width.
	v := RBCD(12)

	// "1000" → right-justified to 12 chars → "000000001000" → BCD.
	enc, err := v.Encode("1000", 12)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if len(enc) != 6 {
		t.Errorf("len(encoded) = %d, want 6", len(enc))
	}
	dec, err := v.Decode(enc)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if dec != "000000001000" {
		t.Errorf("decoded = %q, want %q", dec, "000000001000")
	}

	// Value already at full width: no padding.
	enc, err = v.Encode("123456789012", 12)
	if err != nil {
		t.Fatalf("Encode full: %v", err)
	}
	dec, err = v.Decode(enc)
	if err != nil {
		t.Fatalf("Decode full: %v", err)
	}
	if dec != "123456789012" {
		t.Errorf("decoded = %q, want %q", dec, "123456789012")
	}

	// RBCD(0) uses fieldLen.
	v2 := RBCD(0)
	enc, err = v2.Encode("500", 6)
	if err != nil {
		t.Fatalf("Encode with fieldLen: %v", err)
	}
	dec, err = v2.Decode(enc)
	if err != nil {
		t.Fatalf("Decode with fieldLen: %v", err)
	}
	if dec != "000500" {
		t.Errorf("decoded = %q, want %q", dec, "000500")
	}
}

func TestValue_Hex(t *testing.T) {
	v := Hex
	if v.String() != "Hex" {
		t.Errorf("String() = %q, want %q", v.String(), "Hex")
	}

	// Round-trip.
	enc, err := v.Encode("deadbeef", 0)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if len(enc) != 4 {
		t.Errorf("len(encoded) = %d, want 4", len(enc))
	}
	dec, err := v.Decode(enc)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if dec != "deadbeef" {
		t.Errorf("decoded = %q, want %q", dec, "deadbeef")
	}

	// Invalid hex → error.
	_, err = v.Encode("xyz", 0)
	if err == nil {
		t.Error("expected error for invalid hex, got nil")
	}
}

func TestValue_Text(t *testing.T) {
	v := Text
	if v.String() != "Text" {
		t.Errorf("String() = %q, want %q", v.String(), "Text")
	}

	// Encode and decode with trailing spaces stripped.
	enc, err := v.Encode("hello   ", 0)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	dec, err := v.Decode(enc)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if dec != "hello" {
		t.Errorf("decoded = %q, want %q", dec, "hello")
	}
}

func TestValue_Raw(t *testing.T) {
	v := Raw
	if v.String() != "Raw" {
		t.Errorf("String() = %q, want %q", v.String(), "Raw")
	}

	// []byte round-trip.
	orig := []byte{0x00, 0x01, 0xFF, 0xFE}
	enc, err := v.Encode(orig, 0)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	dec, err := v.Decode(enc)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	out, ok := dec.([]byte)
	if !ok {
		t.Fatalf("decoded type = %T, want []byte", dec)
	}
	if len(out) != len(orig) || out[0] != 0x00 || out[2] != 0xFF {
		t.Errorf("decoded = %v, want %v", out, orig)
	}

	// String round-trip.
	enc, err = v.Encode("rawstring", 0)
	if err != nil {
		t.Fatalf("Encode string: %v", err)
	}
	dec, err = v.Decode(enc)
	if err != nil {
		t.Fatalf("Decode string: %v", err)
	}
	out, ok = dec.([]byte)
	if !ok {
		t.Fatalf("decoded type = %T, want []byte", dec)
	}
	if string(out) != "rawstring" {
		t.Errorf("decoded = %q, want %q", out, "rawstring")
	}

	// Unsupported type.
	_, err = v.Encode(42, 0)
	if err == nil {
		t.Error("expected error for int input, got nil")
	}
}

func TestValue_Binary(t *testing.T) {
	// 2-byte binary.
	v := Bin(2)
	enc, err := v.Encode(int64(0x1234), 0)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if len(enc) != 2 || enc[0] != 0x12 || enc[1] != 0x34 {
		t.Errorf("encoded = %x, want 1234", enc)
	}
	dec, err := v.Decode(enc)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if dec.(int64) != 0x1234 {
		t.Errorf("decoded = %#x, want 0x1234", dec)
	}

	// 4-byte binary.
	v4 := Bin(4)
	enc, err = v4.Encode(int64(0xDEADBEEF), 0)
	if err != nil {
		t.Fatalf("Encode 4: %v", err)
	}
	if len(enc) != 4 {
		t.Errorf("len(encoded) = %d, want 4", len(enc))
	}
	dec, err = v4.Decode(enc)
	if err != nil {
		t.Fatalf("Decode 4: %v", err)
	}
	if dec.(int64) != 0xDEADBEEF {
		t.Errorf("decoded = %#x, want 0xDEADBEEF", dec)
	}

	// Overflow.
	_, err = v.Encode(int64(0x10000), 0)
	if err == nil {
		t.Error("expected overflow error for 2-byte binary with 0x10000")
	}
}

func TestValue_String(t *testing.T) {
	tests := []struct {
		v    Value
		want string
	}{
		{ASCII, "ASCII"},
		{BCD, "BCD"},
		{RBCD(12), "RBCD(12)"},
		{RBCD(0), "RBCD"},
		{Hex, "Hex"},
		{Text, "Text"},
		{Raw, "Raw"},
		{Bin(4), "Binary(4)"},
		{Bin(0), "Binary"},
	}
	for _, tt := range tests {
		if got := tt.v.String(); got != tt.want {
			t.Errorf("%s.String() = %q, want %q", tt.want, got, tt.want)
		}
	}
}

func TestValueByName(t *testing.T) {
	tests := []struct {
		name string
		want Value
	}{
		{"ASCII", ASCII},
		{"ascii", ASCII},
		{"BCD", BCD},
		{"bcd", BCD},
		{"Hex", Hex},
		{"hex", Hex},
		{"Text", Text},
		{"text", Text},
		{"Raw", Raw},
		{"raw", Raw},
		{"unknown", nil},
		{"", nil},
		{"RBCD", nil},  // parameterized, not in lookup
		{"Binary", nil}, // parameterized, not in lookup
	}
	for _, tt := range tests {
		got := ValueByName(tt.name)
		if got != tt.want {
			t.Errorf("ValueByName(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestValue_toString(t *testing.T) {
	tests := []struct {
		v    any
		want string
	}{
		{"hello", "hello"},
		{[]byte("world"), "world"},
		{int64(123), "123"},
		{int(456), "456"},
		{uint64(789), "789"},
		{uint(101), "101"},
		{int32(202), "202"},
		{uint32(303), "303"},
	}
	for _, tt := range tests {
		got, err := toString(tt.v)
		if err != nil {
			t.Errorf("toString(%v): unexpected error: %v", tt.v, err)
			continue
		}
		if got != tt.want {
			t.Errorf("toString(%v) = %q, want %q", tt.v, got, tt.want)
		}
	}

	// Unsupported types.
	_, err := toString(3.14)
	if err == nil {
		t.Error("expected error for float64, got nil")
	}
	_, err = toString(true)
	if err == nil {
		t.Error("expected error for bool, got nil")
	}
}

func TestValue_toInt64(t *testing.T) {
	tests := []struct {
		v    any
		want int64
	}{
		{int64(123), 123},
		{int(456), 456},
		{uint64(789), 789},
		{uint(101), 101},
		{int32(202), 202},
		{uint32(303), 303},
		{"1024", 1024},
	}
	for _, tt := range tests {
		got, err := toInt64(tt.v)
		if err != nil {
			t.Errorf("toInt64(%v): unexpected error: %v", tt.v, err)
			continue
		}
		if got != tt.want {
			t.Errorf("toInt64(%v) = %d, want %d", tt.v, got, tt.want)
		}
	}

	// Overflow.
	_, err := toInt64(uint64(0xFFFFFFFFFFFFFFFF))
	if err == nil {
		t.Error("expected overflow error for max uint64, got nil")
	}

	// Invalid.
	_, err = toInt64(3.14)
	if err == nil {
		t.Error("expected error for float64, got nil")
	}
}
