package main

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strconv"
	"strings"
)

// structField holds parsed info about one struct field with an iso8583 tag.
type structField struct {
	GoName   string
	GoType   string
	FieldNum int  // field number 2-128, or 0 for MTI
	IsMTI    bool
	Tag      string // raw tag value
}

// parsedStruct holds all iso8583-tagged fields of a struct.
type parsedStruct struct {
	Name   string
	Fields []structField
}

// Generate reads the Go source file, finds the requested struct types, and writes
// generated MarshalISO8583/UnmarshalISO8583 methods to the output file.
func Generate(inputFile, outputFile string, typeNames []string) error {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, inputFile, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parse %s: %w", inputFile, err)
	}

	pkgName := f.Name.Name
	want := make(map[string]bool, len(typeNames))
	for _, t := range typeNames {
		want[t] = true
	}

	var structs []parsedStruct
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if !want[ts.Name.Name] {
				continue
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}
			ps := parsedStruct{Name: ts.Name.Name}
			for _, fld := range st.Fields.List {
				if fld.Tag == nil {
					continue
				}
				tagStr := strings.Trim(fld.Tag.Value, "`")
				tag := reflectLikeTag(tagStr)
				isoTag := tag.Get("iso8583")
				if isoTag == "" {
					continue
				}
				sf := structField{
					GoName: fld.Names[0].Name,
					GoType: typeExpr(fld.Type),
					Tag:    isoTag,
				}
				if isoTag == "mti" || strings.HasPrefix(isoTag, "mti,") {
					sf.IsMTI = true
				} else {
					parts := strings.SplitN(isoTag, ",", 2)
					n, err := strconv.Atoi(strings.TrimSpace(parts[0]))
					if err == nil && n >= 2 && n <= 128 {
						sf.FieldNum = n
					}
				}
				ps.Fields = append(ps.Fields, sf)
			}
			structs = append(structs, ps)
		}
	}

	if len(structs) == 0 {
		return fmt.Errorf("no structs found with requested types %v", typeNames)
	}

	src := generateCode(pkgName, structs)
	formatted, err := format.Source([]byte(src))
	if err != nil {
		_ = os.WriteFile(outputFile, []byte(src), 0644)
		return fmt.Errorf("format generated code: %w\nraw source written to %s", err, outputFile)
	}

	if err := os.WriteFile(outputFile, formatted, 0644); err != nil {
		return err
	}
	fmt.Printf("iso8583gen: wrote %s (%d structs)\n", outputFile, len(structs))
	return nil
}

// reflectLikeTag is a minimal reflect.StructTag-like parser.
type reflectLikeTag string

func (t reflectLikeTag) Get(key string) string {
	s := string(t)
	for s != "" {
		for len(s) > 0 && s[0] == ' ' {
			s = s[1:]
		}
		if s == "" {
			break
		}
		i := 0
		for i < len(s) && s[i] != ':' && s[i] != ' ' && s[i] != '"' {
			i++
		}
		if i == 0 || i+1 >= len(s) || s[i] != ':' {
			break
		}
		k := s[:i]
		if k != key {
			s = s[i+1:]
			if len(s) > 0 && s[0] == '"' {
				s = s[1:]
				for len(s) > 0 && s[0] != '"' {
					s = s[1:]
				}
				if len(s) > 0 {
					s = s[1:]
				}
			}
			continue
		}
		s = s[i+1:]
		if len(s) == 0 || s[0] != '"' {
			break
		}
		s = s[1:]
		j := 0
		for j < len(s) && s[j] != '"' {
			j++
		}
		return s[:j]
	}
	return ""
}

// typeExpr converts an AST type expression to its string representation.
func typeExpr(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + typeExpr(t.X)
	case *ast.SelectorExpr:
		return typeExpr(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + typeExpr(t.Elt)
		}
		return "[" + typeExpr(t.Len) + "]" + typeExpr(t.Elt)
	case *ast.BasicLit:
		return t.Value
	default:
		return fmt.Sprintf("%T", expr)
	}
}

// isStringish returns true for Go types that are string or alias thereof.
func isStringish(goType string) bool {
	return goType == "string"
}

// isIntish returns true for Go integer types.
func isIntish(goType string) bool {
	switch goType {
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64":
		return true
	}
	return false
}

// isByteSlice reports whether goType is []byte.
func isByteSlice(goType string) bool {
	return goType == "[]byte"
}

// zeroCheck returns a Go expression that checks whether the field is non-zero.
func zeroCheck(f structField) string {
	name := "m." + f.GoName
	switch {
	case isStringish(f.GoType):
		return name + ` != ""`
	case isIntish(f.GoType):
		return name + ` != 0`
	case isByteSlice(f.GoType):
		return name + ` != nil`
	default:
		return `true` // always include unknown types
	}
}

// ── Code generation ─────────────────────────────────────────────────────────────

func generateCode(pkgName string, structs []parsedStruct) string {
	var b strings.Builder
	b.WriteString("// Code generated by iso8583gen; DO NOT EDIT.\n\n")
	b.WriteString("package " + pkgName + "\n\n")
	b.WriteString(`import (
	"fmt"

	"github.com/Pay8583/iso8583"
	"github.com/Pay8583/iso8583/spec"
)
`)

	for _, ps := range structs {
		generateMarshal(&b, ps)
		generateUnmarshal(&b, ps)
	}
	return b.String()
}

func generateMarshal(b *strings.Builder, ps parsedStruct) {
	sorted := make([]structField, 0, len(ps.Fields))
	var mti *structField
	for i := range ps.Fields {
		if ps.Fields[i].IsMTI {
			f := ps.Fields[i]
			mti = &f
		} else {
			sorted = append(sorted, ps.Fields[i])
		}
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].FieldNum < sorted[j].FieldNum })

	fmt.Fprintf(b, `
func (m *%s) MarshalISO8583(s *spec.Spec) ([]byte, error) {
	var buf []byte

	// MTI
`, ps.Name)
	if mti != nil {
		fmt.Fprintf(b, `	buf = append(buf, m.%s...)
`, mti.GoName)
	} else {
		b.WriteString(`	buf = append(buf, "0200"...)
`)
	}

	// Bitmap assembly.
	b.WriteString("\tvar primary, secondary uint64\n")
	for _, f := range sorted {
		n := f.FieldNum
		bit := 63 - (n - 1)
		if n > 64 {
			bit = 63 - (n - 65)
		}
		fmt.Fprintf(b, `	if %s { `, zeroCheck(f))
		if n <= 64 {
			fmt.Fprintf(b, `primary |= 1 << %d`, bit)
		} else {
			fmt.Fprintf(b, `secondary |= 1 << %d`, bit)
		}
		b.WriteString(" }\n")
	}

	// Bitmap writing — only write secondary if needed.
	b.WriteString(`
	// Bitmap
	if secondary != 0 {
		primary |= 1 << 63
		buf = append(buf,
			byte(primary>>56), byte(primary>>48), byte(primary>>40), byte(primary>>32),
			byte(primary>>24), byte(primary>>16), byte(primary>>8),  byte(primary),
			byte(secondary>>56), byte(secondary>>48), byte(secondary>>40), byte(secondary>>32),
			byte(secondary>>24), byte(secondary>>16), byte(secondary>>8),  byte(secondary),
		)
	} else {
		buf = append(buf,
			byte(primary>>56), byte(primary>>48), byte(primary>>40), byte(primary>>32),
			byte(primary>>24), byte(primary>>16), byte(primary>>8),  byte(primary),
		)
	}
`)

	// Field encoding.
	b.WriteString("\n\t// Fields\n")
	for _, f := range sorted {
		fmt.Fprintf(b, `	// Field %d: %s
	if %s {
		fs := s.GetField(%d)
		if fs == nil { return nil, fmt.Errorf("no spec for field %%d", %d) }
		val, err := iso8583.FormatValue(m.%s)
		if err != nil { return nil, err }
		buf, err = iso8583.PackField(buf, fs, val)
		if err != nil { return nil, err }
	}
`, f.FieldNum, f.GoName, zeroCheck(f), f.FieldNum, f.FieldNum, f.GoName)
	}

	b.WriteString("\n\treturn buf, nil\n}\n")
}

func generateUnmarshal(b *strings.Builder, ps parsedStruct) {
	sorted := make([]structField, 0, len(ps.Fields))
	var mti *structField
	for i := range ps.Fields {
		if ps.Fields[i].IsMTI {
			f := ps.Fields[i]
			mti = &f
		} else {
			sorted = append(sorted, ps.Fields[i])
		}
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].FieldNum < sorted[j].FieldNum })

	fmt.Fprintf(b, `
func (m *%s) UnmarshalISO8583(data []byte, s *spec.Spec) error {
	if len(data) < 12 { return iso8583.ErrTruncated }

`, ps.Name)
	if mti != nil {
		fmt.Fprintf(b, `	m.%s = string(data[:4])
`, mti.GoName)
	}

	b.WriteString(`	// Bitmap
	bm, bmLen, err := iso8583.ParseBitmap(data[4:])
	if err != nil { return err }
	cursor := 4 + bmLen

	// Fields
`)
	for _, f := range sorted {
		fmt.Fprintf(b, `	if bm.IsSet(%d) {
		fs := s.GetField(%d)
		if fs == nil { return fmt.Errorf("no spec for field %%d", %d) }
		val, consumed, err := iso8583.UnpackField(data[cursor:], fs)
		if err != nil { return err }
		if err := iso8583.ParseValue(val, &m.%s); err != nil { return err }
		cursor += consumed
	}
`, f.FieldNum, f.FieldNum, f.FieldNum, f.GoName)
	}

	b.WriteString("\n\treturn nil\n}\n")
}
