package mac

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/des"
	"fmt"

	"github.com/Pay8583/iso8583/padding"
)

// Algo1Signer implements ISO 9797-1 Algorithm 1: single-key CBC-MAC.
// The message is padded (typically ISO 9797 Method 2), then encrypted in CBC
// mode with a zero IV. The final block is the MAC.
type Algo1Signer struct {
	Key    []byte
	Cipher string // "aes" or "des"
	Padder padding.Padder
}

func (s *Algo1Signer) Algorithm() string { return "iso9797-algo1-" + s.Cipher }

func (s *Algo1Signer) Sign(data []byte) ([]byte, error) {
	block, err := makeBlock(s.Cipher, s.Key)
	if err != nil {
		return nil, err
	}
	bs := block.BlockSize()
	padded, err := s.Padder.Pad(data, bs)
	if err != nil {
		return nil, fmt.Errorf("algo1 pad: %w", err)
	}

	iv := make([]byte, bs)
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(padded, padded)
	return padded[len(padded)-bs:], nil
}

func (s *Algo1Signer) Verify(data []byte, signature []byte) error {
	expected, err := s.Sign(data)
	if err != nil {
		return err
	}
	if len(expected) != len(signature) {
		return fmt.Errorf("algo1: signature length mismatch")
	}
	for i := range expected {
		if expected[i] != signature[i] {
			return fmt.Errorf("algo1: signature mismatch")
		}
	}
	return nil
}

// Algo3Signer implements ISO 9797-1 Algorithm 3 (Retail MAC / ANSI X9.19).
// Uses two DES keys: encrypt with key1, decrypt with key2, encrypt with key1.
type Algo3Signer struct {
	Key1, Key2 []byte // 8-byte DES keys
	Padder     padding.Padder
}

func (s *Algo3Signer) Algorithm() string { return "iso9797-algo3-des" }

func (s *Algo3Signer) Sign(data []byte) ([]byte, error) {
	if len(s.Key1) != 8 || len(s.Key2) != 8 {
		return nil, fmt.Errorf("algo3: DES keys must be 8 bytes")
	}
	block1, err := des.NewCipher(s.Key1)
	if err != nil {
		return nil, err
	}
	block2, err := des.NewCipher(s.Key2)
	if err != nil {
		return nil, err
	}

	bs := 8
	padded, err := s.Padder.Pad(data, bs)
	if err != nil {
		return nil, fmt.Errorf("algo3 pad: %w", err)
	}

	// CBC encrypt with key1, zero IV.
	iv := make([]byte, bs)
	enc := cipher.NewCBCEncrypter(block1, iv)
	enc.CryptBlocks(padded, padded)

	// Take last block.
	last := padded[len(padded)-bs:]

	// Decrypt with key2.
	dec := cipher.NewCBCDecrypter(block2, iv)
	dec.CryptBlocks(last, last)

	// Re-encrypt with key1.
	enc2 := cipher.NewCBCEncrypter(block1, iv)
	enc2.CryptBlocks(last, last)

	// Clone to avoid aliasing.
	out := make([]byte, bs)
	copy(out, last)
	return out, nil
}

func (s *Algo3Signer) Verify(data []byte, signature []byte) error {
	expected, err := s.Sign(data)
	if err != nil {
		return err
	}
	if len(expected) != len(signature) {
		return fmt.Errorf("algo3: signature length mismatch")
	}
	for i := range expected {
		if expected[i] != signature[i] {
			return fmt.Errorf("algo3: signature mismatch")
		}
	}
	return nil
}

func makeBlock(cipherName string, key []byte) (cipher.Block, error) {
	switch cipherName {
	case "aes":
		return aes.NewCipher(key)
	case "des":
		return des.NewCipher(key)
	default:
		return nil, fmt.Errorf("unsupported cipher %q", cipherName)
	}
}
