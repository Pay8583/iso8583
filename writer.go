package iso8583

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Pay8583/iso8583/spec"
)

// A Writer packs ISO 8583 messages field by field. It accumulates typed field
// values, validates them on Close(), encodes everything to wire format, and
// writes to the underlying io.Writer.
//
// Bytes() returns the raw bytes accumulated (MTI + bitmap + every field byte)
// for MAC computation. Export() returns a map suitable for JSON logging with
// security-sensitive fields masked.
type Writer struct {
	p      *spec.Protocol
	w      WriterTo
	buf    bytes.Buffer // raw wire bytes accumulator
	mti    uint
	setMTI bool
	fields map[int]fieldValue
	closed bool
}

type fieldValue struct {
	val  any    // the Go value
	kind string // "string", "int", "bytes", "time"
}

// WriterTo is the interface implemented by objects that can have ISO 8583
// bytes written to them. io.Writer satisfies this.
type WriterTo interface {
	Write([]byte) (int, error)
}

// NewWriter returns a Writer that packs messages according to p and writes
// the wire-format output to w.
func NewWriter(p *spec.Protocol, w WriterTo) *Writer {
	return &Writer{
		p:      p,
		w:      w,
		fields: make(map[int]fieldValue),
	}
}

// WriteMTI sets the message MTI.
func (w *Writer) WriteMTI(mti uint) error {
	if w.closed {
		return fmt.Errorf("writer: already closed")
	}
	if w.setMTI {
		return fmt.Errorf("writer: MTI already set")
	}
	w.mti = mti
	w.setMTI = true
	return nil
}

// WriteString writes a string value for ISO field n.
func (w *Writer) WriteString(n int, v string) error {
	return w.writeField(n, v, "string")
}

// WriteInt writes an int64 value for ISO field n.
func (w *Writer) WriteInt(n int, v int64) error {
	return w.writeField(n, v, "int")
}

// WriteBytes writes a []byte value for ISO field n.
func (w *Writer) WriteBytes(n int, v []byte) error {
	return w.writeField(n, v, "bytes")
}

// WriteTime writes a time.Time value for ISO field n.
func (w *Writer) WriteTime(n int, v time.Time) error {
	return w.writeField(n, v, "time")
}

func (w *Writer) writeField(n int, v any, kind string) error {
	if w.closed {
		return fmt.Errorf("writer: already closed")
	}
	if n < 2 || n > 128 {
		return fmt.Errorf("writer: field %d out of range [2,128]", n)
	}
	if _, dup := w.fields[n]; dup {
		return fmt.Errorf("writer: field %d written twice", n)
	}
	w.fields[n] = fieldValue{val: v, kind: kind}
	return nil
}

// Close validates all field values, encodes them to wire format, builds the
// bitmap, and writes the complete message to the underlying writer. After
// Close, no more fields may be written.
func (w *Writer) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true

	if !w.setMTI {
		return fmt.Errorf("writer: MTI not set")
	}

	// Encode MTI.
	mtiBytes, err := w.p.MTI.Encode(w.mti)
	if err != nil {
		return err
	}
	w.buf.Write(mtiBytes)

	// Sort field indices.
	indices := make([]int, 0, len(w.fields))
	for n := range w.fields {
		indices = append(indices, n)
	}
	sort.Ints(indices)

	// Build bitmap.
	bm := &Bitmap{}
	for _, n := range indices {
		bm.Set(n)
	}

	// Write bitmap.
	bmBytes := bm.Bytes()
	w.buf.Write(bmBytes)

	// Write each field.
	for _, n := range indices {
		fv := w.fields[n]
		fs := w.p.GetField(n)
		if fs == nil {
			return fmt.Errorf("writer: field %d not defined in protocol %q", n, w.p.Name)
		}

		fieldBytes, err := w.encodeField(fs, fv, n)
		if err != nil {
			return fmt.Errorf("writer: field %d (%s): %w", n, fs.Name, err)
		}
		w.buf.Write(fieldBytes)
	}

	// Write to underlying writer.
	_, err = w.w.Write(w.buf.Bytes())
	return err
}

// encodeField encodes a single field value to wire bytes.
func (w *Writer) encodeField(fs *spec.Field, fv fieldValue, n int) ([]byte, error) {
	if fs.Len == nil || fs.Value == nil {
		return nil, fmt.Errorf("field %d (%s): incomplete definition (nil Len or Value)", n, fs.Name)
	}

	var (
		encoded []byte
		err     error
	)

	// Convert Go value and validate.
	strVal, err := w.validateField(fs, fv, n)
	if err != nil {
		return nil, err
	}

	// For fixed-length fields, pad the string value before encoding.
	// RBCD, Hex, Raw, Binary handle their own padding; others need pre-padding.
	fixedLen := fs.Len.FixedLen()
	selfPadding := isSelfPaddingValue(fs.Value)

	if fs.Len.IsFixed() && fixedLen > 0 && len(strVal) < fixedLen && !selfPadding {
		switch {
		case isNumericValidator(fs.Valid):
			strVal = strings.Repeat("0", fixedLen-len(strVal)) + strVal
		case fs.Value.String() == "Hex":
			strVal = strings.Repeat("0", fixedLen-len(strVal)) + strVal
		default:
			strVal = strVal + strings.Repeat(" ", fixedLen-len(strVal))
		}
	}

	// Encode the string value using the field's Value encoding.
	encoded, err = fs.Value.Encode(strVal, fixedLen)
	if err != nil {
		return nil, fmt.Errorf("encode: %w", err)
	}

	// For fixed-length BCD/RBCD fields, ensure byte count is correct.
	if fs.Len.IsFixed() {
		expected := wireByteLen(fs)
		if len(encoded) < expected {
			// Pad bytes after encoding (for BCD nibble alignment etc.).
			pad := fs.Len.Pad()
			encoded = append(encoded, bytes.Repeat([]byte{pad}, expected-len(encoded))...)
		} else if len(encoded) > expected {
			encoded = encoded[:expected]
		}
	}

	// Build field bytes: length prefix (if variable) + content.
	var fieldBuf bytes.Buffer
	if !fs.Len.IsFixed() {
		if err := fs.Len.WriteLen(&fieldBuf, len(encoded)); err != nil {
			return nil, fmt.Errorf("write length: %w", err)
		}
	}
	fieldBuf.Write(encoded)

	return fieldBuf.Bytes(), nil
}

// isSelfPaddingValue reports whether a Value handles its own string-level
// padding (RBCD right-justifies with zeros, Hex is byte-oriented, Raw is
// passthrough, Binary is integer-to-bytes).
func isSelfPaddingValue(v spec.Value) bool {
	if v == nil {
		return false
	}
	name := v.String()
	if len(name) >= 4 && name[:4] == "RBCD" {
		return true
	}
	switch name {
	case "Hex", "Raw", "Binary":
		return true
	}
	return false
}

// isNumericValidator reports whether the validator is for numeric content.
func isNumericValidator(v spec.Validator) bool {
	if v == nil {
		return false
	}
	switch v.String() {
	case "N", "XN", "NS":
		return true
	}
	return false
}

// validateField converts a Go value to its string representation and
// validates it against the field's validator.
func (w *Writer) validateField(fs *spec.Field, fv fieldValue, _ int) (string, error) {
	var strVal string

	switch fv.kind {
	case "string":
		strVal = fv.val.(string)
	case "int":
		v := fv.val.(int64)
		strVal = fmt.Sprintf("%d", v)
	case "bytes":
		b := fv.val.([]byte)
		strVal = string(b)
	case "time":
		t := fv.val.(time.Time)
		strVal = t.Format("0102150405") // MMDDHHMMSS
	default:
		return "", fmt.Errorf("unknown field kind %q", fv.kind)
	}

	if fs.Valid != nil && !fs.Valid.Ok(strVal) {
		display := strVal
		if fs.Secure && len(strVal) > 4 {
			display = "****" + strVal[len(strVal)-4:]
		} else if fs.Secure {
			display = "****"
		}
		return "", fmt.Errorf("validation failed: %s (got %q)", fs.Valid.String(), display)
	}

	return strVal, nil
}

// Bytes returns all raw wire bytes accumulated so far (MTI + bitmap +
// encoded fields). Call after Close() to get the complete message bytes
// for MAC computation.
func (w *Writer) Bytes() []byte {
	return w.buf.Bytes()
}

// Export returns a map of field index → string value for all fields
// written. Fields marked Secure in the protocol are masked as "".
func (w *Writer) Export() map[int]string {
	out := make(map[int]string, len(w.fields))
	for n, fv := range w.fields {
		fs := w.p.GetField(n)
		if fs != nil && fs.Secure {
			out[n] = ""
			continue
		}
		out[n] = formatFieldValue(fv)
	}
	return out
}

// ── Helpers ─────────────────────────────────────────────────────────────────────

// formatFieldValue returns a string representation of a field value for export.
func formatFieldValue(fv fieldValue) string {
	switch fv.kind {
	case "string":
		return fv.val.(string)
	case "int":
		return fmt.Sprintf("%d", fv.val.(int64))
	case "bytes":
		return fmt.Sprintf("%x", fv.val.([]byte))
	case "time":
		return fv.val.(time.Time).Format("0102150405")
	default:
		return ""
	}
}

// wireByteLen returns the number of wire bytes a fixed-length field occupies.
func wireByteLen(fs *spec.Field) int {
	if !fs.Len.IsFixed() {
		return 0
	}
	n := fs.Len.FixedLen()
	// Determine byte count from the value encoding.
	name := fs.Value.String()
	// Handle parameterized names: "RBCD(12)" → half-width, "Binary(4)" → 4 bytes.
	if len(name) >= 4 && name[:4] == "RBCD" {
		return (n + 1) / 2
	}
	switch name {
	case "BCD":
		return (n + 1) / 2
	case "Hex":
		return (n + 1) / 2 // hex: 2 chars per byte
	case "ASCII", "Text", "Raw", "EBCDIC":
		return n
	default:
		// Binary(N) or unknown: use FixedLen as byte count.
		if len(name) >= 6 && name[:6] == "Binary" {
			return n
		}
		return n
	}
}
