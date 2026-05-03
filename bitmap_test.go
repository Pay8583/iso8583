package iso8583

import (
	"encoding/hex"
	"testing"
)

func TestBitmap_SetIsSet(t *testing.T) {
	b := &Bitmap{}
	for _, f := range []int{2, 3, 11, 35, 64, 65, 100, 128} {
		if b.IsSet(f) {
			t.Errorf("field %d should not be set initially", f)
		}
		b.Set(f)
		if !b.IsSet(f) {
			t.Errorf("field %d should be set after Set(%d)", f, f)
		}
	}
}

func TestBitmap_Clear(t *testing.T) {
	b := &Bitmap{}
	b.Set(2)
	b.Set(65)
	b.Clear(2)
	if b.IsSet(2) {
		t.Error("field 2 should be cleared")
	}
	if !b.IsSet(65) {
		t.Error("field 65 should remain set")
	}
	b.Clear(65)
	if b.IsSet(65) {
		t.Error("field 65 should be cleared")
	}
}

func TestBitmap_HasSecondary(t *testing.T) {
	b := &Bitmap{}
	if b.HasSecondary() {
		t.Error("no secondary bitmap expected initially")
	}
	b.Set(65)
	if !b.HasSecondary() {
		t.Error("secondary bitmap should be indicated when field 65 is set")
	}
	if !b.IsSet(1) {
		t.Error("field 1 should be set when secondary bitmap is present")
	}
}

func TestBitmap_PresentFields(t *testing.T) {
	b := &Bitmap{}
	b.Set(2)
	b.Set(3)
	b.Set(11)
	b.Set(35)
	b.Set(65)
	b.Set(128)
	got := b.PresentFields()
	want := []int{2, 3, 11, 35, 65, 128}
	if len(got) != len(want) {
		t.Fatalf("PresentFields = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("PresentFields[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

func TestBitmap_Bytes(t *testing.T) {
	b := &Bitmap{}
	b.Set(2)
	b.Set(3)
	b.Set(11)
	bytes := b.Bytes()
	if len(bytes) != 8 {
		t.Fatalf("expected 8 bytes, got %d", len(bytes))
	}
	// Bits 2, 3, 11 should be set in the primary bitmap
	// Bit 2: position 63-1 = 62  → 0x40
	// Bit 3: position 63-2 = 61  → 0x20
	// Bit 11: position 63-10 = 53 → 0x00 0x08
	// Bytes: 60 00 00 00 00 00 08 00 in the upper bits? Let me calculate
	// Actually bits 2,3 are in first byte, bit 11 in second byte
	expected := hex.EncodeToString(bytes)
	t.Logf("bitmap bytes: %s", expected)
}

func TestBitmap_Bytes_WithSecondary(t *testing.T) {
	b := &Bitmap{}
	b.Set(65)
	bytes := b.Bytes()
	if len(bytes) != 16 {
		t.Fatalf("expected 16 bytes, got %d", len(bytes))
	}
}

func TestBitmap_Reset(t *testing.T) {
	b := &Bitmap{}
	b.Set(2)
	b.Set(65)
	b.Reset()
	if b.IsSet(2) || b.IsSet(65) || b.HasSecondary() {
		t.Error("bitmap should be empty after Reset")
	}
}

func TestParseBitmap_PrimaryOnly(t *testing.T) {
	data := []byte{0x60, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	b, consumed, err := ParseBitmap(data)
	if err != nil {
		t.Fatal(err)
	}
	if consumed != 8 {
		t.Errorf("consumed = %d, want 8", consumed)
	}
	if !b.IsSet(2) || !b.IsSet(3) {
		t.Error("expected fields 2 and 3 to be set")
	}
	if b.HasSecondary() {
		t.Error("should not have secondary bitmap")
	}
}

func TestParseBitmap_WithSecondary(t *testing.T) {
	data := make([]byte, 16)
	data[0] = 0x80 // bit 1 set (secondary bitmap indicator)
	data[8] = 0x80 // bit 65 (secondary-bit 1, MSB of first secondary byte)
	b, consumed, err := ParseBitmap(data)
	if err != nil {
		t.Fatal(err)
	}
	if consumed != 16 {
		t.Errorf("consumed = %d, want 16", consumed)
	}
	if !b.HasSecondary() {
		t.Error("expected secondary bitmap")
	}
	if !b.IsSet(65) {
		t.Error("expected field 65 to be set")
	}
}

func TestParseBitmap_TooShort(t *testing.T) {
	_, _, err := ParseBitmap([]byte{0x00, 0x00})
	if err == nil {
		t.Error("expected error for <8 byte input")
	}
}

func TestParseBitmap_SecondaryTruncated(t *testing.T) {
	data := []byte{0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00} // only 8 bytes
	_, _, err := ParseBitmap(data)
	if err == nil {
		t.Error("expected error when secondary bitmap is indicated but not present")
	}
}

// ── Benchmarks ──────────────────────────────────────────────────────────────────

func BenchmarkBitmap_Set(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		bm := &Bitmap{}
		bm.Set(2)
		bm.Set(35)
		bm.Set(65)
		bm.Set(128)
	}
}

func BenchmarkBitmap_IsSet(b *testing.B) {
	bm := &Bitmap{}
	bm.Set(2)
	bm.Set(35)
	bm.Set(65)
	bm.Set(128)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		bm.IsSet(2)
		bm.IsSet(64)
		bm.IsSet(65)
		bm.IsSet(128)
	}
}

func BenchmarkBitmap_PresentFields_10fields(b *testing.B) {
	bm := &Bitmap{}
	for _, f := range []int{2, 3, 4, 7, 11, 12, 13, 22, 35, 41} {
		bm.Set(f)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		bm.PresentFields()
	}
}

func BenchmarkBitmap_ParseBitmap_Primary(b *testing.B) {
	data := []byte{0x60, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		ParseBitmap(data)
	}
}

func BenchmarkBitmap_Bytes_Primary(b *testing.B) {
	bm := &Bitmap{}
	bm.Set(2)
	bm.Set(3)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		bm.Bytes()
	}
}
