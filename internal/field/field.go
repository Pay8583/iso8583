// Package field provides low-level read/write primitives for ISO 8583 data elements.
// These handle length-prefix reading/writing and fixed-length field slicing,
// but not encoding/decoding (that is the Encoder's responsibility).
package field

import (
	"encoding/binary"
	"errors"
	"fmt"
)

var (
	ErrOverflow  = errors.New("field data exceeds available bytes")
	ErrTooLong   = errors.New("field value exceeds maximum length")
	ErrTooShort  = errors.New("field value shorter than minimum length")
)

// ReadFixed reads exactly length bytes from data.
func ReadFixed(data []byte, length int) ([]byte, error) {
	if len(data) < length {
		return nil, fmt.Errorf("fixed field: need %d bytes, have %d: %w", length, len(data), ErrOverflow)
	}
	return data[:length], nil
}

// ReadLLVAR reads a 1-byte length prefix (binary uint8) followed by that many bytes.
// Returns the raw field bytes and total bytes consumed.
func ReadLLVAR(data []byte) (raw []byte, consumed int, err error) {
	if len(data) < 1 {
		return nil, 0, fmt.Errorf("llvar: need at least 1 byte for length: %w", ErrOverflow)
	}
	n := int(data[0])
	consumed = 1 + n
	if len(data) < consumed {
		return nil, 0, fmt.Errorf("llvar: length=%d but only %d bytes remain: %w", n, len(data)-1, ErrOverflow)
	}
	return data[1:consumed], consumed, nil
}

// ReadLLLVAR reads a 2-byte length prefix (big-endian uint16) followed by that many bytes.
func ReadLLLVAR(data []byte) (raw []byte, consumed int, err error) {
	if len(data) < 2 {
		return nil, 0, fmt.Errorf("lllvar: need at least 2 bytes for length: %w", ErrOverflow)
	}
	n := int(binary.BigEndian.Uint16(data[:2]))
	consumed = 2 + n
	if len(data) < consumed {
		return nil, 0, fmt.Errorf("lllvar: length=%d but only %d bytes remain: %w", n, len(data)-2, ErrOverflow)
	}
	return data[2:consumed], consumed, nil
}

// ReadLLLLVAR reads a 3-byte length prefix (big-endian, stored in uint32) followed by that many bytes.
func ReadLLLLVAR(data []byte) (raw []byte, consumed int, err error) {
	if len(data) < 3 {
		return nil, 0, fmt.Errorf("llllvar: need at least 3 bytes for length: %w", ErrOverflow)
	}
	n := int(uint32(data[0])<<16 | uint32(data[1])<<8 | uint32(data[2]))
	consumed = 3 + n
	if len(data) < consumed {
		return nil, 0, fmt.Errorf("llllvar: length=%d but only %d bytes remain: %w", n, len(data)-3, ErrOverflow)
	}
	return data[3:consumed], consumed, nil
}

// ── Writers ──────────────────────────────────────────────────────────────────────

// WriteFixed appends exactly length bytes to buf, padding with zeros if raw is shorter,
// truncating if longer.
func WriteFixed(buf, raw []byte, length int) []byte {
	if len(raw) >= length {
		return append(buf, raw[:length]...)
	}
	padded := make([]byte, length)
	copy(padded, raw)
	return append(buf, padded...)
}

// WriteLLVAR writes a 1-byte length prefix (binary uint8) followed by the raw bytes.
func WriteLLVAR(buf, raw []byte) []byte {
	buf = append(buf, byte(len(raw)))
	return append(buf, raw...)
}

// WriteLLLVAR writes a 2-byte length prefix (big-endian uint16) followed by the raw bytes.
func WriteLLLVAR(buf, raw []byte) []byte {
	buf = binary.BigEndian.AppendUint16(buf, uint16(len(raw)))
	return append(buf, raw...)
}

// WriteLLLLVAR writes a 3-byte length prefix (big-endian, 3 bytes) followed by the raw bytes.
func WriteLLLLVAR(buf, raw []byte) []byte {
	n := len(raw)
	buf = append(buf, byte(n>>16), byte(n>>8), byte(n))
	return append(buf, raw...)
}
