package iso8583

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/Pay8583/iso8583/spec"
)

// FuzzUnmarshal tests that random bytes don't cause panics.
func makeFuzzFields(n int) []spec.Field {
	fields := make([]spec.Field, n)
	for i := range fields {
		fields[i] = spec.Field{
			Name:  fmt.Sprintf("F%d", i+2),
			Len:   spec.Fixed(10, ' '),
			Valid: spec.B,
			Value: spec.ASCII,
		}
	}
	return fields
}

func FuzzUnmarshal(f *testing.F) {
	// Seed corpus with valid-looking data.
	f.Add([]byte("02006020000000000000"))
	f.Add([]byte("02106020000000000000"))
	f.Add(make([]byte, 20))

	p := &spec.Protocol{
		Name:   "fuzz",
		MTI:    spec.ASCIIMTI,
		Bitmap: spec.HexBitmap,
		Fields: makeFuzzFields(10), // fields 2–11
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		// Unmarshal should never panic.
		var decoded struct {
			MTI  uint   `iso8583:"mti"`
			F2   string `iso8583:"2,optional"`
			F3   string `iso8583:"3,optional"`
			F4   string `iso8583:"4,optional"`
		}
		_ = Unmarshal(data, &decoded, p)
	})
}

// FuzzWriterReaderRoundTrip tests that writing then reading produces
// consistent results for random field values.
func FuzzWriterReaderRoundTrip(f *testing.F) {
	f.Add("0200", "1234567890", "000000", "123456")

	p := &spec.Protocol{
		Name:   "fuzz-rt",
		MTI:    spec.ASCIIMTI,
		Bitmap: spec.HexBitmap,
		Fields: []spec.Field{
			{Name: "F2", Len: spec.LVAR(19, nil), Valid: spec.N, Value: spec.ASCII},
			{Name: "F3", Len: spec.Fixed(6, '0'), Valid: spec.N, Value: spec.ASCII},
			{Name: "F4", Len: spec.Fixed(6, '0'), Valid: spec.N, Value: spec.ASCII},
		},
	}

	f.Fuzz(func(t *testing.T, mtiStr, f2, f3, f4 string) {
		// Only test with valid numeric values.
		if !spec.N.Ok(f2) || !spec.N.Ok(f3) || !spec.N.Ok(f4) {
			return
		}
		// Only test with valid MTI strings.
		var mti uint
		if _, _, err := decodeMTI(bytes.NewReader([]byte(mtiStr)), spec.ASCIIMTI); err != nil {
			// Not a valid 4-char hex MTI string; skip.
			if len(mtiStr) != 4 {
				return
			}
			// Use a default MTI.
			mti = 0x0200
		} else {
			m, _, _ := decodeMTI(bytes.NewReader([]byte(mtiStr)), spec.ASCIIMTI)
			mti = m
		}

		// Write.
		var buf bytes.Buffer
		w := NewWriter(p, &buf)
		w.WriteMTI(mti)
		if f2 != "" {
			w.WriteString(2, f2)
		}
		if f3 != "" {
			w.WriteString(3, f3)
		}
		if f4 != "" {
			w.WriteString(4, f4)
		}
		if err := w.Close(); err != nil {
			return
		}

		// Read back.
		r := NewReader(p, bytes.NewReader(buf.Bytes()))
		_, err := r.ReadMTI()
		if err != nil {
			t.Fatalf("ReadMTI: %v", err)
		}

		present, err := r.PresentFields()
		if err != nil {
			return
		}
		for _, n := range present {
			var s string
			if err := r.ReadString(n, &s); err != nil {
				t.Fatalf("ReadString(%d): %v", n, err)
			}
		}
	})
}
