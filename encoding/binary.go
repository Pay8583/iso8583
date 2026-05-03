package encoding

import "errors"

// binary is a raw byte passthrough: the value string is treated as its byte
// representation directly (each byte of the string is one wire byte).
type binary struct{}

func (binary) Name() string { return "binary" }

func (binary) Encode(value string) ([]byte, error) {
	if len(value) == 0 {
		return nil, errors.New("binary: empty value")
	}
	return []byte(value), nil
}

func (binary) Decode(data []byte) (string, error) {
	if len(data) == 0 {
		return "", errors.New("binary: empty data")
	}
	return string(data), nil
}

func init() { Register(binary{}) }
