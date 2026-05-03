package iso8583

import (
	"encoding/hex"
	"testing"

	"github.com/Pay8583/iso8583/spec"
)

func TestPackUnpack_Roundtrip(t *testing.T) {
	s := spec.MustGet("1987")

	msg := NewMessage(s, "0200")
	msg.Set(2, "4000001234567890")  // PAN
	msg.Set(3, "301000")            // Processing Code
	msg.Set(4, "000000001000")      // Transaction Amount
	msg.Set(7, "0503143015")        // Transmission Date & Time
	msg.Set(11, "123456")           // STAN
	msg.Set(12, "143015")           // Time Local
	msg.Set(13, "0503")             // Date Local
	msg.Set(35, "4000001234567890=250510100000000") // Track 2
	msg.Set(37, "123456789012")     // Retrieval Ref Number
	msg.Set(41, "TERM0001")         // Terminal ID
	msg.Set(42, "MERCHANT01     ") // Merchant ID (15 chars)
	msg.Set(49, "840")              // Currency Code

	// Pack.
	data, err := msg.Pack()
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}
	t.Logf("packed message (%d bytes): %s", len(data), hex.EncodeToString(data))

	// Unpack.
	unpacked := NewMessage(s, "")
	if err := unpacked.Unpack(data); err != nil {
		t.Fatalf("Unpack: %v", err)
	}

	// Verify MTI.
	if unpacked.MTI != "0200" {
		t.Errorf("MTI = %q, want %q", unpacked.MTI, "0200")
	}

	// Verify fields.
	checks := map[int]string{
		2:  "4000001234567890",
		3:  "301000",
		4:  "000000001000",
		7:  "0503143015",
		11: "123456",
		12: "143015",
		13: "0503",
		35: "4000001234567890=250510100000000",
		37: "123456789012",
		41: "TERM0001",
		42: "MERCHANT01     ",
		49: "840",
	}
	for idx, want := range checks {
		got := unpacked.Get(idx)
		if got != want {
			t.Errorf("field %d: got %q, want %q", idx, got, want)
		}
	}
}

func TestPackUnpack_MinimalMessage(t *testing.T) {
	s := spec.MustGet("1987")
	msg := NewMessage(s, "0100")
	msg.Set(2, "4000001234567890")
	msg.Set(3, "000000")
	msg.Set(4, "000000000000")

	data, err := msg.Pack()
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}

	unpacked := NewMessage(s, "")
	if err := unpacked.Unpack(data); err != nil {
		t.Fatalf("Unpack: %v", err)
	}

	if unpacked.MTI != "0100" {
		t.Errorf("MTI = %q", unpacked.MTI)
	}
	if unpacked.Get(2) != "4000001234567890" {
		t.Errorf("field 2 mismatch")
	}
}

func TestPackMessage_NoSpec(t *testing.T) {
	msg := &Message{MTI: "0200"}
	_, err := msg.Pack()
	if err == nil {
		t.Error("expected error for message with no spec")
	}
}

func TestUnpackMessage_Truncated(t *testing.T) {
	s := spec.MustGet("1987")
	_, err := UnpackMessage([]byte{0x30, 0x31}, s) // only 2 bytes
	if err == nil {
		t.Error("expected error for truncated message")
	}
}

// ── Benchmarks ──────────────────────────────────────────────────────────────────

func BenchmarkPackMessage_10fields(b *testing.B) {
	s := spec.MustGet("1987")
	msg := NewMessage(s, "0200")
	msg.Set(2, "4000001234567890")
	msg.Set(3, "301000")
	msg.Set(4, "000000001000")
	msg.Set(7, "0503143015")
	msg.Set(11, "123456")
	msg.Set(12, "143015")
	msg.Set(13, "0503")
	msg.Set(37, "123456789012")
	msg.Set(41, "TERM0001")
	msg.Set(49, "840")

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		msg.Pack()
	}
}

func BenchmarkUnpackMessage_10fields(b *testing.B) {
	s := spec.MustGet("1987")
	msg := NewMessage(s, "0200")
	msg.Set(2, "4000001234567890")
	msg.Set(3, "301000")
	msg.Set(4, "000000001000")
	msg.Set(7, "0503143015")
	msg.Set(11, "123456")
	msg.Set(12, "143015")
	msg.Set(13, "0503")
	msg.Set(37, "123456789012")
	msg.Set(41, "TERM0001")
	msg.Set(49, "840")
	data, _ := msg.Pack()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		UnpackMessage(data, s)
	}
}
