package encoding

import "testing"

func TestBinary_EncodeDecode(t *testing.T) {
	e := MustGet("binary")
	data := []byte{0x00, 0x01, 0xFF, 0xFE, 0x80, 0x7F}
	enc, err := e.Encode(string(data))
	if err != nil {
		t.Fatal(err)
	}
	if string(enc) != string(data) {
		t.Errorf("encode: got %x, want %x", enc, data)
	}

	dec, err := e.Decode(enc)
	if err != nil {
		t.Fatal(err)
	}
	if dec != string(data) {
		t.Errorf("decode: got %x, want %x", []byte(dec), data)
	}
}

func TestBinary_Empty(t *testing.T) {
	e := MustGet("binary")
	if _, err := e.Encode(""); err == nil {
		t.Error("expected error for empty value")
	}
	if _, err := e.Decode(nil); err == nil {
		t.Error("expected error for empty data")
	}
}

// ── Benchmarks ──────────────────────────────────────────────────────────────────

func BenchmarkBinary_Encode_8B(b *testing.B) {
	e := MustGet("binary")
	s := "\x00\x01\x02\x03\x04\x05\x06\x07"
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		e.Encode(s)
	}
}

func BenchmarkBinary_Decode_8B(b *testing.B) {
	e := MustGet("binary")
	data := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		e.Decode(data)
	}
}
