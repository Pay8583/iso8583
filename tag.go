package iso8583

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Pay8583/iso8583/encoding"
	"github.com/Pay8583/iso8583/spec"
)

// ParsedTag holds the result of parsing one struct field's iso8583 tag.
type ParsedTag struct {
	FieldNumber   int    // ISO field number (2–128), 0 if MTI
	IsMTI         bool   // true if tag is "mti"
	LengthType    string // "fixed", "llvar", "lllvar", "llllvar", "" = inherit
	ValueName     string // "ascii", "bcd", "rbcd", "hex", "text", "raw", "binary", "" = inherit
	ValidatorName string // "n", "an", "ans", "b", "z", "ns", "xn", "" = inherit
	FixedLen      int    // for fixed=N tag
	MaxLen        int    // for len=N tag
	Pad           *byte  // for pad=X tag; nil = use default
	Secure        *bool  // for secure tag; nil = inherit
	Optional      bool   // for optional tag
	Name          string // field name override
	RBCDWidth     int    // width for RBCD value (0 = use FixedLen)
	BinWidth      int    // byte width for binary value (0 = use FixedLen)
}

// ParseTag parses an iso8583 struct tag value.
//
// Format: "<field_id>[,<option>...]"
//
//	field_id: "mti" for the MTI, or an integer 2–128 for a field number.
//
// Options (order-independent):
//
//	fixed=N    — fixed length of N characters
//	llvar      — 1-byte BCD length prefix
//	lllvar     — 2-byte BCD length prefix
//	llllvar    — 3-byte BCD length prefix
//	bcd        — BCD value encoding
//	rbcd       — right-aligned BCD value encoding
//	ascii      — ASCII value encoding
//	ebcdic     — EBCDIC value encoding
//	binary     — binary (big-endian int) value encoding
//	hex        — hex value encoding
//	text       — text value encoding (space-trim on decode)
//	raw        — raw bytes value encoding
//	n          — N validator (numeric [0-9])
//	an         — AN validator (alpha-numeric)
//	ans        — ANS validator (printable ASCII)
//	b          — B validator (accept anything)
//	z          — Z validator (track data)
//	ns         — NS validator (numeric-special)
//	xn         — XN validator (signed numeric)
//	pad=X      — pad byte for fixed-length (pad=0x00, pad='0', pad=' ')
//	secure     — mark field as security-sensitive
//	optional   — field may be absent (zero value → omit from bitmap)
//	len=N      — max content length in characters
//	name="..." — human-readable field name
//
// Backward-compatible synonyms: numeric → n, alpha → an.
func ParseTag(tag string) (*ParsedTag, error) {
	parts := strings.Split(tag, ",")
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty tag")
	}

	first := strings.TrimSpace(parts[0])
	pt := &ParsedTag{}

	if first == "mti" {
		pt.IsMTI = true
	} else {
		n, err := strconv.Atoi(first)
		if err != nil {
			return nil, fmt.Errorf("invalid field number %q: %w", first, err)
		}
		if n < 1 || n > 128 {
			return nil, fmt.Errorf("field number %d out of range [1,128]", n)
		}
		pt.FieldNumber = n
	}

	for _, p := range parts[1:] {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if err := pt.parseOption(p); err != nil {
			return nil, err
		}
	}
	return pt, nil
}

func (pt *ParsedTag) parseOption(p string) error {
	switch {
	// ── Length types ──────────────────────────────────────────────
	case p == "llvar":
		pt.LengthType = "llvar"
	case p == "lllvar":
		pt.LengthType = "lllvar"
	case p == "llllvar":
		pt.LengthType = "llllvar"
	case strings.HasPrefix(p, "fixed="):
		pt.LengthType = "fixed"
		n, err := strconv.Atoi(p[6:])
		if err != nil {
			return fmt.Errorf("invalid fixed value: %q", p)
		}
		pt.FixedLen = n

	// ── Value encodings ───────────────────────────────────────────
	case p == "bcd":
		pt.ValueName = "bcd"
	case p == "rbcd":
		pt.ValueName = "rbcd"
	case p == "ascii":
		pt.ValueName = "ascii"
	case p == "ebcdic":
		pt.ValueName = "ebcdic"
	case p == "binary":
		// Check context: "binary" can mean value encoding or validator.
		// If we already set a validator from a previous option, "binary" means value.
		// Otherwise, in the old grammar "binary" was a ContentType (validator-like).
		// New grammar: "binary" is value encoding; "b" is validator.
		pt.ValueName = "binary"
	case p == "hex":
		pt.ValueName = "hex"
	case p == "text":
		pt.ValueName = "text"
	case p == "raw":
		pt.ValueName = "raw"

	// ── Validators (new short names) ──────────────────────────────
	case p == "n":
		pt.ValidatorName = "n"
	case p == "an":
		pt.ValidatorName = "an"
	case p == "ans":
		pt.ValidatorName = "ans"
	case p == "b":
		pt.ValidatorName = "b"
	case p == "z":
		pt.ValidatorName = "z"
	case p == "ns":
		pt.ValidatorName = "ns"
	case p == "xn":
		pt.ValidatorName = "xn"

	// ── Validators (old long names — backward compatibility) ─────
	case p == "numeric":
		pt.ValidatorName = "n"
	case p == "alpha":
		pt.ValidatorName = "an"

	// ── Flags ─────────────────────────────────────────────────────
	case p == "secure":
		t := true
		pt.Secure = &t
	case p == "optional":
		pt.Optional = true

	// ── Parameterised options ─────────────────────────────────────
	case strings.HasPrefix(p, "pad="):
		b, err := parsePadByte(p[4:])
		if err != nil {
			return fmt.Errorf("invalid pad value: %q: %w", p, err)
		}
		pt.Pad = &b
	case strings.HasPrefix(p, "len="):
		n, err := strconv.Atoi(p[4:])
		if err != nil {
			return fmt.Errorf("invalid len value: %q", p)
		}
		pt.MaxLen = n
	case strings.HasPrefix(p, "max="): // backward-compat: max=N → len=N
		n, err := strconv.Atoi(p[4:])
		if err != nil {
			return fmt.Errorf("invalid max value: %q", p)
		}
		pt.MaxLen = n
	case strings.HasPrefix(p, "name="):
		pt.Name = strings.Trim(p[5:], `"'`)

	default:
		return fmt.Errorf("unknown tag option: %q", p)
	}
	return nil
}

// parsePadByte parses a pad byte value. Accepts:
//   - 0x00 style hex bytes
//   - '0' style character literals
//   - ' ' for space
func parsePadByte(s string) (byte, error) {
	if len(s) == 0 {
		return 0, fmt.Errorf("empty pad value")
	}
	// Hex: 0x00
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		n, err := strconv.ParseUint(s[2:], 16, 8)
		if err != nil {
			return 0, err
		}
		return byte(n), nil
	}
	// Quoted character: '0', ' '
	if len(s) >= 3 && s[0] == '\'' && s[len(s)-1] == '\'' {
		inner := s[1 : len(s)-1]
		if len(inner) == 1 {
			return inner[0], nil
		}
		return 0, fmt.Errorf("invalid quoted pad: %q", s)
	}
	// Raw digit: 0
	if len(s) == 1 {
		return s[0], nil
	}
	return 0, fmt.Errorf("cannot parse pad: %q", s)
}

// ── Resolution ──────────────────────────────────────────────────────────────────

// ResolveField merges a ParsedTag with a Protocol field definition.
// Tag options override Protocol defaults. If protoField is nil, a Field is
// built entirely from tag options (the tag MUST specify length and value).
func (pt *ParsedTag) ResolveField(protoField *spec.Field) (*spec.Field, error) {
	if pt.IsMTI {
		return nil, nil // MTI is not a data field
	}

	var out spec.Field
	var defaults spec.Field
	if protoField != nil {
		defaults = *protoField
	}

	// ── Length ────────────────────────────────────────────────────
	l := defaults.Len
	if pt.LengthType != "" || pt.FixedLen != 0 || pt.MaxLen != 0 {
		l = pt.resolveLength(defaults.Len)
	}
	if l == nil {
		return nil, fmt.Errorf("field %d: no length specified (tag %q)", pt.FieldNumber, pt.fieldTag())
	}
	out.Len = l

	// ── Value ─────────────────────────────────────────────────────
	v := defaults.Value
	if pt.ValueName != "" {
		var err error
		v, err = pt.resolveValue()
		if err != nil {
			return nil, fmt.Errorf("field %d: %w", pt.FieldNumber, err)
		}
	}
	if v == nil {
		return nil, fmt.Errorf("field %d: no value encoding specified (tag %q)", pt.FieldNumber, pt.fieldTag())
	}
	out.Value = v

	// ── Validator ─────────────────────────────────────────────────
	out.Valid = defaults.Valid
	if pt.ValidatorName != "" {
		out.Valid = pt.resolveValidator()
		if out.Valid == nil {
			return nil, fmt.Errorf("field %d: unknown validator %q", pt.FieldNumber, pt.ValidatorName)
		}
	}
	if out.Valid == nil {
		out.Valid = spec.B // default: accept anything
	}

	// ── Secure ────────────────────────────────────────────────────
	out.Secure = defaults.Secure
	if pt.Secure != nil {
		out.Secure = *pt.Secure
	}

	// ── Name ──────────────────────────────────────────────────────
	out.Name = defaults.Name
	if pt.Name != "" {
		out.Name = pt.Name
	}

	return &out, nil
}

func (pt *ParsedTag) resolveLength(protoLen spec.Length) spec.Length {
	max := pt.MaxLen
	if max == 0 && protoLen != nil {
		max = protoLen.MaxLen()
	}

	var enc encoding.Encoder = encoding.MustGet("bcd")
	if protoLen != nil {
		// Inherit encoder from proto if it's a varLen.
		if vl, ok := protoLen.(*spec.VarLen); ok {
			enc = vl.Enc()
		}
	}

	switch pt.LengthType {
	case "fixed":
		n := pt.FixedLen
		if n == 0 {
			n = max
		}
		pad := byte(' ')
		if pt.Pad != nil {
			pad = *pt.Pad
		} else if pt.ValueName != "" {
			pad = inferPad(pt.ValueName)
		} else if protoLen != nil {
			pad = protoLen.Pad()
		}
		return spec.Fixed(n, pad)

	case "llvar":
		if max == 0 {
			max = 99
		}
		return spec.LVAR(max, enc)

	case "lllvar":
		if max == 0 {
			max = 999
		}
		return spec.LLVAR(max, enc)

	case "llllvar":
		if max == 0 {
			max = 9999
		}
		return spec.LLLVAR(max, enc)

	default:
		// Inherit but override max if specified.
		if pt.MaxLen > 0 && protoLen != nil {
			// We can't easily change MaxLen on an existing Length, so create a new one.
			if protoLen.IsFixed() {
				pad := protoLen.Pad()
				return spec.Fixed(pt.MaxLen, pad)
			}
			// For variable-length, recreate with same wire size + new max.
			wl := protoLen.WireLen()
			switch wl {
			case 1:
				return spec.LVAR(pt.MaxLen, enc)
			case 2:
				return spec.LLVAR(pt.MaxLen, enc)
			case 3:
				return spec.LLLVAR(pt.MaxLen, enc)
			}
		}
		return protoLen
	}
}

func (pt *ParsedTag) resolveValue() (spec.Value, error) {
	switch pt.ValueName {
	case "ascii":
		return spec.ASCII, nil
	case "bcd":
		return spec.BCD, nil
	case "rbcd":
		w := pt.RBCDWidth
		if w == 0 {
			w = pt.FixedLen
		}
		return spec.RBCD(w), nil
	case "ebcdic":
		return nil, fmt.Errorf("EBCDIC value encoding not yet implemented")
	case "hex":
		return spec.Hex, nil
	case "text":
		return spec.Text, nil
	case "raw":
		return spec.Raw, nil
	case "binary":
		w := pt.BinWidth
		if w == 0 && pt.FixedLen > 0 {
			w = pt.FixedLen
		}
		if w == 0 {
			w = 8 // default: int64
		}
		return spec.Bin(w), nil
	default:
		return nil, fmt.Errorf("unknown value encoding %q", pt.ValueName)
	}
}

func (pt *ParsedTag) resolveValidator() spec.Validator {
	return spec.ValidatorByName(pt.ValidatorName)
}

// inferPad returns the default pad byte for a value encoding name.
func inferPad(valueName string) byte {
	switch valueName {
	case "bcd", "rbcd", "binary", "hex", "raw":
		return 0x00
	default:
		return ' '
	}
}

func (pt *ParsedTag) fieldTag() string {
	if pt.IsMTI {
		return "mti"
	}
	return fmt.Sprintf("%d", pt.FieldNumber)
}
