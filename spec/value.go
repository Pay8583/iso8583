package spec

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/Pay8583/iso8583/encoding"
)

// Value converts between a Go value and the wire-format bytes for one field.
// It sits above encoding.Encoder — Value handles Go-type conversions (string,
// int64, []byte, time.Time) while Encoder handles the byte-level transformation
// (BCD nibble packing, EBCDIC code page, etc.).
type Value interface {
	// Encode converts a Go value to wire bytes. fieldLen is the field's fixed
	// length in characters (0 for variable-length fields). Not all Value
	// implementations use it.
	Encode(v any, fieldLen int) ([]byte, error)
	// Decode converts wire bytes to a Go value.
	Decode(b []byte) (any, error)
	// String returns a human-readable name for this value encoding.
	String() string
}

// ── Built-in values ────────────────────────────────────────────────────────────

// ASCII is a passthrough string value using ASCII encoding.
var ASCII Value = asciiValue{}

// BCD is a left-aligned packed BCD value. For odd-length input strings the
// encoding.BCD Encoder left-pads with a zero nibble.
var BCD Value = bcdValue{}

// Hex encodes binary data as a hex string. Encode expects a string of hex
// characters; Decode returns a hex string.
var Hex Value = hexValue{}

// Text is a string value that trims trailing spaces on decode. It uses ASCII
// encoding on the wire.
var Text Value = textValue{}

// Raw passes through []byte without any encoding.
var Raw Value = rawValue{}

// RBCD returns a right-aligned packed BCD value. fieldWidth specifies the
// character width for right-justification with leading zeros before BCD
// encoding. Use 0 for variable-length fields (no padding).
func RBCD(fieldWidth int) Value { return rbcdValue{width: fieldWidth} }

// Bin returns a big-endian binary integer value. byteWidth specifies the
// number of bytes for the integer on the wire (1, 2, 4, or 8).
func Bin(byteWidth int) Value { return binValue{width: byteWidth} }

// ── Implementations ────────────────────────────────────────────────────────────

type asciiValue struct{}

func (asciiValue) Encode(v any, _ int) ([]byte, error) {
	s, err := toString(v)
	if err != nil {
		return nil, err
	}
	return encoding.MustGet("ascii").Encode(s)
}

func (asciiValue) Decode(b []byte) (any, error) {
	s, err := encoding.MustGet("ascii").Decode(b)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (asciiValue) String() string { return "ASCII" }

// ────────────────────────────────────────────────────────────────────────────────

type bcdValue struct{}

func (bcdValue) Encode(v any, _ int) ([]byte, error) {
	s, err := toString(v)
	if err != nil {
		return nil, err
	}
	return encoding.MustGet("bcd").Encode(s)
}

func (bcdValue) Decode(b []byte) (any, error) {
	s, err := encoding.MustGet("bcd").Decode(b)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (bcdValue) String() string { return "BCD" }

// ────────────────────────────────────────────────────────────────────────────────

type rbcdValue struct {
	width int // character width for right-justification; 0 = no padding
}

func (v rbcdValue) Encode(val any, fieldLen int) ([]byte, error) {
	s, err := toString(val)
	if err != nil {
		return nil, err
	}
	w := v.width
	if w == 0 {
		w = fieldLen
	}
	if w > 0 && len(s) < w {
		// Right-justify: left-pad with zeros.
		s = strings.Repeat("0", w-len(s)) + s
	}
	return encoding.MustGet("bcd").Encode(s)
}

func (rbcdValue) Decode(b []byte) (any, error) {
	s, err := encoding.MustGet("bcd").Decode(b)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (v rbcdValue) String() string {
	if v.width > 0 {
		return fmt.Sprintf("RBCD(%d)", v.width)
	}
	return "RBCD"
}

// ────────────────────────────────────────────────────────────────────────────────

type hexValue struct{}

func (hexValue) Encode(v any, _ int) ([]byte, error) {
	s, err := toString(v)
	if err != nil {
		return nil, err
	}
	return hex.DecodeString(s)
}

func (hexValue) Decode(b []byte) (any, error) {
	return hex.EncodeToString(b), nil
}

func (hexValue) String() string { return "Hex" }

// ────────────────────────────────────────────────────────────────────────────────

type textValue struct{}

func (textValue) Encode(v any, _ int) ([]byte, error) {
	s, err := toString(v)
	if err != nil {
		return nil, err
	}
	return encoding.MustGet("ascii").Encode(s)
}

func (textValue) Decode(b []byte) (any, error) {
	s, err := encoding.MustGet("ascii").Decode(b)
	if err != nil {
		return nil, err
	}
	return strings.TrimRight(s, " "), nil
}

func (textValue) String() string { return "Text" }

// ────────────────────────────────────────────────────────────────────────────────

type rawValue struct{}

func (rawValue) Encode(v any, _ int) ([]byte, error) {
	switch val := v.(type) {
	case []byte:
		out := make([]byte, len(val))
		copy(out, val)
		return out, nil
	case string:
		return []byte(val), nil
	default:
		return nil, fmt.Errorf("raw: expected []byte or string, got %T", v)
	}
}

func (rawValue) Decode(b []byte) (any, error) {
	out := make([]byte, len(b))
	copy(out, b)
	return out, nil
}

func (rawValue) String() string { return "Raw" }

// ────────────────────────────────────────────────────────────────────────────────

type binValue struct {
	width int // byte width: 1, 2, 4, or 8
}

func (v binValue) Encode(val any, fieldLen int) ([]byte, error) {
	n, err := toInt64(val)
	if err != nil {
		return nil, err
	}
	w := v.width
	if w == 0 {
		w = fieldLen
	}
	switch w {
	case 1:
		if n < math.MinInt8 || n > math.MaxUint8 {
			return nil, fmt.Errorf("binary: value %d overflows 1 byte", n)
		}
		return []byte{byte(n)}, nil
	case 2:
		if n < math.MinInt16 || n > math.MaxUint16 {
			return nil, fmt.Errorf("binary: value %d overflows 2 bytes", n)
		}
		buf := make([]byte, 2)
		binary.BigEndian.PutUint16(buf, uint16(n))
		return buf, nil
	case 4:
		if n < math.MinInt32 || n > math.MaxUint32 {
			return nil, fmt.Errorf("binary: value %d overflows 4 bytes", n)
		}
		buf := make([]byte, 4)
		binary.BigEndian.PutUint32(buf, uint32(n))
		return buf, nil
	case 8:
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, uint64(n))
		return buf, nil
	default:
		return nil, fmt.Errorf("binary: unsupported byte width %d (want 1, 2, 4, or 8)", w)
	}
}

func (binValue) Decode(b []byte) (any, error) {
	switch len(b) {
	case 1:
		return int64(b[0]), nil
	case 2:
		return int64(binary.BigEndian.Uint16(b)), nil
	case 4:
		return int64(binary.BigEndian.Uint32(b)), nil
	case 8:
		return int64(binary.BigEndian.Uint64(b)), nil
	default:
		// For unusual sizes, left-pad or truncate.
		buf := make([]byte, 8)
		copy(buf[8-len(b):], b)
		return int64(binary.BigEndian.Uint64(buf)), nil
	}
}

func (v binValue) String() string {
	if v.width > 0 {
		return fmt.Sprintf("Binary(%d)", v.width)
	}
	return "Binary"
}

// ── Helpers ─────────────────────────────────────────────────────────────────────

// toString converts common Go types to their string representation.
// Supported: string, []byte, int64, uint64, int, uint.
func toString(v any) (string, error) {
	switch val := v.(type) {
	case string:
		return val, nil
	case []byte:
		return string(val), nil
	case int64:
		return fmt.Sprintf("%d", val), nil
	case int:
		return fmt.Sprintf("%d", val), nil
	case uint64:
		return fmt.Sprintf("%d", val), nil
	case uint:
		return fmt.Sprintf("%d", val), nil
	case int32:
		return fmt.Sprintf("%d", val), nil
	case uint32:
		return fmt.Sprintf("%d", val), nil
	default:
		return "", fmt.Errorf("value: cannot convert %T to string", v)
	}
}

// toInt64 converts common Go types to int64.
func toInt64(v any) (int64, error) {
	switch val := v.(type) {
	case int64:
		return val, nil
	case int:
		return int64(val), nil
	case uint64:
		if val > math.MaxInt64 {
			return 0, fmt.Errorf("binary: uint64 value %d overflows int64", val)
		}
		return int64(val), nil
	case uint:
		return int64(val), nil
	case int32:
		return int64(val), nil
	case uint32:
		return int64(val), nil
	case string:
		n, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("binary: cannot parse %q as integer", val)
		}
		return n, nil
	default:
		return 0, fmt.Errorf("binary: cannot convert %T to int64", v)
	}
}

// ValueByName returns a simple (non-parameterized) Value by case-insensitive
// name. RBCD and Binary are parameterized and not included — use the
// constructors directly.
func ValueByName(name string) Value {
	switch strings.ToUpper(name) {
	case "ASCII":
		return ASCII
	case "BCD":
		return BCD
	case "HEX":
		return Hex
	case "TEXT":
		return Text
	case "RAW":
		return Raw
	default:
		return nil
	}
}
