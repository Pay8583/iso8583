package padding

import (
	"crypto/rand"
	"fmt"
)

// iso9797m1 is ISO 9797-1 Method 1: append zeros to fill the block.
type iso9797m1 struct{}

func (iso9797m1) Name() string { return "iso9797-1" }

func (iso9797m1) Pad(data []byte, blockSize int) ([]byte, error) {
	if blockSize < 1 || blockSize > 255 {
		return nil, fmt.Errorf("iso9797-1: invalid blockSize %d", blockSize)
	}
	padLen := blockSize - len(data)%blockSize
	if padLen == blockSize {
		return data, nil // already aligned
	}
	out := make([]byte, len(data)+padLen)
	copy(out, data)
	return out, nil
}

func (iso9797m1) Unpad(data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 || len(data)%blockSize != 0 {
		return nil, fmt.Errorf("%w: iso9797-1: data length %d not multiple of %d", ErrBadPadding, len(data), blockSize)
	}
	// Strip trailing zeros.
	i := len(data) - 1
	for i >= 0 && data[i] == 0 {
		i--
	}
	return data[:i+1], nil
}

// iso9797m2 is ISO 9797-1 Method 2: append 0x80 followed by zeros.
type iso9797m2 struct{}

func (iso9797m2) Name() string { return "iso9797-2" }

func (iso9797m2) Pad(data []byte, blockSize int) ([]byte, error) {
	if blockSize < 1 || blockSize > 255 {
		return nil, fmt.Errorf("iso9797-2: invalid blockSize %d", blockSize)
	}
	out := make([]byte, len(data)+1)
	copy(out, data)
	out[len(data)] = 0x80
	remain := blockSize - len(out)%blockSize
	if remain < blockSize {
		out = append(out, make([]byte, remain)...)
	}
	return out, nil
}

func (iso9797m2) Unpad(data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 || len(data)%blockSize != 0 {
		return nil, fmt.Errorf("%w: iso9797-2: data length %d not multiple of %d", ErrBadPadding, len(data), blockSize)
	}
	i := len(data) - 1
	for i >= 0 && data[i] == 0x00 {
		i--
	}
	if i < 0 || data[i] != 0x80 {
		return nil, fmt.Errorf("%w: iso9797-2: missing 0x80 marker", ErrBadPadding)
	}
	return data[:i], nil
}

// iso10126 is ISO 10126: random bytes followed by count byte.
type iso10126 struct{}

func (iso10126) Name() string { return "iso10126" }

func (iso10126) Pad(data []byte, blockSize int) ([]byte, error) {
	if blockSize < 1 || blockSize > 255 {
		return nil, fmt.Errorf("iso10126: invalid blockSize %d", blockSize)
	}
	padLen := blockSize - len(data)%blockSize
	out := make([]byte, len(data)+padLen)
	copy(out, data)
	if padLen > 1 {
		if _, err := rand.Read(out[len(data) : len(data)+padLen-1]); err != nil {
			return nil, err
		}
	}
	out[len(out)-1] = byte(padLen)
	return out, nil
}

func (iso10126) Unpad(data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 || len(data)%blockSize != 0 {
		return nil, fmt.Errorf("%w: iso10126: data length %d not multiple of %d", ErrBadPadding, len(data), blockSize)
	}
	padLen := int(data[len(data)-1])
	if padLen == 0 || padLen > blockSize || padLen > len(data) {
		return nil, fmt.Errorf("%w: iso10126: invalid pad length %d", ErrBadPadding, padLen)
	}
	return data[:len(data)-padLen], nil
}

// zero is simple zero-byte padding (used by some legacy systems).
type zero struct{}

func (zero) Name() string { return "zero" }

func (zero) Pad(data []byte, blockSize int) ([]byte, error) {
	if blockSize < 1 || blockSize > 255 {
		return nil, fmt.Errorf("zero: invalid blockSize %d", blockSize)
	}
	padLen := blockSize - len(data)%blockSize
	if padLen == blockSize {
		return data, nil
	}
	out := make([]byte, len(data)+padLen)
	copy(out, data)
	return out, nil
}

func (zero) Unpad(data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 || len(data)%blockSize != 0 {
		return nil, fmt.Errorf("%w: zero: data length %d not multiple of %d", ErrBadPadding, len(data), blockSize)
	}
	i := len(data) - 1
	for i >= 0 && data[i] == 0 {
		i--
	}
	return data[:i+1], nil
}

func init() {
	Register(iso9797m1{})
	Register(iso9797m2{})
	Register(iso10126{})
	Register(zero{})
}
