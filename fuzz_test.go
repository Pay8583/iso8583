package iso8583

import (
	"testing"

	"github.com/Pay8583/iso8583/spec"
)

// FuzzPackUnpack verifies that pack → unpack roundtrip preserves all fields.
func FuzzPackUnpack(f *testing.F) {
	s := spec.MustGet("1987")

	// Seed corpus with a representative message.
	f.Add("0200", "4000001234567890", "301000", "000000001000", "123456", "TERM0001", "840")

	f.Fuzz(func(t *testing.T, mti, pan, procCode, amount, stan, terminalID, currency string) {
		if len(mti) != 4 {
			return
		}
		msg := NewMessage(s, mti)
		if len(pan) > 0 && len(pan) <= 19 {
			msg.Set(2, padNumeric(pan, 0))
		}
		if len(procCode) > 0 && len(procCode) <= 6 {
			msg.Set(3, padNumeric(procCode, 6))
		}
		if len(amount) > 0 && len(amount) <= 12 {
			msg.Set(4, padNumeric(amount, 12))
		}
		if len(stan) > 0 && len(stan) <= 6 {
			msg.Set(11, padNumeric(stan, 6))
		}
		if len(terminalID) > 0 && len(terminalID) <= 8 {
			msg.Set(41, padAlpha(terminalID, 8))
		}
		if len(currency) > 0 && len(currency) <= 3 {
			msg.Set(49, padAlpha(currency, 3))
		}

		data, err := msg.Pack()
		if err != nil {
			t.Skip() // invalid combination, skip
		}

		unpacked := NewMessage(s, "")
		if err := unpacked.Unpack(data); err != nil {
			t.Fatalf("Unpack failed for valid packed message: %v", err)
		}

		for n, v := range msg.Fields {
			got := unpacked.Get(n)
			// BCD encoding pads odd-length values; normalize before comparing.
			if fs := s.GetField(n); fs != nil && fs.Encoder.Name() == "bcd" && len(v)%2 != 0 {
				v = "0" + v
			}
			if got != v {
				t.Errorf("field %d: got %q, want %q", n, got, v)
			}
		}
	})
}

// FuzzBitmapBitmap verifies ParseBitmap roundtrips through Bytes.
func FuzzBitmapRoundtrip(f *testing.F) {
	f.Add(uint64(0x6020000000000000), uint64(0))

	f.Fuzz(func(t *testing.T, primary, secondary uint64) {
		bm := &Bitmap{Primary: primary, Secondary: secondary}
		bytes := bm.Bytes()
		if len(bytes) != 8 && len(bytes) != 16 {
			t.Errorf("Bytes returned %d bytes", len(bytes))
		}

		parsed, consumed, err := ParseBitmap(bytes)
		if err != nil {
			t.Fatalf("ParseBitmap: %v", err)
		}
		if consumed != len(bytes) {
			t.Errorf("consumed %d != %d", consumed, len(bytes))
		}
		// Primary should roundtrip exactly.
		if parsed.Primary != primary {
			// If secondary was zero, primary is unchanged; if non-zero,
			// bit 1 might have been set by Bytes.
			if parsed.Primary&^uint64(1<<63) != primary&^uint64(1<<63) {
				t.Errorf("primary: got %016x, want %016x", parsed.Primary, primary)
			}
		}
	})
}

func padNumeric(s string, length int) string {
	if length == 0 {
		return s
	}
	for len(s) < length {
		s = "0" + s
	}
	if len(s) > length {
		s = s[len(s)-length:]
	}
	return s
}

func padAlpha(s string, length int) string {
	if length == 0 {
		return s
	}
	for len(s) < length {
		s = s + " "
	}
	if len(s) > length {
		s = s[:length]
	}
	return s
}
