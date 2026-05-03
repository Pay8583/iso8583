package iso8583

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Pay8583/iso8583/encoding"
	"github.com/Pay8583/iso8583/internal/field"
	"github.com/Pay8583/iso8583/spec"
)

// wireLen returns the number of wire bytes a fixed-length field occupies.
func wireLen(fs *spec.FieldSpec) int {
	if fs.FixedLen == 0 {
		return 0
	}
	if fs.Encoder != nil && fs.Encoder.Name() == "bcd" {
		return (fs.FixedLen + 1) / 2
	}
	return fs.FixedLen
}

// padValue pads a string value to fs.FixedLen before encoding.
// Numeric fields are left-padded with '0', others are right-padded with space.
func padValue(v string, fs *spec.FieldSpec) string {
	if fs.FixedLen == 0 {
		return v
	}
	if len(v) >= fs.FixedLen {
		return v[:fs.FixedLen]
	}
	switch fs.ContentType {
	case spec.Numeric:
		return strings.Repeat("0", fs.FixedLen-len(v)) + v
	default:
		return v + strings.Repeat(" ", fs.FixedLen-len(v))
	}
}

// BuildBitmap creates a Bitmap from a set of present field numbers (2–128).
func BuildBitmap(numbers []int) *Bitmap {
	bm := &Bitmap{}
	for _, n := range numbers {
		bm.Set(n)
	}
	return bm
}

// WriteMTI appends the 4-byte MTI to buf.
func WriteMTI(buf []byte, mti string, enc encoding.Encoder) ([]byte, error) {
	if len(mti) != 4 {
		return nil, fmt.Errorf("MTI must be 4 bytes, got %d", len(mti))
	}
	b, err := enc.Encode(mti)
	if err != nil {
		return nil, newError("encode", 0, err)
	}
	return append(buf, b...), nil
}

// PackField encodes a field value and appends it to buf with the appropriate length prefix.
func PackField(buf []byte, fs *spec.FieldSpec, value string) ([]byte, error) {
	switch fs.LengthType {
	case spec.Fixed:
		value = padValue(value, fs)
		encoded, err := fs.Encoder.Encode(value)
		if err != nil {
			return nil, newError("encode", fs.Index, err)
		}
		buf = append(buf, encoded...)

	case spec.LLVAR:
		encoded, err := fs.Encoder.Encode(value)
		if err != nil {
			return nil, newError("encode", fs.Index, err)
		}
		if fs.MaxLen > 0 && len(encoded) > fs.MaxLen {
			return nil, newError("encode", fs.Index,
				fmt.Errorf("%w: %d > %d", ErrFieldTooLong, len(encoded), fs.MaxLen))
		}
		if len(encoded) > 99 {
			return nil, newError("encode", fs.Index,
				fmt.Errorf("%w: llvar max 99 bytes, got %d", ErrFieldTooLong, len(encoded)))
		}
		buf = field.WriteLLVAR(buf, encoded)

	case spec.LLLVAR:
		encoded, err := fs.Encoder.Encode(value)
		if err != nil {
			return nil, newError("encode", fs.Index, err)
		}
		if fs.MaxLen > 0 && len(encoded) > fs.MaxLen {
			return nil, newError("encode", fs.Index,
				fmt.Errorf("%w: %d > %d", ErrFieldTooLong, len(encoded), fs.MaxLen))
		}
		if len(encoded) > 999 {
			return nil, newError("encode", fs.Index,
				fmt.Errorf("%w: lllvar max 999 bytes, got %d", ErrFieldTooLong, len(encoded)))
		}
		buf = field.WriteLLLVAR(buf, encoded)

	case spec.LLLLVAR:
		encoded, err := fs.Encoder.Encode(value)
		if err != nil {
			return nil, newError("encode", fs.Index, err)
		}
		if fs.MaxLen > 0 && len(encoded) > fs.MaxLen {
			return nil, newError("encode", fs.Index,
				fmt.Errorf("%w: %d > %d", ErrFieldTooLong, len(encoded), fs.MaxLen))
		}
		if len(encoded) > 9999 {
			return nil, newError("encode", fs.Index,
				fmt.Errorf("%w: llllvar max 9999 bytes, got %d", ErrFieldTooLong, len(encoded)))
		}
		buf = field.WriteLLLLVAR(buf, encoded)
	}
	return buf, nil
}

// UnpackField reads one field from data, returning the decoded string value and bytes consumed.
func UnpackField(data []byte, fs *spec.FieldSpec) (value string, consumed int, err error) {
	var raw []byte

	switch fs.LengthType {
	case spec.Fixed:
		wl := wireLen(fs)
		raw, err = field.ReadFixed(data, wl)
		if err != nil {
			return "", 0, newError("decode", fs.Index, err)
		}
		consumed = wl

	case spec.LLVAR:
		raw, consumed, err = field.ReadLLVAR(data)
		if err != nil {
			return "", 0, newError("decode", fs.Index, err)
		}
		if fs.MaxLen > 0 && len(raw) > fs.MaxLen {
			return "", 0, newError("decode", fs.Index,
				fmt.Errorf("%w: %d > %d", ErrFieldTooLong, len(raw), fs.MaxLen))
		}

	case spec.LLLVAR:
		raw, consumed, err = field.ReadLLLVAR(data)
		if err != nil {
			return "", 0, newError("decode", fs.Index, err)
		}
		if fs.MaxLen > 0 && len(raw) > fs.MaxLen {
			return "", 0, newError("decode", fs.Index,
				fmt.Errorf("%w: %d > %d", ErrFieldTooLong, len(raw), fs.MaxLen))
		}

	case spec.LLLLVAR:
		raw, consumed, err = field.ReadLLLLVAR(data)
		if err != nil {
			return "", 0, newError("decode", fs.Index, err)
		}
		if fs.MaxLen > 0 && len(raw) > fs.MaxLen {
			return "", 0, newError("decode", fs.Index,
				fmt.Errorf("%w: %d > %d", ErrFieldTooLong, len(raw), fs.MaxLen))
		}
	}

	value, err = fs.Encoder.Decode(raw)
	if err != nil {
		return "", 0, newError("decode", fs.Index, err)
	}
	return value, consumed, nil
}

// PackMessage serializes a Message into ISO 8583 wire format bytes.
func PackMessage(msg *Message) ([]byte, error) {
	if msg.Spec == nil {
		return nil, fmt.Errorf("message has no spec")
	}
	s := msg.Spec

	// Collect present field numbers.
	numbers := make([]int, 0, len(msg.Fields))
	for n := range msg.Fields {
		if n < 2 || n > s.MaxField {
			continue
		}
		if fs := s.GetField(n); fs == nil {
			continue
		}
		numbers = append(numbers, n)
	}
	sort.Ints(numbers)
	bm := BuildBitmap(numbers)

	// Pre-calculate total size for buffer allocation.
	total := 4 + len(bm.Bytes()) // MTI + bitmap
	for _, n := range numbers {
		fs := s.GetField(n)
		if fs == nil {
			continue
		}
		switch fs.LengthType {
		case spec.Fixed:
			total += wireLen(fs)
		case spec.LLVAR:
			enc, err := fs.Encoder.Encode(msg.Fields[n])
			if err != nil {
				return nil, newError("encode", n, err)
			}
			total += 1 + len(enc)
		case spec.LLLVAR:
			enc, err := fs.Encoder.Encode(msg.Fields[n])
			if err != nil {
				return nil, newError("encode", n, err)
			}
			total += 2 + len(enc)
		case spec.LLLLVAR:
			enc, err := fs.Encoder.Encode(msg.Fields[n])
			if err != nil {
				return nil, newError("encode", n, err)
			}
			total += 3 + len(enc)
		}
	}

	buf := make([]byte, 0, total)

	// MTI.
	buf, err := WriteMTI(buf, msg.MTI, s.MtiEncoder)
	if err != nil {
		return nil, err
	}

	// Bitmap.
	buf = append(buf, bm.Bytes()...)

	// Fields.
	for _, n := range numbers {
		fs := s.GetField(n)
		buf, err = PackField(buf, fs, msg.Fields[n])
		if err != nil {
			return nil, err
		}
	}
	return buf, nil
}

// UnpackMessage deserializes ISO 8583 wire format bytes into a Message.
func UnpackMessage(data []byte, s *spec.Spec) (*Message, error) {
	if len(data) < 12 {
		return nil, newError("decode", 0, ErrTruncated)
	}

	msg := &Message{Spec: s, Fields: make(map[int]string)}

	// MTI (4 bytes).
	mti, err := s.MtiEncoder.Decode(data[:4])
	if err != nil {
		return nil, newError("decode", 0, err)
	}
	msg.MTI = mti

	// Bitmap.
	bm, bmLen, err := ParseBitmap(data[4:])
	if err != nil {
		return nil, newError("decode", 0, err)
	}

	cursor := 4 + bmLen
	fields := bm.PresentFields()

	for _, n := range fields {
		fs := s.GetField(n)
		if fs == nil {
			return nil, newError("decode", n, fmt.Errorf("%w: field %d", ErrUnknownField, n))
		}
		if cursor >= len(data) {
			return nil, newError("decode", n, ErrTruncated)
		}
		value, consumed, err := UnpackField(data[cursor:], fs)
		if err != nil {
			return nil, err
		}
		msg.Fields[n] = value
		cursor += consumed
	}

	return msg, nil
}
