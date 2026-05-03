package encoding

import "testing"

func TestBCD_Encode(t *testing.T) {
	e := MustGet("bcd")
	tests := []struct {
		input string
		want  []byte
	}{
		{"00", []byte{0x00}},
		{"12", []byte{0x12}},
		{"99", []byte{0x99}},
		{"123", []byte{0x01, 0x23}},  // odd length: left-pad zero nibble
		{"1234", []byte{0x12, 0x34}},
		{"0", []byte{0x00}},
		{"9", []byte{0x09}},
		{"1234567890", []byte{0x12, 0x34, 0x56, 0x78, 0x90}},
	}
	for _, tt := range tests {
		got, err := e.Encode(tt.input)
		if err != nil {
			t.Errorf("Encode(%q): unexpected error: %v", tt.input, err)
			continue
		}
		if string(got) != string(tt.want) {
			t.Errorf("Encode(%q) = %x, want %x", tt.input, got, tt.want)
		}
	}
}

func TestBCD_Encode_Invalid(t *testing.T) {
	e := MustGet("bcd")
	_, err := e.Encode("12A4")
	if err == nil {
		t.Error("expected error for non-digit character")
	}
	_, err = e.Encode("")
	if err == nil {
		t.Error("expected error for empty string")
	}
}

func TestBCD_Decode(t *testing.T) {
	e := MustGet("bcd")
	tests := []struct {
		input []byte
		want  string
	}{
		{[]byte{0x00}, "00"},
		{[]byte{0x12}, "12"},
		{[]byte{0x01, 0x23}, "0123"},
		{[]byte{0x12, 0x34}, "1234"},
		{[]byte{0x09}, "09"},
	}
	for _, tt := range tests {
		got, err := e.Decode(tt.input)
		if err != nil {
			t.Errorf("Decode(%x): unexpected error: %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("Decode(%x) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestBCD_Roundtrip(t *testing.T) {
	e := MustGet("bcd")
	// Only even-length values roundtrip perfectly; odd-length values get a
	// leading zero nibble (e.g. "123" → 0x01 0x23 → "0123").
	inputs := []string{"00", "12", "99", "1234", "1234567890123456"}
	for _, in := range inputs {
		data, err := e.Encode(in)
		if err != nil {
			t.Fatal(err)
		}
		got, err := e.Decode(data)
		if err != nil {
			t.Fatal(err)
		}
		if got != in {
			t.Errorf("roundtrip(%q): got %q", in, got)
		}
	}
}

// ── Benchmarks ──────────────────────────────────────────────────────────────────

func BenchmarkBCD_Encode_6digits(b *testing.B) {
	e := MustGet("bcd")
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		e.Encode("123456")
	}
}

func BenchmarkBCD_Encode_19digits(b *testing.B) {
	e := MustGet("bcd")
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		e.Encode("1234567890123456789")
	}
}

func BenchmarkBCD_Decode_6digits(b *testing.B) {
	e := MustGet("bcd")
	data := []byte{0x12, 0x34, 0x56}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		e.Decode(data)
	}
}

func BenchmarkBCD_Decode_19digits(b *testing.B) {
	e := MustGet("bcd")
	data := []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0x01, 0x23, 0x45, 0x67, 0x89}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		e.Decode(data)
	}
}
