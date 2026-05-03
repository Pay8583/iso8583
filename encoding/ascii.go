package encoding

import "errors"

type ascii struct{}

func (ascii) Name() string { return "ascii" }

func (ascii) Encode(value string) ([]byte, error) {
	if len(value) == 0 {
		return nil, errors.New("ascii: empty value")
	}
	return []byte(value), nil
}

func (ascii) Decode(data []byte) (string, error) {
	if len(data) == 0 {
		return "", errors.New("ascii: empty data")
	}
	return string(data), nil
}

func init() { Register(ascii{}) }
