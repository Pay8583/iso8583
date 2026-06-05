package iso8583

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/Pay8583/iso8583/mac"
	"github.com/Pay8583/iso8583/spec"
)

func testMACProtocol() *spec.Protocol {
	return &spec.Protocol{
		Name:   "test-mac",
		MTI:    spec.ASCIIMTI,
		Bitmap: spec.HexBitmap,
		Sign:   &spec.SignConfig{MACLength: 8, MACField: 64},
		Fields: []spec.Field{
			{Name: "F2", Len: spec.LVAR(19, nil), Valid: spec.N, Value: spec.ASCII},
			{Name: "F64", Len: spec.Fixed(16, 0x00), Valid: spec.B, Value: spec.Hex},
		},
	}
}

func TestComputeMAC_And_CheckMAC(t *testing.T) {
	p := testMACProtocol()

	var buf bytes.Buffer
	w := NewWriter(p, &buf)
	w.WriteMTI(0x0200)
	w.WriteString(2, "4000001234567890")
	w.WriteBytes(64, make([]byte, 8)) // MAC placeholder (8 bytes)
	w.Close()

	signer := &mac.HMACSigner{Key: []byte("test-key-123456"), Hash: "sha256"}

	// Compute MAC — uses Protocol.Sign for MACLength=8.
	msg, macBytes, err := ComputeMAC(w, signer)
	if err != nil {
		t.Fatalf("ComputeMAC: %v", err)
	}
	if len(macBytes) == 0 {
		t.Error("MAC is empty")
	}

	// Check MAC — uses Protocol.Sign for MACLength=8.
	if err := CheckMAC(msg, p, signer); err != nil {
		t.Errorf("CheckMAC failed: %v", err)
	}

	// Tampered message should fail.
	tampered := make([]byte, len(msg))
	copy(tampered, msg)
	tampered[5] ^= 0xFF
	if err := CheckMAC(tampered, p, signer); err == nil {
		t.Error("CheckMAC should fail on tampered data")
	}
}

func TestComputeMAC_TooShort(t *testing.T) {
	p := &spec.Protocol{
		Name:   "test-short",
		MTI:    spec.ASCIIMTI,
		Bitmap: spec.HexBitmap,
		Fields: []spec.Field{},
	}

	var buf bytes.Buffer
	w := NewWriter(p, &buf)
	w.WriteMTI(0x0200)
	w.Close()

	// Placeholder larger than message.
	signer := &mac.HMACSigner{Key: []byte("key"), Hash: "sha256"}
	_, _, err := ComputeMAC(w, signer, WithMACLength(100))
	if err == nil {
		t.Error("expected error for placeholder larger than message")
	}
}

func TestCheckMAC_TooShort(t *testing.T) {
	p := &spec.Protocol{Name: "test"}
	signer := &mac.HMACSigner{Key: []byte("key"), Hash: "sha256"}
	if err := CheckMAC([]byte{0x00}, p, signer, WithMACLength(8)); err == nil {
		t.Error("expected error for data shorter than MAC length")
	}
}

func TestWriter_BinaryMTI(t *testing.T) {
	p := &spec.Protocol{
		Name:   "test-bin-mti",
		MTI:    spec.BinaryMTI,
		Bitmap: spec.HexBitmap,
		Fields: []spec.Field{},
	}

	var buf bytes.Buffer
	w := NewWriter(p, &buf)
	w.WriteMTI(0x80) // pepiso-style MTI
	w.Close()

	raw := w.Bytes()
	if len(raw) < 1 {
		t.Fatal("empty output")
	}
	if raw[0] != 0x80 {
		t.Errorf("MTI byte = %#x, want 0x80", raw[0])
	}

	// Read back.
	r := NewReader(p, bytes.NewReader(raw))
	mti, err := r.ReadMTI()
	if err != nil {
		t.Fatalf("ReadMTI: %v", err)
	}
	if mti != 0x80 {
		t.Errorf("MTI = %#x, want 0x80", mti)
	}
}

func TestWriter_BinaryMTI_Overflow(t *testing.T) {
	p := &spec.Protocol{
		Name:   "test-overflow",
		MTI:    spec.BinaryMTI,
		Bitmap: spec.HexBitmap,
		Fields: []spec.Field{},
	}

	var buf bytes.Buffer
	w := NewWriter(p, &buf)
	w.WriteMTI(0x100) // > 1 byte, should error
	if err := w.Close(); err == nil {
		t.Error("expected error for MTI > 0xFF with BinaryMTI")
	}
}

func TestProtocol_EffectiveBitmapWidth(t *testing.T) {
	p := &spec.Protocol{Fields: make([]spec.Field, 10)}
	if p.EffectiveBitmapWidth() != 8 {
		t.Errorf("10 fields: EffectiveBitmapWidth = %d, want 8", p.EffectiveBitmapWidth())
	}
	p2 := &spec.Protocol{Fields: make([]spec.Field, 70)}
	if p2.EffectiveBitmapWidth() != 16 {
		t.Errorf("70 fields: EffectiveBitmapWidth = %d, want 16", p2.EffectiveBitmapWidth())
	}
	p3 := &spec.Protocol{BitmapWidth: 5, Fields: make([]spec.Field, 40)}
	if p3.EffectiveBitmapWidth() != 5 {
		t.Errorf("BitmapWidth=5: EffectiveBitmapWidth = %d, want 5", p3.EffectiveBitmapWidth())
	}
}

func TestWriter_PepisoStyleProtocol(t *testing.T) {
	p := &spec.Protocol{
		Name:        "pepiso-style",
		MTI:         spec.BinaryMTI,
		Bitmap:      spec.BinaryBitmap,
		BitmapWidth: 5,
		Sign:        &spec.SignConfig{MACLength: 4, MACField: 40},
		Fields:      makeFields(40),
	}

	var buf bytes.Buffer
	w := NewWriter(p, &buf)
	w.WriteMTI(0x80)
	w.WriteString(2, "1234567890")
	w.WriteInt(4, 9999)
	w.Close()

	raw := w.Bytes()
	if len(raw) < 6 {
		t.Errorf("expected at least 6 bytes, got %d", len(raw))
	}

	r := NewReader(p, bytes.NewReader(raw))
	mti, err := r.ReadMTI()
	if err != nil {
		t.Fatalf("ReadMTI: %v", err)
	}
	if mti != 0x80 {
		t.Errorf("MTI = %#x, want 0x80", mti)
	}

	present, err := r.PresentFields()
	if err != nil {
		t.Fatalf("PresentFields: %v", err)
	}
	if len(present) != 2 {
		t.Errorf("got %d present fields, want 2", len(present))
	}
}

func makeFields(n int) []spec.Field {
	fields := make([]spec.Field, n)
	for i := range fields {
		fields[i] = spec.Field{
			Name:  fmt.Sprintf("F%d", i+2),
			Len:   spec.Fixed(10, ' '),
			Valid: spec.ANS,
			Value: spec.ASCII,
		}
	}
	return fields
}

func TestMTIEncoder_RoundTrip(t *testing.T) {
	tests := []struct {
		name string
		enc  spec.MTIEncoder
		mti  uint
	}{
		{"ASCII 0x0200", spec.ASCIIMTI, 0x0200},
		{"ASCII 0x0430", spec.ASCIIMTI, 0x0430},
		{"BCD 0x0200", spec.BCDMTI, 0x0200},
		{"BCD 0x0430", spec.BCDMTI, 0x0430},
		{"Binary 0x80", spec.BinaryMTI, 0x80},
		{"Binary 0x00", spec.BinaryMTI, 0x00},
		{"Binary 0xFF", spec.BinaryMTI, 0xFF},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := tt.enc.Encode(tt.mti)
			if err != nil {
				t.Fatalf("Encode: %v", err)
			}
			got, err := tt.enc.Decode(b)
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}
			if got != tt.mti {
				t.Errorf("round-trip: got %#x, want %#x", got, tt.mti)
			}
		})
	}
}

func TestSignOptions(t *testing.T) {
	// Test that functional options compose correctly.
	p := &spec.Protocol{
		Name: "test-sign",
		Sign: &spec.SignConfig{MACLength: 8, MACField: 64},
	}

	cfg := signConfig(p, []SignOption{WithMACLength(4), WithMTI()})
	if cfg.MACLength != 4 {
		t.Errorf("MACLength = %d, want 4 (overridden)", cfg.MACLength)
	}
	if cfg.MACField != 64 {
		t.Errorf("MACField = %d, want 64 (inherited)", cfg.MACField)
	}
	if !cfg.IncludeMTI {
		t.Error("IncludeMTI should be true")
	}
}
