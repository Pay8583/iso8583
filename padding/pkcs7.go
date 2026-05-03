package padding

import "fmt"

// PKCS#7 padding: each padding byte is the number of bytes added.
type pkcs7 struct{}

func (pkcs7) Name() string { return "pkcs7" }

func (pkcs7) Pad(data []byte, blockSize int) ([]byte, error) {
	if blockSize < 1 || blockSize > 255 {
		return nil, fmt.Errorf("pkcs7: blockSize must be 1-255, got %d", blockSize)
	}
	padLen := blockSize - len(data)%blockSize
	out := make([]byte, len(data)+padLen)
	copy(out, data)
	for i := len(data); i < len(out); i++ {
		out[i] = byte(padLen)
	}
	return out, nil
}

func (pkcs7) Unpad(data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 || len(data)%blockSize != 0 {
		return nil, fmt.Errorf("%w: pkcs7: data length %d not multiple of %d", ErrBadPadding, len(data), blockSize)
	}
	padLen := int(data[len(data)-1])
	if padLen == 0 || padLen > blockSize || padLen > len(data) {
		return nil, fmt.Errorf("%w: pkcs7: invalid pad length %d", ErrBadPadding, padLen)
	}
	for i := len(data) - padLen; i < len(data); i++ {
		if data[i] != byte(padLen) {
			return nil, fmt.Errorf("%w: pkcs7: inconsistent padding", ErrBadPadding)
		}
	}
	return data[:len(data)-padLen], nil
}

func init() { Register(pkcs7{}) }
