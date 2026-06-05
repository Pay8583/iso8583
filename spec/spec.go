// Package spec defines the field-layout specification types for ISO 8583 messages.
// It provides Protocol and Field types, a global registry of named protocols,
// and built-in definitions for ISO 8583:1987, 1993, and 2003.
package spec

import "fmt"

// MTIEncoder encodes/decodes the Message Type Indicator to/from wire bytes.
type MTIEncoder interface {
	// Encode converts an MTI value to wire bytes.
	Encode(mti uint) ([]byte, error)
	// Decode converts wire bytes to an MTI value.
	Decode(b []byte) (uint, error)
	// WireLen returns the number of bytes the MTI occupies on the wire.
	WireLen() int
	// Name returns a human-readable name.
	Name() string
}

// ── Built-in MTI encoders ──────────────────────────────────────────────────────

type asciiMTI struct{}

func (asciiMTI) Encode(mti uint) ([]byte, error) { return []byte(fmt.Sprintf("%04X", mti)), nil }
func (asciiMTI) Decode(b []byte) (uint, error) {
	if len(b) != 4 {
		return 0, fmt.Errorf("ASCII MTI: expected 4 bytes, got %d", len(b))
	}
	var mti uint
	if _, err := fmt.Sscanf(string(b), "%04X", &mti); err != nil {
		return 0, fmt.Errorf("ASCII MTI: %w", err)
	}
	return mti, nil
}
func (asciiMTI) WireLen() int { return 4 }
func (asciiMTI) Name() string { return "ASCII" }

type bcdMTI struct{}

func (bcdMTI) Encode(mti uint) ([]byte, error) {
	return []byte{byte(mti >> 8), byte(mti)}, nil
}
func (bcdMTI) Decode(b []byte) (uint, error) {
	if len(b) != 2 {
		return 0, fmt.Errorf("BCD MTI: expected 2 bytes, got %d", len(b))
	}
	return uint(b[0])<<8 | uint(b[1]), nil
}
func (bcdMTI) WireLen() int { return 2 }
func (bcdMTI) Name() string { return "BCD" }

type binaryMTI struct{}

func (binaryMTI) Encode(mti uint) ([]byte, error) {
	if mti > 0xFF {
		return nil, fmt.Errorf("binary MTI: value %#x exceeds 1 byte", mti)
	}
	return []byte{byte(mti)}, nil
}
func (binaryMTI) Decode(b []byte) (uint, error) {
	if len(b) != 1 {
		return 0, fmt.Errorf("binary MTI: expected 1 byte, got %d", len(b))
	}
	return uint(b[0]), nil
}
func (binaryMTI) WireLen() int { return 1 }
func (binaryMTI) Name() string { return "Binary" }

// Built-in MTI encoders.
var (
	ASCIIMTI  MTIEncoder = asciiMTI{}  // 4 ASCII hex bytes, e.g. 0x0200 → "0200"
	BCDMTI    MTIEncoder = bcdMTI{}    // 2 packed BCD bytes, e.g. 0x0200 → 0x02 0x00
	BinaryMTI MTIEncoder = binaryMTI{} // 1 raw byte, e.g. 0x80 (used by PEP ISO)
)

// ── Bitmap ──────────────────────────────────────────────────────────────────────

// BitmapEncoding describes how the bitmap bytes are represented on the wire.
type BitmapEncoding uint8

const (
	HexBitmap    BitmapEncoding = iota // 2 hex chars per byte (ASCII)
	BinaryBitmap                       // raw bytes
)

// SecurityLevel controls whether sensitive fields are visible in exports.
type SecurityLevel uint8

const (
	SecurityLevelLow  SecurityLevel = iota // logs/exports — Secure fields masked
	SecurityLevelHigh                      // raw/internal — all fields visible
)

// SignConfig carries the signing configuration for a protocol.
// It is stored in Protocol so that MAC helpers can derive the correct
// payload extraction rules and placeholder positioning from the protocol
// definition itself.
type SignConfig struct {
	IncludeMTI    bool
	IncludeBitmap bool
	ExcludeFields []int // fields to omit (e.g., the MAC field)
	MACLength     int   // byte length of the MAC (0 = use signer output)
	MACField      int   // which field holds the MAC (64 or 128)
}

// Protocol defines a complete ISO 8583 message layout — an MTI encoding,
// a bitmap encoding, and an ordered slice of field definitions.
//
// Fields are indexed by position: Fields[0] corresponds to ISO field 2,
// Fields[1] to field 3, ..., Fields[126] to field 128.
type Protocol struct {
	Name        string          // e.g. "ISO8583:1987-ASCII"
	Version     string          // "1987", "1993", "2003", or custom
	MTI         MTIEncoder      // how the MTI is encoded on the wire
	Bitmap      BitmapEncoding  // how the bitmap is encoded on the wire
	BitmapWidth int             // bytes: 0=auto(8/16), 8=standard, 5=pepiso, etc.
	Fields      []Field         // index 0 → ISO field 2
	Sign        *SignConfig     // signing configuration (may be nil)
	Description string
}

// GetField returns the Field definition for ISO field index n (2–128),
// or nil if n is out of range or not defined.
func (p *Protocol) GetField(n int) *Field {
	if n < 2 {
		return nil
	}
	idx := n - 2
	if idx >= len(p.Fields) {
		return nil
	}
	return &p.Fields[idx]
}

// NumFields returns the number of field definitions in the protocol.
func (p *Protocol) NumFields() int { return len(p.Fields) }

// HasSecondaryBitmap reports whether the protocol uses fields 65–128
// and therefore requires a secondary bitmap.
func (p *Protocol) HasSecondaryBitmap() bool { return len(p.Fields) > 63 }

// EffectiveBitmapWidth returns the bitmap byte width for this protocol.
// If BitmapWidth is set (non-zero), it is used. Otherwise, 8 bytes for
// primary-only, 16 bytes for primary+secondary.
func (p *Protocol) EffectiveBitmapWidth() int {
	if p.BitmapWidth > 0 {
		return p.BitmapWidth
	}
	if p.HasSecondaryBitmap() {
		return 16
	}
	return 8
}

// MaxBitmapField returns the highest field number that can be represented
// in this protocol's bitmap. For an 8-byte bitmap it's 64; for 5-byte it's 40.
func (p *Protocol) MaxBitmapField() int {
	bits := p.EffectiveBitmapWidth() * 8
	if bits > 128 {
		bits = 128
	}
	return bits
}

// Field describes one ISO 8583 data element (fields 2–128).
// Field 1 is the bitmap itself and is not represented as a Field.
type Field struct {
	Name   string     // human-readable name, e.g. "PAN", "Amount"
	Len    Length     // wire-level length encoding
	Valid  Validator  // value validation rule
	Value  Value      // Go-value ↔ wire-bytes encoding
	Secure bool       // if true, masked when exported for logging
}

// IsAllowed reports whether the field's security level allows it to be
// visible at the given level. Secure fields are only visible at
// SecurityLevelHigh (raw/internal use).
func (f *Field) IsAllowed(level SecurityLevel) bool {
	if !f.Secure {
		return true
	}
	return level >= SecurityLevelHigh
}
