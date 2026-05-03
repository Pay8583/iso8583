package mac

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"
)

// ECDSASigner implements ECDSA signing and verification using SHA-256.
type ECDSASigner struct {
	PrivateKey *ecdsa.PrivateKey
	PublicKey  *ecdsa.PublicKey
}

func (s *ECDSASigner) Algorithm() string { return "ecdsa-sha256" }

func (s *ECDSASigner) Sign(data []byte) ([]byte, error) {
	if s.PrivateKey == nil {
		return nil, fmt.Errorf("ecdsa: private key required for signing")
	}
	h := sha256.Sum256(data)
	r, ss, err := ecdsa.Sign(rand.Reader, s.PrivateKey, h[:])
	if err != nil {
		return nil, err
	}
	// Encode as r || s (each 32 bytes, big-endian).
	out := make([]byte, 64)
	rBytes := r.Bytes()
	sBytes := ss.Bytes()
	copy(out[32-len(rBytes):32], rBytes)
	copy(out[64-len(sBytes):64], sBytes)
	return out, nil
}

func (s *ECDSASigner) Verify(data []byte, signature []byte) error {
	pub := s.PublicKey
	if pub == nil && s.PrivateKey != nil {
		pub = &s.PrivateKey.PublicKey
	}
	if pub == nil {
		return fmt.Errorf("ecdsa: public key required for verification")
	}
	if len(signature) != 64 {
		return fmt.Errorf("ecdsa: signature must be 64 bytes (r||s), got %d", len(signature))
	}
	h := sha256.Sum256(data)
	r := new(big.Int).SetBytes(signature[:32])
	ss := new(big.Int).SetBytes(signature[32:])
	if !ecdsa.Verify(pub, h[:], r, ss) {
		return fmt.Errorf("ecdsa: signature verification failed")
	}
	return nil
}
