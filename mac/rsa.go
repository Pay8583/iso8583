package mac

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"fmt"
)

// RSASigner implements RSA-SHA256 signing and verification.
type RSASigner struct {
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
}

func (s *RSASigner) Algorithm() string { return "rsa-sha256" }

func (s *RSASigner) Sign(data []byte) ([]byte, error) {
	if s.PrivateKey == nil {
		return nil, fmt.Errorf("rsa: private key required for signing")
	}
	h := sha256.Sum256(data)
	return rsa.SignPKCS1v15(rand.Reader, s.PrivateKey, crypto.SHA256, h[:])
}

func (s *RSASigner) Verify(data []byte, signature []byte) error {
	pub := s.PublicKey
	if pub == nil && s.PrivateKey != nil {
		pub = &s.PrivateKey.PublicKey
	}
	if pub == nil {
		return fmt.Errorf("rsa: public key required for verification")
	}
	h := sha256.Sum256(data)
	return rsa.VerifyPKCS1v15(pub, crypto.SHA256, h[:], signature)
}
