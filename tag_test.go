package iso8583

import (
	"testing"

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
		{"2,llvar,bcd,numeric", 2},
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

func TestParseTag_Hints(t *testing.T) {
	pt, err := ParseTag("2,llvar,bcd,numeric,max=19,optional,name=\"PAN\"")
	if err != nil {
		t.Fatal(err)
	}
	if pt.LengthType != spec.LLVAR {
		t.Errorf("LengthType = %v", pt.LengthType)
	}
	if pt.EncoderName != "bcd" {
		t.Errorf("EncoderName = %q", pt.EncoderName)
	}
	if pt.ContentType != spec.Numeric {
		t.Errorf("ContentType = %v", pt.ContentType)
	}
	if pt.MaxLen != 19 {
		t.Errorf("MaxLen = %d", pt.MaxLen)
	}
	if !pt.Optional {
		t.Error("expected optional")
	}
	if pt.Name != "PAN" {
		t.Errorf("Name = %q", pt.Name)
	}
}

func TestParseTag_Errors(t *testing.T) {
	_, err := ParseTag("")
	if err == nil {
		t.Error("expected error for empty tag")
	}
	_, err = ParseTag("0") // field 0 invalid
	if err == nil {
		t.Error("expected error for field 0")
	}
	_, err = ParseTag("200") // field 200 out of range
	if err == nil {
		t.Error("expected error for field 200")
	}
	_, err = ParseTag("2,unknown_hint")
	if err == nil {
		t.Error("expected error for unknown hint")
	}
}

func TestParsedTag_ResolveFieldSpec(t *testing.T) {
	pt := &ParsedTag{
		FieldNumber: 2,
		LengthType:  spec.LLLVAR,
		EncoderName: "ascii",
		MaxLen:      99,
		Optional:    true,
	}
	sf := &spec.FieldSpec{
		Index:      2,
		Name:       "PAN",
		LengthType: spec.LLVAR,
		MaxLen:     19,
	}
	resolved := pt.ResolveFieldSpec(sf)
	if resolved.LengthType != spec.LLLVAR {
		t.Errorf("LengthType should be overridden: %v", resolved.LengthType)
	}
	if resolved.MaxLen != 99 {
		t.Errorf("MaxLen should be overridden: %d", resolved.MaxLen)
	}
	if !resolved.Optional {
		t.Error("optional should be set from tag")
	}
	if resolved.Name != "PAN" {
		t.Errorf("Name should come from spec when tag doesn't set it: %q", resolved.Name)
	}
}
