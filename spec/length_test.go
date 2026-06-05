package spec

import (
	"bytes"
	"io"
	"testing"

	"github.com/Pay8583/iso8583/encoding"
)

func TestFixed(t *testing.T) {
	f := Fixed(12, '0')

	if !f.IsFixed() {
		t.Error("IsFixed() = false, want true")
	}
	if f.FixedLen() != 12 {
		t.Errorf("FixedLen() = %d, want 12", f.FixedLen())
	}
	if f.MaxLen() != 12 {
		t.Errorf("MaxLen() = %d, want 12", f.MaxLen())
	}
	if f.WireLen() != 0 {
		t.Errorf("WireLen() = %d, want 0", f.WireLen())
	}
	if f.(fixedLen).Pad() != '0' {
		t.Errorf("Pad() = %q, want '0'", f.(fixedLen).Pad())
	}

	// WriteLen is a no-op.
	var buf bytes.Buffer
	if err := f.WriteLen(&buf, 10); err != nil {
		t.Errorf("WriteLen: unexpected error: %v", err)
	}
	if buf.Len() != 0 {
		t.Error("WriteLen wrote bytes, want none")
	}

	// ReadLen is unused for fixed-length.
	n, err := f.ReadLen(nil)
	if err != nil {
		t.Errorf("ReadLen: unexpected error: %v", err)
	}
	if n != 0 {
		t.Errorf("ReadLen() = %d, want 0", n)
	}
}

func TestLVAR_BCD(t *testing.T) {
	l := LVAR(19, encoding.MustGet("bcd"))

	if l.IsFixed() {
		t.Error("IsFixed() = true, want false")
	}
	if l.FixedLen() != 0 {
		t.Errorf("FixedLen() = %d, want 0", l.FixedLen())
	}
	if l.MaxLen() != 19 {
		t.Errorf("MaxLen() = %d, want 19", l.MaxLen())
	}
	if l.WireLen() != 1 {
		t.Errorf("WireLen() = %d, want 1", l.WireLen())
	}

	// Round-trip: write length prefix, read it back.
	var buf bytes.Buffer
	if err := l.WriteLen(&buf, 19); err != nil {
		t.Fatalf("WriteLen(19): %v", err)
	}
	if buf.Len() != 1 {
		t.Fatalf("WriteLen wrote %d bytes, want 1", buf.Len())
	}
	n, err := l.ReadLen(&buf)
	if err != nil {
		t.Fatalf("ReadLen: %v", err)
	}
	if n != 19 {
		t.Errorf("ReadLen() = %d, want 19", n)
	}

	// Single-digit length.
	buf.Reset()
	if err := l.WriteLen(&buf, 5); err != nil {
		t.Fatalf("WriteLen(5): %v", err)
	}
	n, err = l.ReadLen(&buf)
	if err != nil {
		t.Fatalf("ReadLen(5): %v", err)
	}
	if n != 5 {
		t.Errorf("ReadLen() = %d, want 5", n)
	}

	// Max value in 1 BCD byte (need higher max for this content length).
	buf.Reset()
	l99 := LVAR(99, encoding.MustGet("bcd"))
	if err := l99.WriteLen(&buf, 99); err != nil {
		t.Fatalf("WriteLen(99): %v", err)
	}
	n, err = l99.ReadLen(&buf)
	if err != nil {
		t.Fatalf("ReadLen(99): %v", err)
	}
	if n != 99 {
		t.Errorf("ReadLen() = %d, want 99", n)
	}
	_ = n

	// Zero content length should error.
	if err := l.WriteLen(&buf, 0); err == nil {
		t.Error("WriteLen(0): expected error, got nil")
	}
}

func TestLLVAR_BCD(t *testing.T) {
	l := LLVAR(999, encoding.MustGet("bcd"))

	if l.WireLen() != 2 {
		t.Errorf("WireLen() = %d, want 2", l.WireLen())
	}
	if l.MaxLen() != 999 {
		t.Errorf("MaxLen() = %d, want 999", l.MaxLen())
	}

	// Round-trip: 123 bytes.
	var buf bytes.Buffer
	if err := l.WriteLen(&buf, 123); err != nil {
		t.Fatalf("WriteLen(123): %v", err)
	}
	if buf.Len() != 2 {
		t.Fatalf("WriteLen wrote %d bytes, want 2", buf.Len())
	}
	n, err := l.ReadLen(&buf)
	if err != nil {
		t.Fatalf("ReadLen: %v", err)
	}
	if n != 123 {
		t.Errorf("ReadLen() = %d, want 123", n)
	}

	// Max value in 2 BCD bytes.
	buf.Reset()
	if err := l.WriteLen(&buf, 999); err != nil {
		t.Fatalf("WriteLen(999): %v", err)
	}
	n, err = l.ReadLen(&buf)
	if err != nil {
		t.Fatalf("ReadLen(999): %v", err)
	}
	if n != 999 {
		t.Errorf("ReadLen() = %d, want 999", n)
	}
}

func TestLLLVAR_BCD(t *testing.T) {
	l := LLLVAR(9999, encoding.MustGet("bcd"))

	if l.WireLen() != 3 {
		t.Errorf("WireLen() = %d, want 3", l.WireLen())
	}
	if l.MaxLen() != 9999 {
		t.Errorf("MaxLen() = %d, want 9999", l.MaxLen())
	}

	// Round-trip: 1234 bytes.
	var buf bytes.Buffer
	if err := l.WriteLen(&buf, 1234); err != nil {
		t.Fatalf("WriteLen(1234): %v", err)
	}
	if buf.Len() != 3 {
		t.Fatalf("WriteLen wrote %d bytes, want 3", buf.Len())
	}
	n, err := l.ReadLen(&buf)
	if err != nil {
		t.Fatalf("ReadLen: %v", err)
	}
	if n != 1234 {
		t.Errorf("ReadLen() = %d, want 1234", n)
	}

	// Max value in 3 BCD bytes.
	buf.Reset()
	if err := l.WriteLen(&buf, 9999); err != nil {
		t.Fatalf("WriteLen(9999): %v", err)
	}
	n, err = l.ReadLen(&buf)
	if err != nil {
		t.Fatalf("ReadLen(9999): %v", err)
	}
	if n != 9999 {
		t.Errorf("ReadLen() = %d, want 9999", n)
	}
}

func TestLVAR_DefaultsToBCD(t *testing.T) {
	// LVAR with nil encoder defaults to BCD.
	l := LVAR(50, nil)

	var buf bytes.Buffer
	if err := l.WriteLen(&buf, 42); err != nil {
		t.Fatalf("WriteLen(42): %v", err)
	}
	n, err := l.ReadLen(&buf)
	if err != nil {
		t.Fatalf("ReadLen: %v", err)
	}
	if n != 42 {
		t.Errorf("ReadLen() = %d, want 42", n)
	}
}

func TestVarLen_ReadErrors(t *testing.T) {
	l := LVAR(99, encoding.MustGet("bcd"))

	// Empty reader.
	_, err := l.ReadLen(bytes.NewReader(nil))
	if err == nil {
		t.Error("expected error for empty reader, got nil")
	}

	// Truncated reader (only 0 of 2 bytes for LLVAR).
	l2 := LLVAR(999, encoding.MustGet("bcd"))
	_, err = l2.ReadLen(bytes.NewReader([]byte{0x01}))
	if err == nil {
		t.Error("expected error for truncated LLVAR, got nil")
	}

	// Zero-length prefix (not allowed).
	var buf bytes.Buffer
	// Write "00" (zero-length) — this shouldn't happen in practice.
	buf.Write([]byte{0x00})
	_, err = l.ReadLen(&buf)
	if err == nil {
		t.Error("expected error for zero-length prefix, got nil")
	}
}

func TestVarLen_WriteErrors(t *testing.T) {
	l := LVAR(99, encoding.MustGet("bcd"))

	// Zero content length.
	if err := l.WriteLen(io.Discard, 0); err == nil {
		t.Error("WriteLen(0): expected error, got nil")
	}

	// Negative content length.
	if err := l.WriteLen(io.Discard, -1); err == nil {
		t.Error("WriteLen(-1): expected error, got nil")
	}
}

func TestFixed_WithVariousPads(t *testing.T) {
	tests := []struct {
		n   int
		pad byte
	}{
		{6, '0'},
		{8, ' '},
		{16, 0x00},
		{3, 'F'},
	}
	for _, tt := range tests {
		f := Fixed(tt.n, tt.pad)
		if f.FixedLen() != tt.n {
			t.Errorf("Fixed(%d, %q).FixedLen() = %d", tt.n, tt.pad, f.FixedLen())
		}
		if f.(fixedLen).Pad() != tt.pad {
			t.Errorf("Fixed(%d, %q).Pad() = %q", tt.n, tt.pad, f.(fixedLen).Pad())
		}
	}
}
