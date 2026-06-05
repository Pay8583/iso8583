package iso8583

import (
	"bytes"
	"fmt"

	"github.com/Pay8583/iso8583/spec"
)

// Message is a generic map-based representation of a decoded ISO 8583
// message. It is used by the CLI and for dynamic (non-struct-based) use
// cases. For performance-sensitive code, use struct-based Marshal/Unmarshal
// or the streaming Writer/Reader directly.
//
// This is the only place in the public API where a map (map[int]string) is
// used, and it is explicitly a secondary API intended for CLI and scripting.
type Message struct {
	MTI      uint
	Fields   map[int]string
	Protocol *spec.Protocol
}

// NewMessage creates a Message for the given protocol with a default MTI.
func NewMessage(p *spec.Protocol, mti uint) *Message {
	return &Message{
		MTI:      mti,
		Fields:   make(map[int]string),
		Protocol: p,
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

// FieldNames returns the protocol-derived names for all present fields.
func (m *Message) FieldNames() map[int]string {
	names := make(map[int]string, len(m.Fields))
	for n := range m.Fields {
		if fs := m.Protocol.GetField(n); fs != nil && fs.Name != "" {
			names[n] = fs.Name
		} else {
			names[n] = fmt.Sprintf("Field %d", n)
		}
	}
	return names
}

// Pack serializes the message to ISO 8583 wire format bytes using the
// Writer internally.
func (m *Message) Pack() ([]byte, error) {
	if m.Protocol == nil {
		return nil, fmt.Errorf("message has no protocol")
	}

	var buf bytes.Buffer
	w := NewWriter(m.Protocol, &buf)

	if err := w.WriteMTI(m.MTI); err != nil {
		return nil, err
	}

	for n, val := range m.Fields {
		if n < 2 {
			continue
		}
		fs := m.Protocol.GetField(n)
		if fs == nil {
			continue
		}
		// Write all fields as strings; the Writer handles encoding and validation.
		if err := w.WriteString(n, val); err != nil {
			return nil, err
		}
	}

	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Unpack deserializes ISO 8583 wire format bytes into this message,
// replacing its current MTI and fields.
func (m *Message) Unpack(data []byte) error {
	if m.Protocol == nil {
		return fmt.Errorf("message has no protocol")
	}

	r := NewReader(m.Protocol, bytes.NewReader(data))

	mti, err := r.ReadMTI()
	if err != nil {
		return err
	}
	m.MTI = mti

	// Get the list of present fields.
	present, err := r.PresentFields()
	if err != nil {
		return err
	}

	// Read each present field as a string.
	m.Fields = make(map[int]string, len(present))
	for _, n := range present {
		var s string
		if err := r.ReadString(n, &s); err != nil {
			return err
		}
		m.Fields[n] = s
	}

	return nil
}
