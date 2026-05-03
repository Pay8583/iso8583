// Package mac provides cryptographic signing and MAC (Message Authentication Code)
// implementations compatible with ISO 8583 message authentication requirements.
package mac

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
)

// HMACSigner signs and verifies using HMAC with the given hash function.
type HMACSigner struct {
	Key  []byte
	Hash string // "sha256" or "sha512"
}

func (s *HMACSigner) Algorithm() string { return "hmac-" + s.Hash }

func (s *HMACSigner) Sign(data []byte) ([]byte, error) {
	switch s.Hash {
	case "sha256":
		h := hmac.New(sha256.New, s.Key)
		h.Write(data)
		return h.Sum(nil), nil
	case "sha512":
		h := hmac.New(sha512.New, s.Key)
		h.Write(data)
		return h.Sum(nil), nil
	default:
		return nil, fmt.Errorf("hmac: unsupported hash %q", s.Hash)
	}
}

func (s *HMACSigner) Verify(data []byte, signature []byte) error {
	expected, err := s.Sign(data)
	if err != nil {
		return err
	}
	if !hmac.Equal(expected, signature) {
		return fmt.Errorf("hmac: signature mismatch")
	}
	return nil
}
