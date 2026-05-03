package encoding

import "testing"

func TestEBCDIC_EncodeDecode(t *testing.T) {
	e := MustGet("ebcdic")
	input := "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	enc, err := e.Encode(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(enc) != len(input) {
		t.Errorf("encode length mismatch: %d != %d", len(enc), len(input))
	}
	t.Logf("EBCDIC(%q) = %X", input, enc)

	dec, err := e.Decode(enc)
	if err != nil {
		t.Fatal(err)
	}
	if dec != input {
		t.Errorf("roundtrip: got %q, want %q", dec, input)
	}
}

func TestEBCDIC_Hello(t *testing.T) {
	e := MustGet("ebcdic")
	enc, err := e.Encode("HELLO")
	if err != nil {
		t.Fatal(err)
	}
	// Known EBCDIC for "HELLO": C8 C5 D3 D3 D6
	expected := []byte{0xC8, 0xC5, 0xD3, 0xD3, 0xD6}
	if string(enc) != string(expected) {
		t.Errorf("Encode(HELLO) = %X, want %X", enc, expected)
	}
}

func TestEBCDIC_Empty(t *testing.T) {
	e := MustGet("ebcdic")
	if _, err := e.Encode(""); err == nil {
		t.Error("expected error for empty value")
	}
	if _, err := e.Decode(nil); err == nil {
		t.Error("expected error for empty data")
	}
}

func TestEBCDIC_NonAscii(t *testing.T) {
	e := MustGet("ebcdic")
	_, err := e.Encode("hello\xFF")
	if err == nil {
		t.Error("expected error for non-ASCII character")
	}
}

// ── Benchmarks ──────────────────────────────────────────────────────────────────

func BenchmarkEBCDIC_Encode_10B(b *testing.B) {
	e := MustGet("ebcdic")
	s := "0123456789"
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		e.Encode(s)
	}
}

func BenchmarkEBCDIC_Decode_10B(b *testing.B) {
	e := MustGet("ebcdic")
	data, _ := e.Encode("0123456789")
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		e.Decode(data)
	}
}
