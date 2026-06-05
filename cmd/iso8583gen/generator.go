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

// ── Type helpers ────────────────────────────────────────────────────────────────

func isStringish(goType string) bool { return goType == "string" }

func isUintish(goType string) bool {
	switch goType {
	case "uint", "uint8", "uint16", "uint32", "uint64":
		return true
	}
	return false
}

func isIntish(goType string) bool {
	switch goType {
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64":
		return true
	}
	return false
}

func isByteSlice(goType string) bool { return goType == "[]byte" }

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
		return `true`
	}
}

// writeMethod returns the Writer method call for a given Go type.
// e.g. ("m.PAN", "string") → `w.WriteString(%d, m.PAN)`
func writeMethod(name, goType string) string {
	switch {
	case isStringish(goType):
		return fmt.Sprintf("w.WriteString(%%d, %s)", name)
	case isIntish(goType):
		return fmt.Sprintf("w.WriteInt(%%d, int64(%s))", name)
	case isByteSlice(goType):
		return fmt.Sprintf("w.WriteBytes(%%d, %s)", name)
	default:
		return fmt.Sprintf("w.WriteString(%%d, fmt.Sprint(%s))", name)
	}
}

// readMethod returns the Reader method call for a given Go type.
func readMethod(name, goType string) string {
	switch {
	case isStringish(goType):
		return fmt.Sprintf("r.ReadString(%%d, &%s)", name)
	case isIntish(goType):
		return fmt.Sprintf("r.ReadInt(%%d, &%s)", name)
	case isByteSlice(goType):
		return fmt.Sprintf("r.ReadBytes(%%d, &%s)", name)
	default:
		return fmt.Sprintf("r.ReadString(%%d, &%s)", name)
	}
}

// ── Code generation ─────────────────────────────────────────────────────────────

func generateCode(pkgName string, structs []parsedStruct) string {
	var b strings.Builder
	b.WriteString("// Code generated by iso8583gen; DO NOT EDIT.\n\n")
	b.WriteString("package " + pkgName + "\n\n")

	// Only import fmt if any struct has a string MTI field.
	needFmt := false
	for _, ps := range structs {
		for _, f := range ps.Fields {
			if f.IsMTI && isStringish(f.GoType) {
				needFmt = true
			}
		}
	}

	b.WriteString("import (\n")
	b.WriteString("\t\"bytes\"\n")
	if needFmt {
		b.WriteString("\t\"fmt\"\n")
	}
	b.WriteString("\n")
	b.WriteString("\t\"github.com/Pay8583/iso8583\"\n")
	b.WriteString("\t\"github.com/Pay8583/iso8583/spec\"\n")
	b.WriteString(")\n")

	for _, ps := range structs {
		generateMarshal(&b, ps)
		generateUnmarshal(&b, ps)
	}
	return b.String()
}

func generateMarshal(b *strings.Builder, ps parsedStruct) {
	var mti *structField
	sorted := make([]structField, 0, len(ps.Fields))
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
func (m *%s) MarshalISO8583(p *spec.Protocol) ([]byte, error) {
	var buf bytes.Buffer
	w := iso8583.NewWriter(p, &buf)

`, ps.Name)

	// MTI write.
	if mti != nil {
		fmt.Fprintf(b, "\tif err := w.WriteMTI(%s); err != nil { return nil, err }\n",
			mtiWriteExpr("m."+mti.GoName, mti.GoType))
	} else {
		b.WriteString("\tw.WriteMTI(0x0200)\n")
	}

	// Field writes.
	b.WriteString("\n")
	for _, f := range sorted {
		fmt.Fprintf(b, "\tif %s {\n", zeroCheck(f))
		fmt.Fprintf(b, "\t\tif err := %s; err != nil { return nil, err }\n",
			fmt.Sprintf(writeMethod("m."+f.GoName, f.GoType), f.FieldNum))
		b.WriteString("\t}\n")
	}

	b.WriteString(`
	if err := w.Close(); err != nil { return nil, err }
	return buf.Bytes(), nil
}
`)
}

func generateUnmarshal(b *strings.Builder, ps parsedStruct) {
	var mti *structField
	sorted := make([]structField, 0, len(ps.Fields))
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
func (m *%s) UnmarshalISO8583(data []byte, p *spec.Protocol) error {
	r := iso8583.NewReader(p, bytes.NewReader(data))

	mti, err := r.ReadMTI()
	if err != nil { return err }
`, ps.Name)

	// MTI set.
	if mti != nil {
		fmt.Fprintf(b, "\t%s = %s\n", "m."+mti.GoName, mtiAssignExpr(mti.GoType, "mti"))
	} else {
		b.WriteString("\t_ = mti\n")
	}

	// Field reads via present-fields switch.
	b.WriteString(`
	present, err := r.PresentFields()
	if err != nil { return err }
	for _, n := range present {
		switch n {
`)
	for _, f := range sorted {
		fmt.Fprintf(b, "\t\tcase %d:\n", f.FieldNum)
		fmt.Fprintf(b, "\t\t\tif err := %s; err != nil { return err }\n",
			fmt.Sprintf(readMethod("m."+f.GoName, f.GoType), f.FieldNum))
	}
	b.WriteString("\t\t}\n\t}\n\n\treturn nil\n}\n")
}

// mtiWriteExpr returns a Go expression that produces a uint for WriteMTI.
// For string MTI types, it generates inline parsing via fmt.Sscanf.
func mtiWriteExpr(name, goType string) string {
	switch {
	case isUintish(goType):
		return fmt.Sprintf("uint(%s)", name)
	case isIntish(goType):
		return fmt.Sprintf("uint(%s)", name)
	case isStringish(goType):
		return fmt.Sprintf("func() uint { var mti uint; fmt.Sscanf(%s, \"%%04X\", &mti); return mti }()", name)
	default:
		return fmt.Sprintf("uint(%s)", name)
	}
}

// mtiAssignExpr returns a Go expression that assigns a uint MTI to the field.
func mtiAssignExpr(goType, varName string) string {
	switch {
	case isUintish(goType):
		return fmt.Sprintf("%s(%s)", goType, varName)
	case isIntish(goType):
		return fmt.Sprintf("%s(%s)", goType, varName)
	case isStringish(goType):
		return fmt.Sprintf("fmt.Sprintf(\"%%04X\", %s)", varName)
	default:
		return varName
	}
}
