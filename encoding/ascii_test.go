package encoding

import "testing"

func TestASCII_Encode(t *testing.T) {
	e := MustGet("ascii")
	got, err := e.Encode("hello")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello" {
		t.Errorf("Encode = %q, want %q", got, "hello")
	}
}

func TestASCII_Decode(t *testing.T) {
	e := MustGet("ascii")
	got, err := e.Decode([]byte("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello" {
		t.Errorf("Decode = %q, want %q", got, "hello")
	}
}

func TestASCII_Empty(t *testing.T) {
	e := MustGet("ascii")
	if _, err := e.Encode(""); err == nil {
		t.Error("expected error for empty value")
	}
	if _, err := e.Decode(nil); err == nil {
		t.Error("expected error for empty data")
	}
}

// ── Benchmarks ──────────────────────────────────────────────────────────────────

func BenchmarkASCII_Encode_100B(b *testing.B) {
	e := MustGet("ascii")
	s := "1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890"
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		e.Encode(s)
	}
}

func BenchmarkASCII_Decode_100B(b *testing.B) {
	e := MustGet("ascii")
	data := []byte("1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890")
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		e.Decode(data)
	}
}
