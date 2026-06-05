package iso8583

import (
	"bytes"
	"testing"

	"github.com/Pay8583/iso8583/encoding"
	"github.com/Pay8583/iso8583/spec"
)

func testProtocol() *spec.Protocol {
	return &spec.Protocol{
		Name:   "test",
		MTI:    spec.ASCIIMTI,
		Bitmap: spec.HexBitmap,
		Fields: []spec.Field{
			{Name: "F2", Len: spec.LVAR(19, encoding.MustGet("bcd")), Valid: spec.N, Value: spec.ASCII},      // 2
			{Name: "F3", Len: spec.Fixed(6, '0'), Valid: spec.N, Value: spec.ASCII},                           // 3
			{Name: "F4", Len: spec.Fixed(12, '0'), Valid: spec.N, Value: spec.RBCD(12)},                       // 4
			{Name: "F5", Len: spec.Fixed(12, '0'), Valid: spec.N, Value: spec.ASCII},                          // 5
			{Name: "F6", Len: spec.Fixed(12, '0'), Valid: spec.N, Value: spec.ASCII},                          // 6
			{Name: "F7", Len: spec.Fixed(10, '0'), Valid: spec.N, Value: spec.ASCII},                          // 7
			{Name: "F8", Len: spec.Fixed(8, '0'), Valid: spec.N, Value: spec.ASCII},                           // 8
			{Name: "F9", Len: spec.Fixed(8, '0'), Valid: spec.N, Value: spec.ASCII},                           // 9
			{Name: "F10", Len: spec.Fixed(8, '0'), Valid: spec.N, Value: spec.ASCII},                          // 10
			{Name: "F11", Len: spec.Fixed(6, '0'), Valid: spec.N, Value: spec.ASCII},                          // 11
		},
	}
}

func TestWriter_Empty(t *testing.T) {
	p := testProtocol()
	var buf bytes.Buffer
	w := NewWriter(p, &buf)
	w.WriteMTI(0x0200)
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	raw := w.Bytes()
	if len(raw) == 0 {
		t.Error("Bytes() is empty")
	}
	if len(buf.Bytes()) == 0 {
		t.Error("nothing written to writer")
	}
	// MTI (4) + bitmap (8) = 12 bytes minimum.
	if len(raw) < 12 {
		t.Errorf("expected at least 12 bytes, got %d", len(raw))
	}
}

func TestWriter_BasicFields(t *testing.T) {
	p := testProtocol()
	var buf bytes.Buffer
	w := NewWriter(p, &buf)

	w.WriteMTI(0x0200)
	w.WriteString(2, "4000001234567890")
	w.WriteString(3, "301000")
	w.WriteInt(4, 1000)
	w.WriteString(11, "123456")

	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	raw := w.Bytes()
	if len(raw) == 0 {
		t.Error("no bytes")
	}

	// Output should be identical to backing writer.
	if !bytes.Equal(raw, buf.Bytes()) {
		t.Error("Bytes() != buf.Bytes()")
	}
}

func TestWriter_DuplicateField(t *testing.T) {
	p := testProtocol()
	var buf bytes.Buffer
	w := NewWriter(p, &buf)

	w.WriteMTI(0x0200)
	w.WriteString(2, "1234567890")
	if err := w.WriteString(2, "9999999999"); err == nil {
		t.Error("expected error writing duplicate field")
	}
}

func TestWriter_FieldOutOfRange(t *testing.T) {
	p := testProtocol()
	var buf bytes.Buffer
	w := NewWriter(p, &buf)

	if err := w.WriteString(1, "x"); err == nil {
		t.Error("expected error for field 1 (out of range)")
	}
	if err := w.WriteString(200, "x"); err == nil {
		t.Error("expected error for field 200 (out of range)")
	}
}

func TestWriter_MTIRequired(t *testing.T) {
	p := testProtocol()
	var buf bytes.Buffer
	w := NewWriter(p, &buf)

	// Close without MTI.
	if err := w.Close(); err == nil {
		t.Error("expected error: MTI not set")
	}
}

func TestWriter_DoubleMTI(t *testing.T) {
	p := testProtocol()
	var buf bytes.Buffer
	w := NewWriter(p, &buf)

	w.WriteMTI(0x0200)
	if err := w.WriteMTI(0x0210); err == nil {
		t.Error("expected error for double MTI")
	}
}

func TestWriter_Validation(t *testing.T) {
	p := testProtocol()
	var buf bytes.Buffer
	w := NewWriter(p, &buf)

	w.WriteMTI(0x0200)
	// Field 3 has Valid: N (numeric only).
	w.WriteString(3, "ABC") // invalid: non-numeric.

	if err := w.Close(); err == nil {
		t.Error("expected validation error for non-numeric field 3")
	}
}

func TestWriter_CloseAfterClose(t *testing.T) {
	p := testProtocol()
	var buf bytes.Buffer
	w := NewWriter(p, &buf)

	w.WriteMTI(0x0200)
	if err := w.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	// Second close should be a no-op (not an error).
	if err := w.Close(); err != nil {
		t.Errorf("second Close: %v", err)
	}
}

func TestWriter_WriteAfterClose(t *testing.T) {
	p := testProtocol()
	var buf bytes.Buffer
	w := NewWriter(p, &buf)

	w.WriteMTI(0x0200)
	w.Close()

	if err := w.WriteString(2, "test"); err == nil {
		t.Error("expected error writing after close")
	}
	if err := w.WriteInt(4, 100); err == nil {
		t.Error("expected error writing after close")
	}
}

func TestWriter_Export(t *testing.T) {
	p := testProtocol()
	// Mark field 2 as secure.
	p.Fields[0].Secure = true

	var buf bytes.Buffer
	w := NewWriter(p, &buf)

	w.WriteMTI(0x0200)
	w.WriteString(2, "4000001234567890")
	w.WriteString(3, "301000")

	w.Close()

	exp := w.Export()
	// Field 2 (secure) should be masked.
	if exp[2] != "" {
		t.Errorf("Export[2] = %q, want \"\" (secure)", exp[2])
	}
	// Field 3 (non-secure) should be visible.
	if exp[3] == "" {
		t.Error("Export[3] should not be masked")
	}
}

func TestWriter_Export_IntField(t *testing.T) {
	p := testProtocol()
	var buf bytes.Buffer
	w := NewWriter(p, &buf)

	w.WriteMTI(0x0200)
	w.WriteInt(4, 1000)

	w.Close()

	exp := w.Export()
	if exp[4] != "1000" {
		t.Errorf("Export[4] = %q, want %q", exp[4], "1000")
	}
}

func TestWriter_RBCDEncoding(t *testing.T) {
	p := testProtocol()
	var buf bytes.Buffer
	w := NewWriter(p, &buf)

	w.WriteMTI(0x0200)
	// Field 4 uses RBCD(12): right-aligned BCD.
	w.WriteInt(4, 1000) // "1000" → "000000001000" → 6 BCD bytes

	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	raw := w.Bytes()
	if len(raw) == 0 {
		t.Error("no bytes")
	}
}

func TestWriter_BCDMTI(t *testing.T) {
	p := testProtocol()
	p.MTI = spec.BCDMTI

	var buf bytes.Buffer
	w := NewWriter(p, &buf)

	w.WriteMTI(0x0200)
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// BCD MTI is 2 bytes instead of 4.
	// Total: 2 (MTI) + 8 (bitmap) = 10 bytes.
	raw := w.Bytes()
	if len(raw) < 10 {
		t.Errorf("expected at least 10 bytes (2 MTI + 8 bitmap), got %d", len(raw))
	}
}

func TestWriter_UndefinedField(t *testing.T) {
	p := testProtocol()
	var buf bytes.Buffer
	w := NewWriter(p, &buf)

	w.WriteMTI(0x0200)
	// Field 64 is not defined in test protocol (only 2,3,4,11).
	w.WriteString(64, "DATA")

	if err := w.Close(); err == nil {
		t.Error("expected error: field 64 not defined")
	}
}

func TestWriter_Bytes_MatchesUnderlyingWriter(t *testing.T) {
	p := testProtocol()
	var buf bytes.Buffer
	w := NewWriter(p, &buf)

	w.WriteMTI(0x0200)
	w.WriteString(2, "1234567890")
	w.WriteString(3, "000000")
	w.WriteInt(4, 9999)
	w.WriteString(11, "000001")

	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if !bytes.Equal(w.Bytes(), buf.Bytes()) {
		t.Errorf("Bytes() (%d bytes) != buf.Bytes() (%d bytes)", len(w.Bytes()), len(buf.Bytes()))
	}
}
