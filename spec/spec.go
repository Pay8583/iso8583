// Package spec defines the field-layout specification system for ISO 8583 messages.
// It provides the Spec and FieldSpec types, a global registry of named specs,
// and built-in definitions for ISO 8583:1987, 1993, and 2003.
package spec

import "github.com/Pay8583/iso8583/encoding"

// LengthType describes how a field's length is indicated on the wire.
type LengthType uint8

const (
	Fixed  LengthType = iota // fixed length, no prefix
	LLVAR                    // 1-byte length prefix (1-99)
	LLLVAR                   // 2-byte length prefix (1-999)
	LLLLVAR                  // 3-byte length prefix (1-9999)
)

// ContentType classifies the data within a field.
type ContentType uint8

const (
	Numeric ContentType = iota
	Alpha
	Binary
)

// FieldSpec describes one ISO 8583 data element (fields 2–128).
// Field 1 is the bitmap itself and is not represented as a FieldSpec.
type FieldSpec struct {
	Index       int              // field number (2–128)
	Name        string           // human-readable name
	LengthType  LengthType       // fixed, LLVAR, LLLVAR, LLLLVAR
	ContentType ContentType      // numeric, alpha, binary
	Encoder     encoding.Encoder // wire encoding (bcd, ascii, ebcdic, binary)
	MaxLen      int              // maximum content length in characters
	MinLen      int              // minimum content length
	FixedLen    int              // non-zero for fixed-length fields
	Optional    bool             // the field may be absent
}

// Spec defines a complete ISO 8583 version or variant.
type Spec struct {
	Name                string // e.g. "1987", "1993", "visa", "mastercard"
	Version             string // base version: "1987", "1993", "2003"
	Fields              map[int]*FieldSpec
	MtiEncoder          encoding.Encoder // usually ASCII
	MaxField            int              // highest defined field (64 or 128)
	HasSecondaryBitmap  bool             // true for 1993+, false for 1987
	Description         string
}

// NewSpec creates an empty spec with the given name and version.
func NewSpec(name, version string) *Spec {
	return &Spec{
		Name:     name,
		Version:  version,
		Fields:   make(map[int]*FieldSpec),
		MaxField: 64,
	}
}

// Clone returns a deep copy of the spec for derivation.
func (s *Spec) Clone() *Spec {
	c := &Spec{
		Name:               s.Name,
		Version:            s.Version,
		Fields:             make(map[int]*FieldSpec, len(s.Fields)),
		MtiEncoder:         s.MtiEncoder,
		MaxField:           s.MaxField,
		HasSecondaryBitmap: s.HasSecondaryBitmap,
		Description:        s.Description,
	}
	for i, f := range s.Fields {
		fc := *f
		c.Fields[i] = &fc
	}
	return c
}

// AddField adds a field definition to the spec.
func (s *Spec) AddField(index int, fs FieldSpec) {
	fs.Index = index
	s.Fields[index] = &fs
	if index > s.MaxField {
		s.MaxField = index
	}
}

// RemoveField deletes a field definition from the spec.
func (s *Spec) RemoveField(index int) {
	delete(s.Fields, index)
}

// UpdateField merges a FieldSpec on top of an existing definition.
func (s *Spec) UpdateField(index int, fs FieldSpec) {
	fs.Index = index
	s.Fields[index] = &fs
	if index > s.MaxField {
		s.MaxField = index
	}
}

// GetField returns the FieldSpec for a given index, or nil if not defined.
func (s *Spec) GetField(index int) *FieldSpec {
	return s.Fields[index]
}
