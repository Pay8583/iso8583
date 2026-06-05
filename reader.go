package iso8583

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"strconv"
	"time"

	"github.com/Pay8583/iso8583/spec"
)

// A Reader unpacks ISO 8583 messages field by field. It reads an MTI and
// bitmap from the wire, then on each Read* call decodes, validates, and
// stores the field value in the destination pointer.
//
// Bytes() returns the raw bytes consumed so far for MAC verification.
// Export() returns a map suitable for JSON logging with security-sensitive
// fields masked.
type Reader struct {
	p       *spec.Protocol
	r       io.Reader
	buf     bytes.Buffer // raw wire bytes accumulator
	mti     uint
	mtiRead bool
	parsed  bool         // bitmap + all fields parsed
	raw     map[int][]byte // field index → raw content bytes (no length prefix)
	strs    map[int]string // field index → string value (after decode)
	err     error          // parse error, returned on next Read* call
}

// NewReader returns a Reader that unpacks messages according to p from r.
func NewReader(p *spec.Protocol, r io.Reader) *Reader {
	return &Reader{
		p:   p,
		r:   r,
		raw: make(map[int][]byte),
		strs: make(map[int]string),
	}
}

// ReadMTI reads and decodes the MTI from the wire.
func (r *Reader) ReadMTI() (uint, error) {
	if r.mtiRead {
		return 0, fmt.Errorf("reader: MTI already read")
	}
	r.mtiRead = true

	mti, mtiBytes, err := decodeMTI(r.r, r.p.MTI)
	if err != nil {
		return 0, err
	}
	r.mti = mti
	r.buf.Write(mtiBytes)
	return mti, nil
}

// ReadString reads field n, decodes it as a string, and stores it in dst.
func (r *Reader) ReadString(n int, dst *string) error {
	if err := r.ensureParsed(); err != nil {
		return err
	}
	return r.readField(n, dst, "string")
}

// ReadInt reads field n, decodes it as an int64, and stores it in dst.
func (r *Reader) ReadInt(n int, dst *int64) error {
	if err := r.ensureParsed(); err != nil {
		return err
	}
	return r.readField(n, dst, "int")
}

// ReadBytes reads field n, decodes it as []byte, and stores it in dst.
func (r *Reader) ReadBytes(n int, dst *[]byte) error {
	if err := r.ensureParsed(); err != nil {
		return err
	}
	return r.readField(n, dst, "bytes")
}

// ReadTime reads field n, decodes it as time.Time, and stores it in dst.
func (r *Reader) ReadTime(n int, dst *time.Time) error {
	if err := r.ensureParsed(); err != nil {
		return err
	}
	return r.readField(n, dst, "time")
}

// PresentFields returns the sorted list of field indices present in the
// message, or an error if parsing failed. Must be called after the first
// Read* call (which triggers parsing).
func (r *Reader) PresentFields() ([]int, error) {
	if err := r.ensureParsed(); err != nil {
		return nil, err
	}
	indices := make([]int, 0, len(r.raw))
	for n := range r.raw {
		indices = append(indices, n)
	}
	sort.Ints(indices)
	return indices, nil
}

// Bytes returns all raw wire bytes read so far (MTI + bitmap + fields).
func (r *Reader) Bytes() []byte {
	return r.buf.Bytes()
}

// Export returns a map of field index → string value for all fields read.
// Fields marked Secure are masked as "".
func (r *Reader) Export() map[int]string {
	out := make(map[int]string, len(r.strs))
	for n, s := range r.strs {
		fs := r.p.GetField(n)
		if fs != nil && fs.Secure {
			out[n] = ""
		} else {
			out[n] = s
		}
	}
	return out
}

// ── Internals ───────────────────────────────────────────────────────────────────

// ensureParsed reads the bitmap and all present fields from the wire.
func (r *Reader) ensureParsed() error {
	if r.parsed {
		return nil
	}
	if r.err != nil {
		return r.err
	}
	if !r.mtiRead {
		return fmt.Errorf("reader: MTI must be read before fields")
	}

	// Read bitmap.
	bm, bmBytes, err := r.readBitmap()
	if err != nil {
		r.err = err
		return err
	}
	r.buf.Write(bmBytes)

	// Get present fields in index order.
	present := bm.PresentFields()

	// Read each present field.
	for _, n := range present {
		raw, err := r.readRawField(n)
		if err != nil {
			r.err = err
			return err
		}
		r.raw[n] = raw
	}

	r.parsed = true
	return nil
}

// readBitmap reads and parses the bitmap from the wire.
func (r *Reader) readBitmap() (*Bitmap, []byte, error) {
	// Read primary bitmap (8 bytes).
	primary := make([]byte, 8)
	if _, err := io.ReadFull(r.r, primary); err != nil {
		return nil, nil, fmt.Errorf("bitmap: %w", err)
	}
	bm := &Bitmap{Primary: uint64(primary[0])<<56 | uint64(primary[1])<<48 |
		uint64(primary[2])<<40 | uint64(primary[3])<<32 |
		uint64(primary[4])<<24 | uint64(primary[5])<<16 |
		uint64(primary[6])<<8 | uint64(primary[7])}
	consumed := primary

	// Bit 1 indicates secondary bitmap.
	if bm.primaryBit(1) {
		secondary := make([]byte, 8)
		if _, err := io.ReadFull(r.r, secondary); err != nil {
			return nil, nil, fmt.Errorf("bitmap secondary: %w", err)
		}
		bm.Secondary = uint64(secondary[0])<<56 | uint64(secondary[1])<<48 |
			uint64(secondary[2])<<40 | uint64(secondary[3])<<32 |
			uint64(secondary[4])<<24 | uint64(secondary[5])<<16 |
			uint64(secondary[6])<<8 | uint64(secondary[7])
		consumed = append(consumed, secondary...)
	}

	return bm, consumed, nil
}

// readRawField reads one field's raw content bytes from the wire.
func (r *Reader) readRawField(n int) ([]byte, error) {
	fs := r.p.GetField(n)
	if fs == nil {
		return nil, fmt.Errorf("field %d not defined in protocol %q", n, r.p.Name)
	}
	if fs.Len == nil || fs.Value == nil {
		return nil, fmt.Errorf("field %d (%s): incomplete definition (nil Len or Value)", n, fs.Name)
	}

	var contentLen int

	if fs.Len.IsFixed() {
		contentLen = wireByteLen(fs)
	} else {
		var err error
		contentLen, err = fs.Len.ReadLen(r.r)
		if err != nil {
			return nil, fmt.Errorf("field %d (%s): %w", n, fs.Name, err)
		}
	}

	if contentLen <= 0 {
		return nil, fmt.Errorf("field %d (%s): invalid content length %d", n, fs.Name, contentLen)
	}

	content := make([]byte, contentLen)
	if _, err := io.ReadFull(r.r, content); err != nil {
		return nil, fmt.Errorf("field %d (%s): %w", n, fs.Name, err)
	}

	// Accumulate the length prefix bytes + content bytes for Bytes().
	// Re-create what was on the wire.
	if !fs.Len.IsFixed() {
		var prefixBuf bytes.Buffer
		fs.Len.WriteLen(&prefixBuf, contentLen)
		r.buf.Write(prefixBuf.Bytes())
	}
	r.buf.Write(content)

	return content, nil
}

// readField decodes a previously-read raw field, validates, and sets dst.
func (r *Reader) readField(n int, dst any, kind string) error {
	raw, ok := r.raw[n]
	if !ok {
		return fmt.Errorf("field %d not present in message", n)
	}

	fs := r.p.GetField(n)
	if fs == nil {
		return fmt.Errorf("field %d not defined in protocol", n)
	}

	// Decode.
	decoded, err := fs.Value.Decode(raw)
	if err != nil {
		return fmt.Errorf("field %d (%s): decode: %w", n, fs.Name, err)
	}

	// Convert to string for validation and storage.
	strVal, err := fieldToString(decoded)
	if err != nil {
		return fmt.Errorf("field %d (%s): %w", n, fs.Name, err)
	}

	// Validate.
	if fs.Valid != nil && !fs.Valid.Ok(strVal) {
		display := strVal
		if fs.Secure && len(strVal) > 4 {
			display = "****" + strVal[len(strVal)-4:]
		} else if fs.Secure {
			display = "****"
		}
		return fmt.Errorf("field %d (%s): validation failed: %s (got %q)", n, fs.Name, fs.Valid.String(), display)
	}

	// Store string for Export.
	r.strs[n] = strVal

	// Mark field as consumed (prevent double-read).
	delete(r.raw, n)

	// Coerce decoded value to the target kind.
	targetVal, err := coerceValue(decoded, kind)
	if err != nil {
		return fmt.Errorf("field %d (%s): %w", n, fs.Name, err)
	}

	// Set destination.
	return setDst(dst, targetVal, kind)
}

// coerceValue converts a decoded value to the target kind expected by the caller.
func coerceValue(v any, kind string) (any, error) {
	switch kind {
	case "string":
		s, err := fieldToString(v)
		if err != nil {
			return nil, err
		}
		return s, nil
	case "int":
		switch val := v.(type) {
		case int64:
			return val, nil
		case string:
			n, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("cannot parse %q as int64", val)
			}
			return n, nil
		default:
			return nil, fmt.Errorf("cannot convert %T to int64", v)
		}
	case "bytes":
		switch val := v.(type) {
		case []byte:
			return val, nil
		case string:
			return []byte(val), nil
		default:
			return nil, fmt.Errorf("cannot convert %T to []byte", v)
		}
	case "time":
		s, err := fieldToString(v)
		if err != nil {
			return nil, err
		}
		t, err := parseTime(s)
		if err != nil {
			return nil, err
		}
		return t, nil
	default:
		return nil, fmt.Errorf("unknown kind %q", kind)
	}
}

// fieldToString converts a decoded Go value to its string representation.
func fieldToString(v any) (string, error) {
	switch val := v.(type) {
	case string:
		return val, nil
	case int64:
		return fmt.Sprintf("%d", val), nil
	case []byte:
		return fmt.Sprintf("%x", val), nil
	default:
		return "", fmt.Errorf("cannot convert %T to string", v)
	}
}

// setDst sets a destination pointer to the coerced value.
func setDst(dst any, val any, kind string) error {
	switch kind {
	case "string":
		d, ok := dst.(*string)
		if !ok {
			return fmt.Errorf("expected *string destination, got %T", dst)
		}
		v, ok := val.(string)
		if !ok {
			return fmt.Errorf("expected string value, got %T", val)
		}
		*d = v
	case "int":
		d, ok := dst.(*int64)
		if !ok {
			return fmt.Errorf("expected *int64 destination, got %T", dst)
		}
		v, ok := val.(int64)
		if !ok {
			return fmt.Errorf("expected int64 value, got %T", val)
		}
		*d = v
	case "bytes":
		d, ok := dst.(*[]byte)
		if !ok {
			return fmt.Errorf("expected *[]byte destination, got %T", dst)
		}
		v, ok := val.([]byte)
		if !ok {
			return fmt.Errorf("expected []byte value, got %T", val)
		}
		*d = v
	case "time":
		d, ok := dst.(*time.Time)
		if !ok {
			return fmt.Errorf("expected *time.Time destination, got %T", dst)
		}
		v, ok := val.(time.Time)
		if !ok {
			return fmt.Errorf("expected time.Time value, got %T", val)
		}
		*d = v
	default:
		return fmt.Errorf("unknown destination kind %q", kind)
	}
	return nil
}

// parseTime tries common ISO 8583 time formats.
func parseTime(s string) (time.Time, error) {
	formats := []string{
		"0102150405",       // MMDDHHMMSS
		"20060102150405",    // YYYYMMDDHHMMSS
		"060102150405",      // YYMMDDHHMMSS
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse time %q", s)
}

// decodeMTI reads and decodes an MTI from the wire.
func decodeMTI(r io.Reader, enc spec.MTIEncoder) (uint, []byte, error) {
	b := make([]byte, enc.WireLen())
	if _, err := io.ReadFull(r, b); err != nil {
		return 0, nil, fmt.Errorf("MTI: %w", err)
	}
	mti, err := enc.Decode(b)
	if err != nil {
		return 0, nil, fmt.Errorf("MTI: %w", err)
	}
	return mti, b, nil
}
