package iso8583

import (
	"fmt"

	"github.com/Pay8583/iso8583/spec"
)

// Message is a generic, map-based representation of a decoded ISO 8583 message.
// It is used by the CLI and for dynamic (non-struct-based) use cases.
// For performance-sensitive code, use struct-based marshaling with codec or the
// code generator.
type Message struct {
	MTI    string
	Fields map[int]string // field index → raw string value
	Spec   *spec.Spec
}

// NewMessage creates a Message for the given spec with a default MTI.
func NewMessage(s *spec.Spec, mti string) *Message {
	return &Message{
		MTI:    mti,
		Fields: make(map[int]string),
		Spec:   s,
	}
}

// Set sets the string value of a field.
func (m *Message) Set(index int, value string) {
	m.Fields[index] = value
}

// Get returns the string value of a field, or "" if not present.
func (m *Message) Get(index int) string {
	return m.Fields[index]
}

// Has reports whether a field is present.
func (m *Message) Has(index int) bool {
	_, ok := m.Fields[index]
	return ok
}

// Delete removes a field from the message.
func (m *Message) Delete(index int) {
	delete(m.Fields, index)
}

// FieldNames returns the spec-derived names for all present fields.
func (m *Message) FieldNames() map[int]string {
	names := make(map[int]string, len(m.Fields))
	for n := range m.Fields {
		if fs := m.Spec.GetField(n); fs != nil {
			names[n] = fs.Name
		} else {
			names[n] = fmt.Sprintf("Field %d", n)
		}
	}
	return names
}

// Pack serializes the message to ISO 8583 wire format bytes.
func (m *Message) Pack() ([]byte, error) {
	return PackMessage(m)
}

// Unpack deserializes ISO 8583 wire format bytes into this message,
// replacing its current MTI and fields.
func (m *Message) Unpack(data []byte) error {
	msg, err := UnpackMessage(data, m.Spec)
	if err != nil {
		return err
	}
	m.MTI = msg.MTI
	m.Fields = msg.Fields
	return nil
}
