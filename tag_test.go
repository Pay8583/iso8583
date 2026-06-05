package iso8583

import (
	"testing"

	"github.com/Pay8583/iso8583/encoding"
	"github.com/Pay8583/iso8583/spec"
)

func TestParseTag_MTI(t *testing.T) {
	pt, err := ParseTag("mti")
	if err != nil {
		t.Fatal(err)
	}
	if !pt.IsMTI {
		t.Error("expected IsMTI")
	}
}

func TestParseTag_FieldNumber(t *testing.T) {
	tests := []struct {
		tag   string
		field int
	}{
		{"2", 2},
		{"2,llvar,bcd,n", 2},
		{"35,llvar,ascii", 35},
		{"128,binary,len=8", 128},
		{"64", 64},
	}
	for _, tt := range tests {
		pt, err := ParseTag(tt.tag)
		if err != nil {
			t.Errorf("ParseTag(%q): %v", tt.tag, err)
			continue
		}
		if pt.FieldNumber != tt.field {
			t.Errorf("ParseTag(%q).FieldNumber = %d, want %d", tt.tag, pt.FieldNumber, tt.field)
		}
	}
}

func TestParseTag_LengthTypes(t *testing.T) {
	tests := []struct {
		tag    string
		lenTyp string
	}{
		{"2,llvar", "llvar"},
		{"2,lllvar", "lllvar"},
		{"2,llllvar", "llllvar"},
		{"2,fixed=6", "fixed"},
	}
	for _, tt := range tests {
		pt, err := ParseTag(tt.tag)
		if err != nil {
			t.Errorf("ParseTag(%q): %v", tt.tag, err)
			continue
		}
		if pt.LengthType != tt.lenTyp {
			t.Errorf("ParseTag(%q).LengthType = %q, want %q", tt.tag, pt.LengthType, tt.lenTyp)
		}
		if tt.lenTyp == "fixed" && pt.FixedLen != 6 {
			t.Errorf("ParseTag(%q).FixedLen = %d, want 6", tt.tag, pt.FixedLen)
		}
	}
}

func TestParseTag_ValueEncodings(t *testing.T) {
	tests := []struct {
		tag  string
		val  string
	}{
		{"2,bcd", "bcd"},
		{"2,ascii", "ascii"},
		{"2,rbcd", "rbcd"},
		{"2,hex", "hex"},
		{"2,text", "text"},
		{"2,raw", "raw"},
		{"2,binary", "binary"},
		{"2,ebcdic", "ebcdic"},
	}
	for _, tt := range tests {
		pt, err := ParseTag(tt.tag)
		if err != nil {
			t.Errorf("ParseTag(%q): %v", tt.tag, err)
			continue
		}
		if pt.ValueName != tt.val {
			t.Errorf("ParseTag(%q).ValueName = %q, want %q", tt.tag, pt.ValueName, tt.val)
		}
	}
}

func TestParseTag_Validators(t *testing.T) {
	tests := []struct {
		tag  string
		valid string
	}{
		{"2,n", "n"},
		{"2,an", "an"},
		{"2,ans", "ans"},
		{"2,b", "b"},
		{"2,z", "z"},
		{"2,ns", "ns"},
		{"2,xn", "xn"},
		// Backward-compatible old names.
		{"2,numeric", "n"},
		{"2,alpha", "an"},
	}
	for _, tt := range tests {
		pt, err := ParseTag(tt.tag)
		if err != nil {
			t.Errorf("ParseTag(%q): %v", tt.tag, err)
			continue
		}
		if pt.ValidatorName != tt.valid {
			t.Errorf("ParseTag(%q).ValidatorName = %q, want %q", tt.tag, pt.ValidatorName, tt.valid)
		}
	}
}

func TestParseTag_Flags(t *testing.T) {
	// Secure.
	pt, err := ParseTag("2,secure")
	if err != nil {
		t.Fatal(err)
	}
	if pt.Secure == nil || !*pt.Secure {
		t.Error("expected Secure=true")
	}

	// Optional.
	pt, err = ParseTag("2,optional")
	if err != nil {
		t.Fatal(err)
	}
	if !pt.Optional {
		t.Error("expected Optional=true")
	}
}

func TestParseTag_Pad(t *testing.T) {
	tests := []struct {
		tag string
		pad byte
	}{
		{"2,pad='0'", '0'},
		{"2,pad=' '", ' '},
		{"2,pad=0x00", 0x00},
		{"2,pad=0xFF", 0xFF},
	}
	for _, tt := range tests {
		pt, err := ParseTag(tt.tag)
		if err != nil {
			t.Errorf("ParseTag(%q): %v", tt.tag, err)
			continue
		}
		if pt.Pad == nil || *pt.Pad != tt.pad {
			t.Errorf("ParseTag(%q).Pad = %v, want %v", tt.tag, pt.Pad, tt.pad)
		}
	}
}

func TestParseTag_Len(t *testing.T) {
	pt, err := ParseTag("2,len=99")
	if err != nil {
		t.Fatal(err)
	}
	if pt.MaxLen != 99 {
		t.Errorf("MaxLen = %d, want 99", pt.MaxLen)
	}
}

func TestParseTag_Name(t *testing.T) {
	pt, err := ParseTag(`2,name="PrimaryAccountNumber"`)
	if err != nil {
		t.Fatal(err)
	}
	if pt.Name != "PrimaryAccountNumber" {
		t.Errorf("Name = %q, want %q", pt.Name, "PrimaryAccountNumber")
	}
}

func TestParseTag_FullTag(t *testing.T) {
	tag := `2,llvar,ascii,ans,max=19,optional,secure,name="PAN"`
	pt, err := ParseTag(tag)
	if err != nil {
		t.Fatal(err)
	}
	if pt.FieldNumber != 2 {
		t.Errorf("FieldNumber = %d", pt.FieldNumber)
	}
	if pt.LengthType != "llvar" {
		t.Errorf("LengthType = %q", pt.LengthType)
	}
	if pt.ValueName != "ascii" {
		t.Errorf("ValueName = %q", pt.ValueName)
	}
	if pt.ValidatorName != "ans" {
		t.Errorf("ValidatorName = %q", pt.ValidatorName)
	}
	if pt.MaxLen != 19 {
		t.Errorf("MaxLen = %d", pt.MaxLen)
	}
	if !pt.Optional {
		t.Error("expected optional")
	}
	if pt.Secure == nil || !*pt.Secure {
		t.Error("expected secure")
	}
	if pt.Name != "PAN" {
		t.Errorf("Name = %q", pt.Name)
	}
}

func TestParseTag_OldStyleTag(t *testing.T) {
	// Backward compatibility: existing tag patterns should still parse.
	tag := "2,llvar,bcd,numeric,max=19,optional,name=\"PAN\""
	pt, err := ParseTag(tag)
	if err != nil {
		t.Fatal(err)
	}
	if pt.LengthType != "llvar" {
		t.Errorf("LengthType = %q, want llvar", pt.LengthType)
	}
	if pt.ValueName != "bcd" {
		t.Errorf("ValueName = %q, want bcd", pt.ValueName)
	}
	if pt.ValidatorName != "n" {
		t.Errorf("ValidatorName = %q, want n (from numeric)", pt.ValidatorName)
	}
	if pt.MaxLen != 19 {
		t.Errorf("MaxLen = %d", pt.MaxLen)
	}
	if !pt.Optional {
		t.Error("expected optional")
	}
}

func TestParseTag_Errors(t *testing.T) {
	_, err := ParseTag("")
	if err == nil {
		t.Error("expected error for empty tag")
	}
	_, err = ParseTag("0")
	if err == nil {
		t.Error("expected error for field 0")
	}
	_, err = ParseTag("200")
	if err == nil {
		t.Error("expected error for field 200")
	}
	_, err = ParseTag("2,unknown_hint")
	if err == nil {
		t.Error("expected error for unknown hint")
	}
}

func TestResolveField_FromProtocol(t *testing.T) {
	// Resolve a tag against a Protocol field — tag should override.
	protoField := &spec.Field{
		Name:   "PAN",
		Len:    spec.LVAR(19, encoding.MustGet("bcd")),
		Valid:  spec.N,
		Value:  spec.ASCII,
		Secure: true,
	}

	pt := &ParsedTag{
		FieldNumber:   2,
		LengthType:    "fixed",
		FixedLen:      16,
		ValueName:     "rbcd",
		ValidatorName: "n",
	}
	falseVal := false
	pt.Secure = &falseVal

	resolved, err := pt.ResolveField(protoField)
	if err != nil {
		t.Fatalf("ResolveField: %v", err)
	}

	// Tag should override.
	if !resolved.Len.IsFixed() || resolved.Len.FixedLen() != 16 {
		t.Error("Length should be overridden to fixed(16)")
	}
	if resolved.Value.String() != "RBCD(16)" {
		t.Errorf("Value = %s, want RBCD(16)", resolved.Value.String())
	}
	if resolved.Valid.String() != "N" {
		t.Errorf("Valid = %s, want N", resolved.Valid.String())
	}
	if resolved.Secure {
		t.Error("Secure should be overridden to false")
	}
	if resolved.Name != "PAN" {
		t.Errorf("Name = %q, want PAN (inherited)", resolved.Name)
	}
}

func TestResolveField_TagOnly(t *testing.T) {
	// No proto field — tag must supply everything.
	pt := &ParsedTag{
		FieldNumber:   127,
		LengthType:    "fixed",
		FixedLen:      8,
		ValueName:     "raw",
		ValidatorName: "b",
	}
	pt.Name = "CustomField"
	secure := true
	pt.Secure = &secure

	resolved, err := pt.ResolveField(nil)
	if err != nil {
		t.Fatalf("ResolveField: %v", err)
	}

	if !resolved.Len.IsFixed() || resolved.Len.FixedLen() != 8 {
		t.Error("Length should be fixed(8)")
	}
	if resolved.Value.String() != "Raw" {
		t.Errorf("Value = %s, want Raw", resolved.Value.String())
	}
	if resolved.Valid.String() != "B" {
		t.Errorf("Valid = %s, want B", resolved.Valid.String())
	}
	if !resolved.Secure {
		t.Error("Secure should be true")
	}
}

func TestResolveField_TagOnly_MissingLength(t *testing.T) {
	pt := &ParsedTag{
		FieldNumber: 2,
		ValueName:   "ascii",
	}
	_, err := pt.ResolveField(nil)
	if err == nil {
		t.Error("expected error: no length specified")
	}
}

func TestResolveField_TagOnly_MissingValue(t *testing.T) {
	pt := &ParsedTag{
		FieldNumber: 2,
		LengthType:  "fixed",
		FixedLen:    6,
	}
	_, err := pt.ResolveField(nil)
	if err == nil {
		t.Error("expected error: no value encoding specified")
	}
}

func TestResolveField_Inherit(t *testing.T) {
	// Tag specifies nothing — everything inherited from proto.
	protoField := &spec.Field{
		Name:   "STAN",
		Len:    spec.Fixed(6, '0'),
		Valid:  spec.N,
		Value:  spec.ASCII,
		Secure: false,
	}

	pt := &ParsedTag{FieldNumber: 11}
	resolved, err := pt.ResolveField(protoField)
	if err != nil {
		t.Fatalf("ResolveField: %v", err)
	}

	if !resolved.Len.IsFixed() || resolved.Len.FixedLen() != 6 {
		t.Error("Length should be inherited fixed(6)")
	}
	if resolved.Value.String() != "ASCII" {
		t.Errorf("Value = %s, want ASCII", resolved.Value.String())
	}
	if resolved.Valid.String() != "N" {
		t.Errorf("Valid = %s, want N", resolved.Valid.String())
	}
	if resolved.Name != "STAN" {
		t.Errorf("Name = %q, want STAN", resolved.Name)
	}
}

func TestResolveField_DefaultValidator(t *testing.T) {
	// Proto field has no validator — should default to B.
	protoField := &spec.Field{
		Name:  "Custom",
		Len:   spec.Fixed(4, '0'),
		Value: spec.ASCII,
	}
	pt := &ParsedTag{FieldNumber: 48}
	resolved, err := pt.ResolveField(protoField)
	if err != nil {
		t.Fatalf("ResolveField: %v", err)
	}
	if resolved.Valid.String() != "B" {
		t.Errorf("Valid = %s, want B (default)", resolved.Valid.String())
	}
}
