package iso8583

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"github.com/Pay8583/iso8583/mac"
	"github.com/Pay8583/iso8583/padding"
	"github.com/Pay8583/iso8583/spec"
)

func TestHMACSHA256_SignVerify(t *testing.T) {
	signer := &mac.HMACSigner{Key: []byte("secret-key-12345"), Hash: "sha256"}
	data := []byte("test message")

	sig, err := signer.Sign(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(sig) != 32 {
		t.Errorf("HMAC-SHA256 sig length = %d, want 32", len(sig))
	}

	if err := signer.Verify(data, sig); err != nil {
		t.Errorf("Verify: %v", err)
	}

	// Verification with wrong key should fail.
	signer2 := &mac.HMACSigner{Key: []byte("different-key-"), Hash: "sha256"}
	if err := signer2.Verify(data, sig); err == nil {
		t.Error("expected verification failure with wrong key")
	}
}

func TestHMACSHA512_SignVerify(t *testing.T) {
	signer := &mac.HMACSigner{Key: []byte("key"), Hash: "sha512"}
	data := []byte("test")
	sig, err := signer.Sign(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(sig) != 64 {
		t.Errorf("HMAC-SHA512 sig length = %d, want 64", len(sig))
	}
	if err := signer.Verify(data, sig); err != nil {
		t.Error(err)
	}
}

func TestRSASHA256_SignVerify(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	signer := &mac.RSASigner{PrivateKey: key}
	data := []byte("iso 8583 message to sign")

	sig, err := signer.Sign(data)
	if err != nil {
		t.Fatal(err)
	}
	if err := signer.Verify(data, sig); err != nil {
		t.Errorf("Verify: %v", err)
	}

	// Tampered data should fail.
	if err := signer.Verify([]byte("tampered data"), sig); err == nil {
		t.Error("expected verification failure for tampered data")
	}
}

func TestECDSA_SignVerify(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer := &mac.ECDSASigner{PrivateKey: key}
	data := []byte("iso 8583 data")

	sig, err := signer.Sign(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(sig) != 64 {
		t.Errorf("ECDSA sig length = %d, want 64", len(sig))
	}
	if err := signer.Verify(data, sig); err != nil {
		t.Errorf("Verify: %v", err)
	}
}

func TestISO9797Algo1_SignVerify(t *testing.T) {
	// AES-CBC-MAC with ISO 9797-2 padding.
	key := make([]byte, 16) // AES-128
	signer := &mac.Algo1Signer{
		Key:    key,
		Cipher: "aes",
		Padder: padding.Get("iso9797-2"),
	}
	data := []byte("ISO 8583 message for MAC calculation")
	sig, err := signer.Sign(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(sig) != 16 {
		t.Errorf("AES MAC length = %d, want 16", len(sig))
	}
	if err := signer.Verify(data, sig); err != nil {
		t.Errorf("Verify: %v", err)
	}
}

func TestISO9797Algo3_SignVerify(t *testing.T) {
	// Retail MAC with two DES keys.
	key1 := []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF}
	key2 := []byte{0xFE, 0xDC, 0xBA, 0x98, 0x76, 0x54, 0x32, 0x10}
	signer := &mac.Algo3Signer{
		Key1:   key1,
		Key2:   key2,
		Padder: padding.Get("iso9797-2"),
	}
	data := []byte("retail mac test")
	sig, err := signer.Sign(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(sig) != 8 {
		t.Errorf("Retail MAC length = %d, want 8", len(sig))
	}
	if err := signer.Verify(data, sig); err != nil {
		t.Errorf("Verify: %v", err)
	}
}

func TestSignMessage(t *testing.T) {
	signer := &mac.HMACSigner{Key: []byte("test-key"), Hash: "sha256"}
	msg := []byte("01206020000000000000") // MTI + bitmap + minimal data

	sig, err := SignMessage(msg, signer)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyMessage(msg, signer, sig); err != nil {
		t.Errorf("VerifyMessage: %v", err)
	}
}

func TestExtractPayload(t *testing.T) {
	data := []byte("02006020000000000000extra")
	// Default: exclude MTI (first 4 bytes).
	payload := extractPayload(data, nil)
	if len(payload) != len(data)-4 {
		t.Errorf("default payload len = %d, want %d", len(payload), len(data)-4)
	}
	// Include MTI + bitmap.
	payload = extractPayload(data, &spec.SignConfig{IncludeMTI: true, IncludeBitmap: true})
	if len(payload) != len(data) {
		t.Errorf("include MTI+bitmap: payload len = %d, want %d", len(payload), len(data))
	}
}

// ── Benchmarks ──────────────────────────────────────────────────────────────────

func BenchmarkHMACSHA256_Sign_1KB(b *testing.B) {
	signer := &mac.HMACSigner{Key: []byte("bench-key-16byte"), Hash: "sha256"}
	data := make([]byte, 1024)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		signer.Sign(data)
	}
}

func BenchmarkAlgo1_AES_Sign_1KB(b *testing.B) {
	signer := &mac.Algo1Signer{
		Key:    make([]byte, 16),
		Cipher: "aes",
		Padder: padding.Get("iso9797-2"),
	}
	data := make([]byte, 1024)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		signer.Sign(data)
	}
}
