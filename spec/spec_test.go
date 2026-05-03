package spec

import (
	"testing"

	"github.com/Pay8583/iso8583/encoding"
)

func TestSpec_AddField(t *testing.T) {
	s := NewSpec("test", "1987")
	s.MaxField = 0 // reset from default 64 for this test
	enc := encoding.MustGet("ascii")
	s.AddField(2, FieldSpec{
		Name: "PAN", LengthType: LLVAR, ContentType: Numeric,
		Encoder: enc, MaxLen: 19,
	})
	fs := s.GetField(2)
	if fs == nil {
		t.Fatal("field 2 not found")
	}
	if fs.Name != "PAN" {
		t.Errorf("Name = %q, want %q", fs.Name, "PAN")
	}
	if s.MaxField != 2 {
		t.Errorf("MaxField = %d, want 2", s.MaxField)
	}
}

func TestSpec_Clone(t *testing.T) {
	s := NewSpec("original", "1987")
	enc := encoding.MustGet("bcd")
	s.AddField(2, FieldSpec{
		Name: "PAN", LengthType: LLVAR, ContentType: Numeric,
		Encoder: enc, MaxLen: 19,
	})
	c := s.Clone()
	if c.Name != s.Name {
		t.Errorf("Name not copied")
	}
	if c.Fields[2].Name != "PAN" {
		t.Errorf("Field not deep-copied")
	}
	// Modifying clone should not affect original
	c.AddField(3, FieldSpec{
		Name: "ProcCode", LengthType: Fixed, ContentType: Numeric,
		Encoder: enc, FixedLen: 6,
	})
	if s.GetField(3) != nil {
		t.Error("original should not have field 3 after clone modified")
	}
}

func TestSpec_UpdateField(t *testing.T) {
	s := NewSpec("test", "1987")
	enc := encoding.MustGet("bcd")
	s.AddField(2, FieldSpec{
		Name: "PAN", LengthType: LLVAR, ContentType: Numeric,
		Encoder: enc, MaxLen: 19,
	})
	s.UpdateField(2, FieldSpec{
		Name: "PAN-Updated", LengthType: LLVAR, ContentType: Numeric,
		Encoder: enc, MaxLen: 22,
	})
	fs := s.GetField(2)
	if fs.Name != "PAN-Updated" {
		t.Errorf("Name = %q after update", fs.Name)
	}
	if fs.MaxLen != 22 {
		t.Errorf("MaxLen = %d after update", fs.MaxLen)
	}
}

func TestBuiltin1987Spec(t *testing.T) {
	s := MustGet("1987")
	if s == nil {
		t.Fatal("1987 spec not found")
	}
	if s.MaxField != 64 {
		t.Errorf("MaxField = %d, want 64", s.MaxField)
	}
	if s.HasSecondaryBitmap {
		t.Error("1987 spec should not have secondary bitmap")
	}
	// Common fields should exist
	for _, f := range []int{2, 3, 4, 7, 11, 12, 13, 35, 37, 39, 41, 42, 49, 64} {
		if s.GetField(f) == nil {
			t.Errorf("field %d missing from 1987 spec", f)
		}
	}
}
