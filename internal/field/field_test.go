package field

import (
	"bytes"
	"testing"
)

func TestReadFixed(t *testing.T) {
	raw, err := ReadFixed([]byte{1, 2, 3, 4, 5}, 3)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(raw, []byte{1, 2, 3}) {
		t.Errorf("ReadFixed: got %x, want 010203", raw)
	}
}

func TestReadFixed_Overflow(t *testing.T) {
	_, err := ReadFixed([]byte{1, 2}, 5)
	if err == nil {
		t.Error("expected error for insufficient data")
	}
}

func TestReadLLVAR(t *testing.T) {
	// Length byte = 3, then 3 data bytes.
	raw, consumed, err := ReadLLVAR([]byte{3, 'a', 'b', 'c', 'x', 'y'})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(raw, []byte("abc")) {
		t.Errorf("ReadLLVAR: got %q", raw)
	}
	if consumed != 4 {
		t.Errorf("consumed = %d, want 4", consumed)
	}
}

func TestReadLLVAR_Overflow(t *testing.T) {
	_, _, err := ReadLLVAR([]byte{5, 'a'})
	if err == nil {
		t.Error("expected overflow error")
	}
}

func TestReadLLLVAR(t *testing.T) {
	// 2-byte length: 0x00 0x03 = 3, then 3 bytes.
	data := []byte{0, 3, 'x', 'y', 'z', 'e', 'x', 't', 'r', 'a'}
	raw, consumed, err := ReadLLLVAR(data)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(raw, []byte("xyz")) {
		t.Errorf("ReadLLLVAR: got %q", raw)
	}
	if consumed != 5 {
		t.Errorf("consumed = %d, want 5", consumed)
	}
}

func TestReadLLLVAR_Max(t *testing.T) {
	// 2-byte length: 0x03 0xE7 = 999.
	data := make([]byte, 2+999)
	data[0] = 0x03
	data[1] = 0xE7
	raw, consumed, err := ReadLLLVAR(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) != 999 {
		t.Errorf("length = %d, want 999", len(raw))
	}
	if consumed != 1001 {
		t.Errorf("consumed = %d, want 1001", consumed)
	}
}

func TestReadLLLLVAR(t *testing.T) {
	// 3-byte length: 0x00 0x00 0x04 = 4.
	data := []byte{0, 0, 4, 'd', 'a', 't', 'a'}
	raw, consumed, err := ReadLLLLVAR(data)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(raw, []byte("data")) {
		t.Errorf("ReadLLLLVAR: got %q", raw)
	}
	if consumed != 7 {
		t.Errorf("consumed = %d, want 7", consumed)
	}
}

// ── Writer roundtrip ─────────────────────────────────────────────────────────────

func TestWriteRead_Fixed(t *testing.T) {
	buf := WriteFixed(nil, []byte("12"), 6)
	if !bytes.Equal(buf, []byte("12\x00\x00\x00\x00")) {
		t.Errorf("WriteFixed: got %x", buf)
	}
	raw, err := ReadFixed(buf, 6)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) != 6 {
		t.Errorf("ReadFixed: length = %d, want 6", len(raw))
	}
}

func TestWriteRead_LLVAR(t *testing.T) {
	buf := WriteLLVAR(nil, []byte("hello"))
	raw, consumed, err := ReadLLVAR(buf)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(raw, []byte("hello")) {
		t.Errorf("roundtrip: got %q", raw)
	}
	if consumed != len(buf) {
		t.Errorf("consumed mismatch")
	}
}

func TestWriteRead_LLLVAR(t *testing.T) {
	buf := WriteLLLVAR(nil, []byte("world"))
	raw, consumed, err := ReadLLLVAR(buf)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(raw, []byte("world")) {
		t.Errorf("roundtrip: got %q", raw)
	}
	if consumed != len(buf) {
		t.Errorf("consumed mismatch")
	}
}

func TestWriteRead_LLLLVAR(t *testing.T) {
	buf := WriteLLLLVAR(nil, []byte("test"))
	raw, consumed, err := ReadLLLLVAR(buf)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(raw, []byte("test")) {
		t.Errorf("roundtrip: got %q", raw)
	}
	if consumed != len(buf) {
		t.Errorf("consumed mismatch")
	}
}
