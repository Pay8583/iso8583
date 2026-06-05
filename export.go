package iso8583

import (
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/Pay8583/iso8583/spec"
)

// ExportStruct returns a map of field index → string value for a struct
// tagged with iso8583 tags. Fields marked Secure in the protocol are
// replaced with "".
func ExportStruct(v any, p *spec.Protocol) (map[int]string, error) {
	return exportStruct(v, p, "")
}

// ExportStructMasked is like ExportStruct but uses the given mask string
// for Secure fields instead of "".
func ExportStructMasked(v any, p *spec.Protocol, mask string) (map[int]string, error) {
	return exportStruct(v, p, mask)
}

func exportStruct(v any, p *spec.Protocol, mask string) (map[int]string, error) {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("export: expected struct, got %s", rv.Kind())
	}

	out := make(map[int]string)
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
			return nil, fmt.Errorf("export: field %s: %w", ft.Name, err)
		}

		if pt.IsMTI {
			continue // MTI not exported
		}
		if pt.FieldNumber < 2 {
			continue
		}

		// Check if the field is secure — tag override wins over protocol.
		fs := p.GetField(pt.FieldNumber)
		secure := false
		if pt.Secure != nil {
			secure = *pt.Secure // tag-level secure flag
		} else if fs != nil {
			secure = fs.Secure // protocol-level secure flag
		}
		if secure {
			out[pt.FieldNumber] = mask
			continue
		}

		// Convert Go value to string.
		s, err := goValueToString(fv)
		if err != nil {
			return nil, fmt.Errorf("export: field %s: %w", ft.Name, err)
		}
		out[pt.FieldNumber] = s
	}

	return out, nil
}

// goValueToString converts a reflect.Value to its string representation.
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
			return fmt.Sprintf("%x", rv.Bytes()), nil
		}
		return "", fmt.Errorf("unsupported slice type %s", rv.Type())
	case reflect.Struct:
		if rv.Type() == reflect.TypeOf(time.Time{}) {
			t := rv.Interface().(time.Time)
			return t.Format("0102150405"), nil
		}
		return "", fmt.Errorf("unsupported struct type %s", rv.Type())
	default:
		return "", fmt.Errorf("unsupported type %s", rv.Kind())
	}
}
