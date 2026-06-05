package iso8583

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/Pay8583/iso8583/spec"
)

// Marshaler is the interface implemented by types that can marshal themselves
// to ISO 8583 wire bytes. The code generator (iso8583gen) produces methods
// that satisfy this interface, avoiding reflection overhead.
type Marshaler interface {
	MarshalISO8583(*spec.Protocol) ([]byte, error)
}

// Unmarshaler is the interface implemented by types that can unmarshal
// themselves from ISO 8583 wire bytes.
type Unmarshaler interface {
	UnmarshalISO8583([]byte, *spec.Protocol) error
}

// Marshal serializes a Go struct to ISO 8583 wire bytes using the Protocol
// for field layout and the struct's iso8583 tags for field mapping.
// Tag options override Protocol field definitions.
// Zero-value fields are omitted from the output.
//
// If v implements Marshaler, Marshal calls v.MarshalISO8583(p) directly,
// bypassing reflection. The code generator produces types that satisfy this
// interface.
func Marshal(v any, p *spec.Protocol) ([]byte, error) {
	if m, ok := v.(Marshaler); ok {
		return m.MarshalISO8583(p)
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("marshal: expected struct, got %s", rv.Kind())
	}

	fields, mti, resolved, err := collectFields(rv, p)
	if err != nil {
		return nil, err
	}

	// Build a merged protocol with tag overrides applied.
	merged := mergeProtocol(p, resolved)

	var buf bytes.Buffer
	w := NewWriter(merged, &buf)

	if err := w.WriteMTI(mti); err != nil {
		return nil, err
	}

	for _, fi := range fields {
		if fi.tag.IsMTI {
			continue
		}
		// Skip zero-value fields (they're not present in the message).
		if isZero(fi.rv) {
			continue
		}
		if err := writeFieldValue(w, fi); err != nil {
			return nil, err
		}
	}

	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Unmarshal deserializes ISO 8583 wire bytes into a Go struct pointer using
// the Protocol for field layout and the struct's iso8583 tags for field
// mapping.
// Unmarshal deserializes ISO 8583 wire bytes into a Go struct pointer using
// the Protocol for field layout and the struct's iso8583 tags for field
// mapping.
//
// If v implements Unmarshaler, Unmarshal calls v.UnmarshalISO8583(data, p)
// directly, bypassing reflection.
func Unmarshal(data []byte, v any, p *spec.Protocol) error {
	if u, ok := v.(Unmarshaler); ok {
		return u.UnmarshalISO8583(data, p)
	}

	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("unmarshal: expected non-nil pointer to struct, got %s", rv.Kind())
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("unmarshal: expected pointer to struct, got pointer to %s", rv.Kind())
	}

	// Resolve tag overrides and build a merged protocol.
	rt := rv.Type()
	resolved := make(map[int]*spec.Field)
	fieldMap := make(map[int]structField)

	for i := range rt.NumField() {
		fv := rv.Field(i)
		ft := rt.Field(i)
		tagStr, ok := ft.Tag.Lookup("iso8583")
		if !ok || tagStr == "" {
			continue
		}
		pt, err := ParseTag(tagStr)
		if err != nil {
			return fmt.Errorf("%s: %w", ft.Name, err)
		}
		if pt.IsMTI {
			continue
		}
		if pt.FieldNumber < 2 || !fv.CanSet() {
			continue
		}
		protoField := p.GetField(pt.FieldNumber)
		rf, err := pt.ResolveField(protoField)
		if err != nil {
			return fmt.Errorf("%s: %w", ft.Name, err)
		}
		resolved[pt.FieldNumber] = rf
		fieldMap[pt.FieldNumber] = structField{tag: pt, rv: fv}
	}

	merged := mergeProtocol(p, resolved)

	r := NewReader(merged, bytes.NewReader(data))

	mti, err := r.ReadMTI()
	if err != nil {
		return err
	}

	// Set MTI on struct if there's an MTI-tagged field.
	for i := range rt.NumField() {
		fv := rv.Field(i)
		ft := rt.Field(i)
		tagStr, ok := ft.Tag.Lookup("iso8583")
		if !ok || tagStr == "" {
			continue
		}
		pt, err := ParseTag(tagStr)
		if err != nil {
			return fmt.Errorf("%s: %w", ft.Name, err)
		}
		if pt.IsMTI && fv.CanSet() {
			switch fv.Kind() {
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				fv.SetUint(uint64(mti))
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				fv.SetInt(int64(mti))
			case reflect.String:
				fv.SetString(fmt.Sprintf("%04X", mti))
			}
		}
	}

	// Read only fields that are present in the message.
	present, err := r.PresentFields()
	if err != nil {
		return err
	}
	for _, n := range present {
		sf, ok := fieldMap[n]
		if !ok {
			continue // field in message but not in struct — skip
		}
		if err := readFieldValue(r, n, sf); err != nil {
			return err
		}
	}

	return nil
}

// isZero reports whether a reflect.Value is the zero value for its type.
func isZero(rv reflect.Value) bool {
	return rv.IsZero()
}

// ── Internal types ─────────────────────────────────────────────────────────────

type structField struct {
	tag *ParsedTag
	rv  reflect.Value
}

// ── Marshal helpers ────────────────────────────────────────────────────────────

// collectFields extracts iso8583-tagged fields from a struct and resolves
// them against the Protocol. Returns struct fields, the MTI value, and a map
// of tag-resolved field definitions (index → *Field).
func collectFields(rv reflect.Value, p *spec.Protocol) ([]structField, uint, map[int]*spec.Field, error) {
	rt := rv.Type()
	var fields []structField
	var mti uint
	hasMTI := false
	resolved := make(map[int]*spec.Field)

	for i := range rt.NumField() {
		fv := rv.Field(i)
		ft := rt.Field(i)
		tagStr, ok := ft.Tag.Lookup("iso8583")
		if !ok || tagStr == "" {
			continue
		}
		pt, err := ParseTag(tagStr)
		if err != nil {
			return nil, 0, nil, fmt.Errorf("%s: %w", ft.Name, err)
		}

		if pt.IsMTI {
			mtiVal, err := extractMTI(fv)
			if err != nil {
				return nil, 0, nil, fmt.Errorf("%s: %w", ft.Name, err)
			}
			mti = mtiVal
			hasMTI = true
			continue
		}
		if pt.FieldNumber < 2 {
			continue
		}

		// Resolve field definition: tag overrides protocol.
		protoField := p.GetField(pt.FieldNumber)
		rf, err := pt.ResolveField(protoField)
		if err != nil {
			return nil, 0, nil, fmt.Errorf("%s: %w", ft.Name, err)
		}
		resolved[pt.FieldNumber] = rf
		fields = append(fields, structField{tag: pt, rv: fv})
	}

	if !hasMTI {
		return nil, 0, nil, fmt.Errorf("marshal: no MTI field found (tag with \"mti\")")
	}

	return fields, mti, resolved, nil
}

// mergeProtocol creates a copy of p with field overrides from resolved applied.
func mergeProtocol(p *spec.Protocol, resolved map[int]*spec.Field) *spec.Protocol {
	if len(resolved) == 0 {
		return p
	}
	// Clone fields.
	merged := *p
	merged.Fields = make([]spec.Field, len(p.Fields))
	copy(merged.Fields, p.Fields)

	for n, f := range resolved {
		idx := n - 2
		if idx >= 0 && idx < len(merged.Fields) {
			merged.Fields[idx] = *f
		} else if idx >= len(merged.Fields) {
			// Extend the slice to include the new field.
			extended := make([]spec.Field, idx+1)
			copy(extended, merged.Fields)
			extended[idx] = *f
			merged.Fields = extended
		}
	}
	return &merged
}

// extractMTI extracts the MTI as uint from a reflect.Value.
func extractMTI(rv reflect.Value) (uint, error) {
	switch rv.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return uint(rv.Uint()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return uint(rv.Int()), nil
	case reflect.String:
		s := rv.String()
		var mti uint
		if _, err := fmt.Sscanf(s, "%04X", &mti); err != nil {
			return 0, fmt.Errorf("invalid MTI string %q", s)
		}
		return mti, nil
	default:
		return 0, fmt.Errorf("MTI field must be uint or string, got %s", rv.Kind())
	}
}

// writeFieldValue writes a single struct field to the Writer.
func writeFieldValue(w *Writer, fi structField) error {
	n := fi.tag.FieldNumber
	rv := fi.rv

	switch rv.Kind() {
	case reflect.String:
		return w.WriteString(n, rv.String())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return w.WriteInt(n, rv.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return w.WriteInt(n, int64(rv.Uint()))
	case reflect.Float32, reflect.Float64:
		return w.WriteString(n, strconv.FormatFloat(rv.Float(), 'f', -1, 64))
	case reflect.Slice:
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			return w.WriteBytes(n, rv.Bytes())
		}
		return fmt.Errorf("unsupported slice type %s", rv.Type())
	case reflect.Struct:
		if rv.Type() == reflect.TypeOf(time.Time{}) {
			return w.WriteTime(n, rv.Interface().(time.Time))
		}
		return fmt.Errorf("unsupported struct type %s", rv.Type())
	default:
		return fmt.Errorf("unsupported type %s for field %d", rv.Kind(), n)
	}
}

// ── Unmarshal helpers ──────────────────────────────────────────────────────────

// readFieldValue reads one field from the reader into a struct field.
func readFieldValue(r *Reader, n int, sf structField) error {
	rv := sf.rv
	switch rv.Kind() {
	case reflect.String:
		var s string
		if err := r.ReadString(n, &s); err != nil {
			return err
		}
		rv.SetString(s)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var v int64
		if err := r.ReadInt(n, &v); err != nil {
			return err
		}
		rv.SetInt(v)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		var v int64
		if err := r.ReadInt(n, &v); err != nil {
			return err
		}
		rv.SetUint(uint64(v))
	case reflect.Float32, reflect.Float64:
		var s string
		if err := r.ReadString(n, &s); err != nil {
			return err
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return fmt.Errorf("field %d: invalid float %q", n, s)
		}
		rv.SetFloat(f)
	case reflect.Slice:
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			var b []byte
			if err := r.ReadBytes(n, &b); err != nil {
				return err
			}
			rv.SetBytes(b)
		}
	case reflect.Struct:
		if rv.Type() == reflect.TypeOf(time.Time{}) {
			var t time.Time
			if err := r.ReadTime(n, &t); err != nil {
				return err
			}
			rv.Set(reflect.ValueOf(t))
		}
	default:
		return fmt.Errorf("unsupported type %s for field %d", rv.Kind(), n)
	}
	return nil
}
