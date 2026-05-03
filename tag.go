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
	FieldNumber int
	IsMTI       bool
	LengthType  spec.LengthType // zero value means use spec default
	ContentType spec.ContentType
	EncoderName string          // empty means use spec default
	FixedLen    int             // 0 means use spec default
	MinLen      int
	MaxLen      int             // 0 means use spec default
	Optional    bool
	Name        string
}

// ParseTag parses an iso8583 struct tag value.
//
// Format: "<field_id>[,<hint>...]"
//   field_id: "mti" for the MTI, or an integer 1–128 for a field number.
//   hints (all optional): llvar, lllvar, llllvar, bcd, ascii, ebcdic, binary,
//     numeric, alpha, len=N, min=N, max=N, optional, name="..."
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
		switch {
		case p == "llvar":
			pt.LengthType = spec.LLVAR
		case p == "lllvar":
			pt.LengthType = spec.LLLVAR
		case p == "llllvar":
			pt.LengthType = spec.LLLLVAR
		case p == "bcd", p == "ascii", p == "ebcdic", p == "binary":
			pt.EncoderName = p
		case p == "numeric":
			pt.ContentType = spec.Numeric
		case p == "alpha":
			pt.ContentType = spec.Alpha
		case p == "binary":
			pt.ContentType = spec.Binary
		case p == "optional":
			pt.Optional = true
		case strings.HasPrefix(p, "len="):
			n, err := strconv.Atoi(p[4:])
			if err != nil {
				return nil, fmt.Errorf("invalid len value: %q", p)
			}
			pt.FixedLen = n
		case strings.HasPrefix(p, "min="):
			n, err := strconv.Atoi(p[4:])
			if err != nil {
				return nil, fmt.Errorf("invalid min value: %q", p)
			}
			pt.MinLen = n
		case strings.HasPrefix(p, "max="):
			n, err := strconv.Atoi(p[4:])
			if err != nil {
				return nil, fmt.Errorf("invalid max value: %q", p)
			}
			pt.MaxLen = n
		case strings.HasPrefix(p, "name="):
			pt.Name = strings.Trim(p[5:], `"`)
		default:
			return nil, fmt.Errorf("unknown tag hint: %q", p)
		}
	}
	return pt, nil
}

// ResolveFieldSpec merges a ParsedTag with a FieldSpec from the spec registry.
// Tag overrides take precedence over spec defaults.
func (pt *ParsedTag) ResolveFieldSpec(sf *spec.FieldSpec) *spec.FieldSpec {
	if sf == nil {
		return nil
	}
	out := *sf // copy
	if pt.LengthType != 0 {
		out.LengthType = pt.LengthType
	}
	if pt.ContentType != 0 {
		out.ContentType = pt.ContentType
	}
	if pt.EncoderName != "" {
		if e := encoding.Get(pt.EncoderName); e != nil {
			out.Encoder = e
		}
	}
	if pt.FixedLen != 0 {
		out.FixedLen = pt.FixedLen
	}
	if pt.MinLen != 0 {
		out.MinLen = pt.MinLen
	}
	if pt.MaxLen != 0 {
		out.MaxLen = pt.MaxLen
	}
	if pt.Optional {
		out.Optional = true
	}
	if pt.Name != "" {
		out.Name = pt.Name
	}
	return &out
}
