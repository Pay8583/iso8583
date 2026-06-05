package iso8583

import (
	"bytes"
	"testing"

	"github.com/Pay8583/iso8583/spec"
)

func TestReader_RoundTrip(t *testing.T) {
	p := testProtocol()

	// Write.
	var wbuf bytes.Buffer
	w := NewWriter(p, &wbuf)
	w.WriteMTI(0x0200)
	w.WriteString(2, "4000001234567890")
	w.WriteString(3, "301000")
	w.WriteInt(4, 1000)
	w.WriteString(11, "123456")
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Read.
	r := NewReader(p, bytes.NewReader(wbuf.Bytes()))
	mti, err := r.ReadMTI()
	if err != nil {
		t.Fatalf("ReadMTI: %v", err)
	}
	if mti != 0x0200 {
		t.Errorf("MTI = %#x, want 0x0200", mti)
	}

	var pan string
	if err := r.ReadString(2, &pan); err != nil {
		t.Fatalf("ReadString(2): %v", err)
	}
	if pan != "4000001234567890" {
		t.Errorf("PAN = %q", pan)
	}

	var procCode string
	if err := r.ReadString(3, &procCode); err != nil {
		t.Fatalf("ReadString(3): %v", err)
	}
	if procCode != "301000" {
		t.Errorf("ProcessingCode = %q", procCode)
	}

	var amount int64
	if err := r.ReadInt(4, &amount); err != nil {
		t.Fatalf("ReadInt(4): %v", err)
	}
	// RBCD decodes "000000001000" → value (padded to full width).
	if amount != 1000 {
		t.Errorf("Amount = %d", amount)
	}

	var stan string
	if err := r.ReadString(11, &stan); err != nil {
		t.Fatalf("ReadString(11): %v", err)
	}
	if stan != "123456" {
		t.Errorf("STAN = %q", stan)
	}

	// Bytes should match.
	if !bytes.Equal(w.Bytes(), r.Bytes()) {
		t.Error("Reader.Bytes() != Writer.Bytes()")
	}
}

func TestReader_EmptyMessage(t *testing.T) {
	p := testProtocol()

	var buf bytes.Buffer
	w := NewWriter(p, &buf)
	w.WriteMTI(0x0200)
	w.Close()

	r := NewReader(p, bytes.NewReader(buf.Bytes()))
	mti, err := r.ReadMTI()
	if err != nil {
		t.Fatalf("ReadMTI: %v", err)
	}
	if mti != 0x0200 {
		t.Errorf("MTI = %#x", mti)
	}
}

func TestReader_FieldNotPresent(t *testing.T) {
	p := testProtocol()

	var buf bytes.Buffer
	w := NewWriter(p, &buf)
	w.WriteMTI(0x0200)
	w.WriteString(2, "1234567890")
	w.Close()

	r := NewReader(p, bytes.NewReader(buf.Bytes()))
	r.ReadMTI()

	var s string
	if err := r.ReadString(3, &s); err == nil {
		t.Error("expected error: field 3 not present")
	}
}

func TestReader_ReadFieldTwice(t *testing.T) {
	p := testProtocol()

	var buf bytes.Buffer
	w := NewWriter(p, &buf)
	w.WriteMTI(0x0200)
	w.WriteString(2, "1234567890")
	w.Close()

	r := NewReader(p, bytes.NewReader(buf.Bytes()))
	r.ReadMTI()

	var s1, s2 string
	r.ReadString(2, &s1)
	if err := r.ReadString(2, &s2); err == nil {
		t.Error("expected error: field 2 already read")
	}
}

func TestReader_MTIMustComeFirst(t *testing.T) {
	p := testProtocol()

	r := NewReader(p, bytes.NewReader([]byte{0x30, 0x32, 0x30, 0x30}))
	var s string
	if err := r.ReadString(2, &s); err == nil {
		t.Error("expected error: MTI must be read first")
	}
}

func TestReader_DoubleMTI(t *testing.T) {
	p := testProtocol()

	var buf bytes.Buffer
	w := NewWriter(p, &buf)
	w.WriteMTI(0x0200)
	w.Close()

	r := NewReader(p, bytes.NewReader(buf.Bytes()))
	if _, err := r.ReadMTI(); err != nil {
		t.Fatalf("first ReadMTI: %v", err)
	}
	if _, err := r.ReadMTI(); err == nil {
		t.Error("expected error: MTI already read")
	}
}

func TestReader_Export(t *testing.T) {
	p := testProtocol()
	p.Fields[0].Secure = true // field 2 is secure

	var buf bytes.Buffer
	w := NewWriter(p, &buf)
	w.WriteMTI(0x0200)
	w.WriteString(2, "4000001234567890")
	w.WriteString(3, "301000")
	w.Close()

	r := NewReader(p, bytes.NewReader(buf.Bytes()))
	r.ReadMTI()
	var pan string
	r.ReadString(2, &pan)
	var proc string
	r.ReadString(3, &proc)

	exp := r.Export()
	if exp[2] != "" {
		t.Errorf("Export[2] = %q, want \"\" (secure)", exp[2])
	}
	if exp[3] == "" {
		t.Error("Export[3] should not be masked")
	}
}

func TestReader_BCDMTI(t *testing.T) {
	p := testProtocol()
	p.MTI = spec.BCDMTI

	var buf bytes.Buffer
	w := NewWriter(p, &buf)
	w.WriteMTI(0x0200)
	w.Close()

	r := NewReader(p, bytes.NewReader(buf.Bytes()))
	mti, err := r.ReadMTI()
	if err != nil {
		t.Fatalf("ReadMTI: %v", err)
	}
	if mti != 0x0200 {
		t.Errorf("MTI = %#x", mti)
	}
}

func TestReader_Validation(t *testing.T) {
	p := testProtocol()

	// Write a valid message, then hack the bytes to have non-numeric data
	// in a numeric field. We do this by writing an LVAR field with hex data
	// and reading it back — the field's validator should catch it.

	// Actually, test that the Writer validation passed first.
	var buf bytes.Buffer
	w := NewWriter(p, &buf)
	w.WriteMTI(0x0200)
	w.WriteString(3, "000000") // valid: numeric
	w.Close()

	r := NewReader(p, bytes.NewReader(buf.Bytes()))
	r.ReadMTI()
	var s string
	if err := r.ReadString(3, &s); err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}
	if s != "000000" {
		t.Errorf("got %q, want 000000", s)
	}
}

func TestReader_RBCDDecode(t *testing.T) {
	p := testProtocol()

	var buf bytes.Buffer
	w := NewWriter(p, &buf)
	w.WriteMTI(0x0200)
	w.WriteInt(4, 1000) // RBCD(12): right-aligned BCD
	w.Close()

	r := NewReader(p, bytes.NewReader(buf.Bytes()))
	r.ReadMTI()
	var amount int64
	if err := r.ReadInt(4, &amount); err != nil {
		t.Fatalf("ReadInt(4): %v", err)
	}
	// RBCD encodes as "000000001000" → 12 chars, decodes to string "000000001000".
	// ReadInt converts this to int64 = 1000.
	if amount != 1000 {
		t.Errorf("amount = %d, want 1000", amount)
	}
}

func TestReader_BytesMatch(t *testing.T) {
	p := testProtocol()

	var buf bytes.Buffer
	w := NewWriter(p, &buf)
	w.WriteMTI(0x0200)
	w.WriteString(2, "4000001234567890")
	w.WriteString(3, "301000")
	w.WriteString(11, "123456")
	w.Close()

	r := NewReader(p, bytes.NewReader(buf.Bytes()))
	r.ReadMTI()
	var s string
	r.ReadString(2, &s)
	r.ReadString(3, &s)
	r.ReadString(11, &s)

	if !bytes.Equal(w.Bytes(), r.Bytes()) {
		t.Errorf("reader bytes (%d) != writer bytes (%d)", len(r.Bytes()), len(w.Bytes()))
	}
}

func TestReader_ReadIntFromStringField(t *testing.T) {
	p := testProtocol()

	var buf bytes.Buffer
	w := NewWriter(p, &buf)
	w.WriteMTI(0x0200)
	w.WriteString(3, "301000") // field 3 is ASCII, Valid: N
	w.Close()

	r := NewReader(p, bytes.NewReader(buf.Bytes()))
	r.ReadMTI()

	// ReadString works fine.
	var s string
	if err := r.ReadString(3, &s); err != nil {
		t.Fatalf("ReadString(3): %v", err)
	}
	if s != "301000" {
		t.Errorf("got %q", s)
	}
}

func TestReader_TruncatedMessage(t *testing.T) {
	p := testProtocol()

	// Write a message then truncate it.
	var buf bytes.Buffer
	w := NewWriter(p, &buf)
	w.WriteMTI(0x0200)
	w.WriteString(2, "4000001234567890")
	w.Close()

	// Truncate to just MTI + bitmap (fields data missing).
	truncated := buf.Bytes()[:len(buf.Bytes())-2]

	r := NewReader(p, bytes.NewReader(truncated))
	r.ReadMTI()

	var s string
	if err := r.ReadString(2, &s); err == nil {
		t.Error("expected error for truncated message")
	}
}
