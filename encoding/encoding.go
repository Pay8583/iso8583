// Package encoding defines the Encoder interface and built-in wire-format encodings
// used by ISO 8583 data elements.
package encoding

// Encoder converts between a canonical string value and the bytes on the wire for one field.
type Encoder interface {
	Encode(value string) ([]byte, error)
	Decode(data []byte) (string, error)
	Name() string
}

var registry = map[string]Encoder{}

// Register adds an encoder to the global registry. It panics if name is already registered.
func Register(e Encoder) {
	if _, ok := registry[e.Name()]; ok {
		panic("encoding: duplicate encoder name: " + e.Name())
	}
	registry[e.Name()] = e
}

// Get returns the named encoder, or nil if not registered.
func Get(name string) Encoder {
	return registry[name]
}

// MustGet returns the named encoder, panicking if not registered.
func MustGet(name string) Encoder {
	e := Get(name)
	if e == nil {
		panic("encoding: unknown encoder: " + name)
	}
	return e
}
