// Package padding provides standard block-padding schemes used with MAC algorithms
// in ISO 8583 message authentication.
package padding

import "errors"

// Padder applies and removes block padding.
type Padder interface {
	Pad(data []byte, blockSize int) ([]byte, error)
	Unpad(data []byte, blockSize int) ([]byte, error)
	Name() string
}

var (
	registry    = map[string]Padder{}
	ErrBadPadding = errors.New("padding: invalid padding")
)

// Register adds a padder to the registry. Panics on duplicate name.
func Register(p Padder) {
	if _, ok := registry[p.Name()]; ok {
		panic("padding: duplicate padder name: " + p.Name())
	}
	registry[p.Name()] = p
}

// Get returns the named padder, or nil.
func Get(name string) Padder {
	return registry[name]
}
