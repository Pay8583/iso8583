package encoding

import (
	"errors"
	"fmt"
	"strings"
)

// bcd is a packed-BCD encoder: each byte holds two decimal digits (high nibble first).
// Odd-length values get a leading zero nibble in the first byte.
type bcd struct{}

func (bcd) Name() string { return "bcd" }

func (bcd) Encode(value string) ([]byte, error) {
	n := len(value)
	if n == 0 {
		return nil, errors.New("bcd: empty value")
	}
	for i := range n {
		if value[i] < '0' || value[i] > '9' {
			return nil, fmt.Errorf("bcd: invalid digit %c at position %d", value[i], i)
		}
	}
	outLen := (n + 1) / 2
	out := make([]byte, outLen)
	j := 0
	if n%2 != 0 {
		out[0] = value[0] - '0'
		j = 1
		value = value[1:]
	}
	for i := 0; i < len(value); i += 2 {
		out[j] = (value[i]-'0')<<4 | (value[i+1] - '0')
		j++
	}
	return out, nil
}

func (bcd) Decode(data []byte) (string, error) {
	if len(data) == 0 {
		return "", errors.New("bcd: empty data")
	}
	var sb strings.Builder
	sb.Grow(len(data) * 2)
	for _, b := range data {
		sb.WriteByte('0' + (b >> 4))
		sb.WriteByte('0' + (b & 0x0F))
	}
	return sb.String(), nil
}

func init() { Register(bcd{}) }
