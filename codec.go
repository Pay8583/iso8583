package iso8583

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"time"

	"github.com/Pay8583/iso8583/encoding"
	"github.com/Pay8583/iso8583/spec"
)

// Marshal serializes a Go struct into ISO 8583 wire format bytes using the given spec.
// The struct must have iso8583 tags on the fields to map.
func Marshal(v any, s *spec.Spec) ([]byte, error) {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("marshal: expected struct, got %s", rv.Kind())
	}

	fi, err := collectStructFields(rv)
	if err != nil {
		return nil, err
	}
	return marshalFields(fi, s)
}

// Unmarshal deserializes ISO 8583 wire format bytes into a Go struct pointer using the given spec.
// The value must be a non-nil pointer to a struct with iso8583 tags.
func Unmarshal(data []byte, v any, s *spec.Spec) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("unmarshal: expected non-nil pointer to struct, got %s", rv.Kind())
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("unmarshal: expected pointer to struct, got pointer to %s", rv.Kind())
	}

	msg, err := UnpackMessage(data, s)
	if err != nil {
		return err
	}
	return unmarshalFields(rv, msg, s)
}

// fieldInfo holds the parsed information for one struct field.
type fieldInfo struct {
	reflectValue reflect.Value
	tag          *ParsedTag
	fieldSpec    *spec.FieldSpec // may be nil for MTI
}

// collectStructFields iterates over a struct and extracts fields with iso8583 tags.
func collectStructFields(rv reflect.Value) ([]fieldInfo, error) {
	rt := rv.Type()
	var fields []fieldInfo
	for i := range rt.NumField() {
		fv := rv.Field(i)
		ft := rt.Field(i)
		tagStr, ok := ft.Tag.Lookup("iso8583")
		if !ok || tagStr == "" {
			continue
		}
		pt, err := ParseTag(tagStr)
		if err != nil {
			return nil, fmt.Errorf("field %s: %w", ft.Name, err)
		}
		fields = append(fields, fieldInfo{reflectValue: fv, tag: pt})
	}
	return fields, nil
}

// marshalFields packs a collection of struct fields into wire bytes.
func marshalFields(fields []fieldInfo, s *spec.Spec) ([]byte, error) {
	var mti string
	fieldValues := make(map[int]string)
	resolvedSpecs := make(map[int]*spec.FieldSpec)

	for _, fi := range fields {
		if fi.tag.IsMTI {
			mti = fi.reflectValue.String()
			continue
		}
		if fi.tag.FieldNumber < 2 {
			continue
		}
		n := fi.tag.FieldNumber
		sf := s.GetField(n)
		if sf == nil && fi.tag.IsMTI {
			continue
		}
		resolved := fi.tag.ResolveFieldSpec(sf)
		if resolved == nil {
			// Tag points to a field not in spec and no spec default exists.
			// Create a minimal spec from the tag hints.
			resolved = tagToFieldSpec(fi.tag)
		}
		resolvedSpecs[n] = resolved

		val, err := goValueToString(fi.reflectValue)
		if err != nil {
			return nil, newError("marshal", n, err)
		}
		if val == "" || val == "0" {
			if resolved.Optional || fi.reflectValue.IsZero() {
				continue
			}
		}
		fieldValues[n] = val
	}

	if mti == "" {
		return nil, fmt.Errorf("marshal: no MTI field found (tag with \"mti\")")
	}
	if len(mti) != 4 {
		return nil, fmt.Errorf("marshal: MTI must be 4 bytes, got %q", mti)
	}

	// Build message-like structure for packing.
	msg := &Message{
		MTI:    mti,
		Fields: fieldValues,
		Spec:   s,
	}
	return packMessageWithResolved(msg, resolvedSpecs)
}

// packMessageWithResolved is like PackMessage but uses pre-resolved field specs
// (from tag overrides) instead of spec.Fields directly.
func packMessageWithResolved(msg *Message, resolved map[int]*spec.FieldSpec) ([]byte, error) {
	s := msg.Spec

	numbers := make([]int, 0, len(msg.Fields))
	for n := range msg.Fields {
		numbers = append(numbers, n)
	}
	sort.Ints(numbers)
	bm := BuildBitmap(numbers)

	// Pre-calculate size.
	total := 4 + len(bm.Bytes())
	for _, n := range numbers {
		fs := resolveField(msg.Spec, resolved, n)
		if fs == nil {
			continue
		}
		enc, err := fs.Encoder.Encode(msg.Fields[n])
		if err != nil {
			return nil, newError("encode", n, err)
		}
		switch fs.LengthType {
		case spec.Fixed:
			total += wireLen(fs)
		case spec.LLVAR:
			total += 1 + len(enc)
		case spec.LLLVAR:
			total += 2 + len(enc)
		case spec.LLLLVAR:
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
		fs := resolveField(s, resolved, n)
		buf, err = PackField(buf, fs, msg.Fields[n])
		if err != nil {
			return nil, err
		}
	}
	return buf, nil
}

// resolveField returns the resolved field spec: tag override first, then spec default.
func resolveField(s *spec.Spec, resolved map[int]*spec.FieldSpec, n int) *spec.FieldSpec {
	if fs, ok := resolved[n]; ok {
		return fs
	}
	return s.GetField(n)
}

// tagToFieldSpec creates a minimal FieldSpec from tag hints alone (no spec default).
func tagToFieldSpec(pt *ParsedTag) *spec.FieldSpec {
	fs := &spec.FieldSpec{Index: pt.FieldNumber, Name: pt.Name}
	if pt.LengthType != 0 {
		fs.LengthType = pt.LengthType
	} else {
		fs.LengthType = spec.Fixed
	}
	if pt.ContentType != 0 {
		fs.ContentType = pt.ContentType
	}
	if pt.EncoderName != "" {
		if e := encoding.Get(pt.EncoderName); e != nil {
			fs.Encoder = e
		}
	}
	if fs.Encoder == nil {
		fs.Encoder = encoding.MustGet("ascii")
	}
	if pt.FixedLen != 0 {
		fs.FixedLen = pt.FixedLen
	}
	if pt.MaxLen != 0 {
		fs.MaxLen = pt.MaxLen
	}
	fs.Optional = pt.Optional
	return fs
}

// unmarshalFields populates a struct from a decoded Message.
func unmarshalFields(rv reflect.Value, msg *Message, _ *spec.Spec) error {
	rt := rv.Type()
	for i := range rt.NumField() {
		fv := rv.Field(i)
		ft := rt.Field(i)
		tagStr, ok := ft.Tag.Lookup("iso8583")
		if !ok || tagStr == "" {
			continue
		}
		pt, err := ParseTag(tagStr)
		if err != nil {
			return fmt.Errorf("field %s: %w", ft.Name, err)
		}

		if pt.IsMTI {
			fv.SetString(msg.MTI)
			continue
		}
		if pt.FieldNumber < 2 {
			continue
		}

		valStr, ok := msg.Fields[pt.FieldNumber]
		if !ok {
			if pt.Optional {
				continue
			}
			// Check if the field has a zero value — it's OK if not required.
			if fv.IsZero() {
				continue
			}
		}
		if err := stringToGoValue(valStr, fv); err != nil {
			return newError("unmarshal", pt.FieldNumber, err)
		}
	}
	return nil
}

// goValueToString converts a reflect.Value to its string representation for ISO encoding.
func goValueToString(rv reflect.Value) (string, error) {
	if !rv.IsValid() {
		return "", nil
	}
	switch rv.Kind() {
	case reflect.String:
		return rv.String(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(rv.Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(rv.Uint(), 10), nil
	case reflect.Float32:
		return strconv.FormatFloat(rv.Float(), 'f', -1, 32), nil
	case reflect.Float64:
		return strconv.FormatFloat(rv.Float(), 'f', -1, 64), nil
	case reflect.Slice:
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			return string(rv.Bytes()), nil
		}
		return "", fmt.Errorf("unsupported slice type %s", rv.Type())
	case reflect.Struct:
		if rv.Type() == reflect.TypeOf(time.Time{}) {
			t := rv.Interface().(time.Time)
			return t.Format("0102150405"), nil // MMDDHHMMSS
		}
		return "", fmt.Errorf("unsupported struct type %s", rv.Type())
	default:
		return "", fmt.Errorf("unsupported type %s", rv.Kind())
	}
}

// stringToGoValue converts a string value to the appropriate Go type and sets it.
func stringToGoValue(s string, rv reflect.Value) error {
	if !rv.CanSet() {
		return nil
	}
	switch rv.Kind() {
	case reflect.String:
		rv.SetString(s)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid integer %q: %w", s, err)
		}
		rv.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid unsigned integer %q: %w", s, err)
		}
		rv.SetUint(n)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return fmt.Errorf("invalid float %q: %w", s, err)
		}
		rv.SetFloat(f)
	case reflect.Slice:
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			rv.SetBytes([]byte(s))
		}
	case reflect.Struct:
		if rv.Type() == reflect.TypeOf(time.Time{}) {
			t, err := time.Parse("0102150405", s)
			if err != nil {
				// Try other common ISO 8583 date formats.
				t, err = time.Parse("20060102150405", s)
				if err != nil {
					t, err = time.Parse("060102150405", s)
					if err != nil {
						return fmt.Errorf("invalid time %q: %w", s, err)
					}
				}
			}
			rv.Set(reflect.ValueOf(t))
		}
	}
	return nil
}

// FormatValue converts a Go value to its string representation for ISO 8583 encoding.
// Supported types: string, int*, uint*, float*, []byte, time.Time.
func FormatValue(v any) (string, error) {
	return goValueToString(reflect.ValueOf(v))
}

// ParseValue parses an ISO 8583 string value into the given Go variable.
// dst must be a non-nil pointer.
func ParseValue(s string, dst any) error {
	rv := reflect.ValueOf(dst)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("parseValue: expected non-nil pointer, got %s", rv.Kind())
	}
	return stringToGoValue(s, rv.Elem())
}
