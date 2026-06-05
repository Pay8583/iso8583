package spec

import (
	"fmt"
	"io"
	"strconv"

	"github.com/Pay8583/iso8583/encoding"
)

// Length describes how a field's content length is indicated on the wire.
// For fixed-length fields there is no prefix (WireLen returns 0);
// for variable-length fields a 1–3 byte BCD or binary prefix is used.
type Length interface {
	// ReadLen reads the length prefix from r and returns the content byte count.
	// For fixed-length fields this is not used — callers should check IsFixed first.
	ReadLen(r io.Reader) (int, error)

	// WriteLen writes the length prefix for contentLen content bytes to w.
	// For fixed-length fields this is a no-op.
	WriteLen(w io.Writer, contentLen int) error

	// WireLen returns the number of bytes the length prefix occupies on the wire.
	// Returns 0 for fixed-length fields.
	WireLen() int

	// MaxLen returns the maximum content length in characters (for validation).
	MaxLen() int

	// IsFixed reports whether this is a fixed-length field (no length prefix).
	IsFixed() bool

	// FixedLen returns the fixed character length, or 0 if variable-length.
	FixedLen() int

	// Pad returns the padding byte for fixed-length fields, or 0 for variable.
	Pad() byte
}

// ── Constructors ───────────────────────────────────────────────────────────────

// Fixed returns a fixed-length field with no length prefix.
// n is the character count; pad is the byte used for padding short values.
// The caller (Writer/Reader) converts n to a wire-byte count based on the
// field's Value encoding.
func Fixed(n int, pad byte) Length {
	return fixedLen{n: n, pad: pad}
}

// LVAR returns a variable-length field with a 1-byte prefix.
// The prefix is encoded/decoded with e (typically encoding.BCD for standard
// ISO 8583, or Binary for custom protocols like PEP ISO). max is the
// maximum content length in characters; values 1–99 are valid for BCD.
func LVAR(max int, e encoding.Encoder) Length {
	if e == nil {
		e = encoding.MustGet("bcd")
	}
	return &VarLen{wire: 1, max: max, enc: e}
}

// LLVAR returns a variable-length field with a 2-byte prefix.
// max values 1–999 are valid for BCD encoding.
func LLVAR(max int, e encoding.Encoder) Length {
	if e == nil {
		e = encoding.MustGet("bcd")
	}
	return &VarLen{wire: 2, max: max, enc: e}
}

// LLLVAR returns a variable-length field with a 3-byte prefix.
// max values 1–9999 are valid for BCD encoding.
func LLLVAR(max int, e encoding.Encoder) Length {
	if e == nil {
		e = encoding.MustGet("bcd")
	}
	return &VarLen{wire: 3, max: max, enc: e}
}

// LVARbin returns a variable-length field with a 1-byte binary prefix
// (max 255). Used by custom protocols like PEP ISO.
func LVARbin(max int) Length {
	return &VarLen{wire: 1, max: max, enc: encoding.MustGet("binary")}
}

// LLVARbin returns a variable-length field with a 2-byte big-endian
// binary prefix (max 65535).
func LLVARbin(max int) Length {
	return &VarLen{wire: 2, max: max, enc: encoding.MustGet("binary")}
}

// LLLVARbin returns a variable-length field with a 3-byte big-endian
// binary prefix (max 16777215).
func LLLVARbin(max int) Length {
	return &VarLen{wire: 3, max: max, enc: encoding.MustGet("binary")}
}

// ── Implementations ────────────────────────────────────────────────────────────

// fixedLen is a fixed-length field: no prefix on the wire.
type fixedLen struct {
	n   int  // character count
	pad byte // padding byte
}

func (f fixedLen) ReadLen(_ io.Reader) (int, error) {
	// Not used — callers should check IsFixed() first and compute the wire-byte
	// count from FixedLen() and the field's Value encoding.
	return 0, nil
}

func (f fixedLen) WriteLen(_ io.Writer, _ int) error {
	return nil // no length prefix for fixed-length fields
}

func (f fixedLen) WireLen() int  { return 0 }
func (f fixedLen) MaxLen() int   { return f.n }
func (f fixedLen) IsFixed() bool { return true }
func (f fixedLen) FixedLen() int { return f.n }

// Pad returns the padding byte.
func (f fixedLen) Pad() byte { return f.pad }

// ────────────────────────────────────────────────────────────────────────────────

// VarLen is a variable-length field with a BCD or binary length prefix.
type VarLen struct {
	wire int              // prefix byte count: 1, 2, or 3
	max  int              // maximum content length in characters
	enc  encoding.Encoder // encodes/decodes the length prefix
}

func (v *VarLen) ReadLen(r io.Reader) (int, error) {
	prefix := make([]byte, v.wire)
	if _, err := io.ReadFull(r, prefix); err != nil {
		return 0, fmt.Errorf("length prefix: %w", err)
	}
	// Binary encoding: read as raw big-endian integer.
	if v.enc.Name() == "binary" {
		var n int
		for _, b := range prefix {
			n = n<<8 | int(b)
		}
		if n == 0 {
			return 0, fmt.Errorf("length prefix: zero-length field not allowed")
		}
		if n > v.max {
			return 0, fmt.Errorf("length prefix: value %d exceeds maximum %d", n, v.max)
		}
		return n, nil
	}
	// BCD encoding: decode as string, parse as decimal.
	s, err := v.enc.Decode(prefix)
	if err != nil {
		return 0, fmt.Errorf("length prefix decode: %w", err)
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("length prefix parse %q: %w", s, err)
	}
	if n == 0 {
		return 0, fmt.Errorf("length prefix: zero-length field not allowed")
	}
	if n > v.max {
		return 0, fmt.Errorf("length prefix: value %d exceeds maximum %d", n, v.max)
	}
	return n, nil
}

func (v *VarLen) WriteLen(w io.Writer, contentLen int) error {
	if contentLen <= 0 {
		return fmt.Errorf("cannot write length prefix for %d content bytes", contentLen)
	}
	// Binary encoding: write as raw big-endian bytes.
	if v.enc.Name() == "binary" {
		buf := make([]byte, v.wire)
		for i := range v.wire {
			buf[v.wire-1-i] = byte(contentLen >> (i * 8))
		}
		_, err := w.Write(buf)
		return err
	}
	// BCD encoding: format as decimal string, then BCD-encode.
	width := v.wire * 2 // BCD: 1 byte = 2 digits, 2 bytes = 4 digits, 3 bytes = 6 digits
	s := fmt.Sprintf("%0*d", width, contentLen)
	encoded, err := v.enc.Encode(s)
	if err != nil {
		return fmt.Errorf("length prefix encode: %w", err)
	}
	if len(encoded) != v.wire {
		return fmt.Errorf("length prefix: expected %d bytes, got %d", v.wire, len(encoded))
	}
	_, err = w.Write(encoded)
	return err
}

func (v *VarLen) WireLen() int  { return v.wire }
func (v *VarLen) MaxLen() int   { return v.max }
func (v *VarLen) IsFixed() bool { return false }
func (v *VarLen) FixedLen() int { return 0 }
func (v *VarLen) Pad() byte         { return 0 } // variable-length has no pad

// Enc returns the encoder used for the length prefix.
func (v *VarLen) Enc() encoding.Encoder { return v.enc }
