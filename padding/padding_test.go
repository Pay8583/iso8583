package padding

import (
	"bytes"
	"testing"
)

func TestPKCS7_Roundtrip(t *testing.T) {
	p := Get("pkcs7")
	for _, bs := range []int{8, 16} {
		for _, data := range [][]byte{
			[]byte("hello"),
			[]byte("12345678"),
			[]byte("123456789012345"),
			make([]byte, 0),
			[]byte("1234567890123456"),
		} {
			padded, err := p.Pad(data, bs)
			if err != nil {
				t.Fatalf("pkcs7 pad (block=%d, len=%d): %v", bs, len(data), err)
			}
			if len(padded)%bs != 0 {
				t.Errorf("pkcs7 padded len %% %d != 0: %d", bs, len(padded))
			}
			unpadded, err := p.Unpad(padded, bs)
			if err != nil {
				t.Fatalf("pkcs7 unpad: %v", err)
			}
			if !bytes.Equal(unpadded, data) {
				t.Errorf("pkcs7 roundtrip mismatch: got %q, want %q", unpadded, data)
			}
		}
	}
}

func TestISO9797M1_Roundtrip(t *testing.T) {
	p := Get("iso9797-1")
	for _, data := range [][]byte{
		[]byte("hello"),
		[]byte("12345678"),
	} {
		padded, err := p.Pad(data, 8)
		if err != nil {
			t.Fatal(err)
		}
		if len(padded)%8 != 0 {
			t.Errorf("padded len not multiple of 8: %d", len(padded))
		}
		unpadded, err := p.Unpad(padded, 8)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(unpadded, data) {
			t.Errorf("roundtrip mismatch")
		}
	}
}

func TestISO9797M2_Roundtrip(t *testing.T) {
	p := Get("iso9797-2")
	for _, data := range [][]byte{
		[]byte("hello"),
		[]byte("12345678"),
		[]byte{},
	} {
		padded, err := p.Pad(data, 8)
		if err != nil {
			t.Fatal(err)
		}
		unpadded, err := p.Unpad(padded, 8)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(unpadded, data) {
			t.Errorf("roundtrip mismatch: got %q, want %q", unpadded, data)
		}
	}
}

func TestISO10126_Roundtrip(t *testing.T) {
	p := Get("iso10126")
	data := []byte("hello world")
	padded, err := p.Pad(data, 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(padded)%8 != 0 {
		t.Errorf("padded len not multiple of 8")
	}
	unpadded, err := p.Unpad(padded, 8)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(unpadded, data) {
		t.Errorf("roundtrip mismatch")
	}
}

func TestZero_Roundtrip(t *testing.T) {
	p := Get("zero")
	data := []byte("test")
	padded, err := p.Pad(data, 8)
	if err != nil {
		t.Fatal(err)
	}
	unpadded, err := p.Unpad(padded, 8)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(unpadded, data) {
		t.Errorf("roundtrip mismatch")
	}
}

func TestPKCS7_InvalidUnpad(t *testing.T) {
	p := Get("pkcs7")
	// Not a multiple of block size.
	_, err := p.Unpad([]byte{1, 2, 3}, 8)
	if err == nil {
		t.Error("expected error for non-aligned data")
	}
	// Invalid pad value.
	_, err = p.Unpad([]byte{0, 0, 0, 0, 0, 0, 0, 9}, 8)
	if err == nil {
		t.Error("expected error for invalid pad byte")
	}
}

// ── Benchmarks ──────────────────────────────────────────────────────────────────

func BenchmarkPKCS7_Pad_1KB(b *testing.B) {
	p := Get("pkcs7")
	data := make([]byte, 1024)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		p.Pad(data, 8)
	}
}

func BenchmarkPKCS7_Unpad_1KB(b *testing.B) {
	p := Get("pkcs7")
	padded, _ := p.Pad(make([]byte, 1024), 8)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		p.Unpad(padded, 8)
	}
}
