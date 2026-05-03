package spec

import "github.com/Pay8583/iso8583/encoding"

var ISO8583_2003 *Spec

func init() {
	// ISO 8583:2003 is structurally similar to 1993 (fields 1-128, secondary bitmap)
	// but has refined field definitions and adds subfield structures in fields 48, 127.
	s := ISO8583_1993.Clone()
	s.Name = "2003"
	s.Version = "2003"
	s.Description = "ISO 8583:2003 — refined standard, fields 1–128, secondary bitmap"

	// Key differences in 2003: updated encoding for some fields, expanded max lengths.
	s.UpdateField(48, FieldSpec{
		Index: 48, Name: "Additional Data — Private", LengthType: LLLVAR,
		ContentType: Alpha, Encoder: encoding.MustGet("ascii"), MaxLen: 9999,
	})
	s.UpdateField(55, FieldSpec{
		Index: 55, Name: "ICC System Related Data", LengthType: LLLLVAR,
		ContentType: Binary, Encoder: encoding.MustGet("binary"), MaxLen: 2048,
	})
	s.UpdateField(90, FieldSpec{
		Index: 90, Name: "Original Data Elements", LengthType: LLLVAR,
		ContentType: Alpha, Encoder: encoding.MustGet("ascii"), MaxLen: 999, Optional: true,
	})

	ISO8583_2003 = s
	MustRegister(s)
}
