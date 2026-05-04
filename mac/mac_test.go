package mac

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"github.com/Pay8583/iso8583/padding"
)

// ── HMAC ────────────────────────────────────────────────────────────────────────

func TestHMACSHA256_SignVerify(t *testing.T) {
	s := &HMACSigner{Key: []byte("secret-key-12345"), Hash: "sha256"}
	data := []byte("test message")

	sig, err := s.Sign(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(sig) != 32 {
		t.Errorf("sig length = %d, want 32", len(sig))
	}
	if err := s.Verify(data, sig); err != nil {
		t.Errorf("Verify: %v", err)
	}
}

func TestHMACSHA256_Tampered(t *testing.T) {
	s := &HMACSigner{Key: []byte("secret-key-12345"), Hash: "sha256"}
	sig, _ := s.Sign([]byte("original"))

	if err := s.Verify([]byte("tampered"), sig); err == nil {
		t.Error("expected failure for tampered data")
	}
}

func TestHMACSHA256_WrongKey(t *testing.T) {
	s1 := &HMACSigner{Key: []byte("key-A-16bytes!!!"), Hash: "sha256"}
	s2 := &HMACSigner{Key: []byte("key-B-16bytes!!!"), Hash: "sha256"}
	sig, _ := s1.Sign([]byte("data"))

	if err := s2.Verify([]byte("data"), sig); err == nil {
		t.Error("expected failure with wrong key")
	}
}

func TestHMACSHA512_SignVerify(t *testing.T) {
	s := &HMACSigner{Key: []byte("key"), Hash: "sha512"}
	sig, err := s.Sign([]byte("test"))
	if err != nil {
		t.Fatal(err)
	}
	if len(sig) != 64 {
		t.Errorf("sig length = %d, want 64", len(sig))
	}
	if err := s.Verify([]byte("test"), sig); err != nil {
		t.Error(err)
	}
}

func TestHMAC_BadHash(t *testing.T) {
	s := &HMACSigner{Key: []byte("key"), Hash: "md5"}
	_, err := s.Sign([]byte("data"))
	if err == nil {
		t.Error("expected error for unsupported hash")
	}
}

// ── RSA ─────────────────────────────────────────────────────────────────────────

func TestRSASHA256_SignVerify(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	s := &RSASigner{PrivateKey: key}

	sig, err := s.Sign([]byte("iso 8583 message"))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Verify([]byte("iso 8583 message"), sig); err != nil {
		t.Errorf("Verify: %v", err)
	}
}

func TestRSASHA256_Tampered(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	s := &RSASigner{PrivateKey: key}
	sig, _ := s.Sign([]byte("original"))

	if err := s.Verify([]byte("different"), sig); err == nil {
		t.Error("expected failure for tampered data")
	}
}

func TestRSASHA256_NoPrivateKey(t *testing.T) {
	s := &RSASigner{} // no keys set
	_, err := s.Sign([]byte("data"))
	if err == nil {
		t.Error("expected error when signing without private key")
	}
}

func TestRSASHA256_NoPublicKey(t *testing.T) {
	s := &RSASigner{} // no keys set
	err := s.Verify([]byte("data"), []byte("fake-signature"))
	if err == nil {
		t.Error("expected error when verifying without public key")
	}
}

func TestRSASHA256_PublicKeyOnly(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	s := &RSASigner{PublicKey: &key.PublicKey}
	sig, _ := (&RSASigner{PrivateKey: key}).Sign([]byte("data"))

	if err := s.Verify([]byte("data"), sig); err != nil {
		t.Errorf("Verify with explicit public key: %v", err)
	}
}

// ── ECDSA ───────────────────────────────────────────────────────────────────────

func TestECDSA_SignVerify(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	s := &ECDSASigner{PrivateKey: key}

	sig, err := s.Sign([]byte("iso 8583 data"))
	if err != nil {
		t.Fatal(err)
	}
	if len(sig) != 64 {
		t.Errorf("sig length = %d, want 64", len(sig))
	}
	if err := s.Verify([]byte("iso 8583 data"), sig); err != nil {
		t.Errorf("Verify: %v", err)
	}
}

func TestECDSA_Tampered(t *testing.T) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	s := &ECDSASigner{PrivateKey: key}
	sig, _ := s.Sign([]byte("original"))

	if err := s.Verify([]byte("tampered"), sig); err == nil {
		t.Error("expected failure for tampered data")
	}
}

func TestECDSA_BadSigLength(t *testing.T) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	s := &ECDSASigner{PrivateKey: key}

	err := s.Verify([]byte("data"), []byte("too-short"))
	if err == nil {
		t.Error("expected error for wrong signature length")
	}
}

func TestECDSA_NoPrivateKey(t *testing.T) {
	s := &ECDSASigner{}
	_, err := s.Sign([]byte("data"))
	if err == nil {
		t.Error("expected error when signing without private key")
	}
}

func TestECDSA_NoPublicKey(t *testing.T) {
	s := &ECDSASigner{}
	err := s.Verify([]byte("data"), make([]byte, 64))
	if err == nil {
		t.Error("expected error when verifying without public key")
	}
}

// ── ISO 9797-1 Algorithm 1 (CBC-MAC) ───────────────────────────────────────────

func TestAlgo1_AES_SignVerify(t *testing.T) {
	key := make([]byte, 16) // AES-128
	s := &Algo1Signer{Key: key, Cipher: "aes", Padder: padding.Get("iso9797-2")}

	sig, err := s.Sign([]byte("MAC test message"))
	if err != nil {
		t.Fatal(err)
	}
	if len(sig) != 16 {
		t.Errorf("AES MAC length = %d, want 16", len(sig))
	}
	if err := s.Verify([]byte("MAC test message"), sig); err != nil {
		t.Errorf("Verify: %v", err)
	}
}

func TestAlgo1_AES_Tampered(t *testing.T) {
	s := &Algo1Signer{Key: make([]byte, 16), Cipher: "aes", Padder: padding.Get("iso9797-2")}
	sig, _ := s.Sign([]byte("original"))

	if err := s.Verify([]byte("tampered"), sig); err == nil {
		t.Error("expected failure for tampered data")
	}
}

func TestAlgo1_AES_EmptyData(t *testing.T) {
	s := &Algo1Signer{Key: make([]byte, 16), Cipher: "aes", Padder: padding.Get("iso9797-2")}
	sig, err := s.Sign([]byte{})
	if err != nil {
		t.Fatal(err)
	}
	// Empty data padded to one block should produce a valid MAC.
	if len(sig) != 16 {
		t.Errorf("AES MAC length = %d, want 16", len(sig))
	}
}

func TestAlgo1_DES_SignVerify(t *testing.T) {
	key := []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF}
	s := &Algo1Signer{Key: key, Cipher: "des", Padder: padding.Get("iso9797-2")}

	sig, err := s.Sign([]byte("DES MAC test"))
	if err != nil {
		t.Fatal(err)
	}
	if len(sig) != 8 {
		t.Errorf("DES MAC length = %d, want 8", len(sig))
	}
	if err := s.Verify([]byte("DES MAC test"), sig); err != nil {
		t.Error(err)
	}
}

func TestAlgo1_BadCipher(t *testing.T) {
	s := &Algo1Signer{Key: []byte("key"), Cipher: "rc4", Padder: padding.Get("pkcs7")}
	_, err := s.Sign([]byte("data"))
	if err == nil {
		t.Error("expected error for unsupported cipher")
	}
}

func TestAlgo1_DifferentPadders(t *testing.T) {
	data := []byte("test message for padding comparison")
	key := make([]byte, 16)

	// iso10126 uses random padding — MACs differ on each call, so skip it.
	for _, padName := range []string{"pkcs7", "iso9797-1", "iso9797-2", "zero"} {
		t.Run(padName, func(t *testing.T) {
			s := &Algo1Signer{Key: key, Cipher: "aes", Padder: padding.Get(padName)}
			sig, err := s.Sign(data)
			if err != nil {
				t.Fatalf("%s: Sign: %v", padName, err)
			}
			if len(sig) != 16 {
				t.Errorf("%s: sig len = %d", padName, len(sig))
			}
			if err := s.Verify(data, sig); err != nil {
				t.Errorf("%s: Verify: %v", padName, err)
			}
		})
	}
}

// ── ISO 9797-1 Algorithm 3 (Retail MAC) ────────────────────────────────────────

func TestAlgo3_SignVerify(t *testing.T) {
	key1 := []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF}
	key2 := []byte{0xFE, 0xDC, 0xBA, 0x98, 0x76, 0x54, 0x32, 0x10}
	s := &Algo3Signer{Key1: key1, Key2: key2, Padder: padding.Get("iso9797-2")}

	sig, err := s.Sign([]byte("retail mac test message"))
	if err != nil {
		t.Fatal(err)
	}
	if len(sig) != 8 {
		t.Errorf("retail MAC length = %d, want 8", len(sig))
	}
	if err := s.Verify([]byte("retail mac test message"), sig); err != nil {
		t.Errorf("Verify: %v", err)
	}
}

func TestAlgo3_Tampered(t *testing.T) {
	s := &Algo3Signer{
		Key1:   []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF},
		Key2:   []byte{0xFE, 0xDC, 0xBA, 0x98, 0x76, 0x54, 0x32, 0x10},
		Padder: padding.Get("iso9797-2"),
	}
	sig, _ := s.Sign([]byte("original"))

	if err := s.Verify([]byte("tampered"), sig); err == nil {
		t.Error("expected failure for tampered data")
	}
}

func TestAlgo3_BadKeySizes(t *testing.T) {
	s := &Algo3Signer{
		Key1:   []byte("short"),
		Key2:   []byte("alsoshort"),
		Padder: padding.Get("iso9797-2"),
	}
	_, err := s.Sign([]byte("data"))
	if err == nil {
		t.Error("expected error for wrong key sizes")
	}
}

// ── Algorithm strings ──────────────────────────────────────────────────────────

func TestAlgorithmNames(t *testing.T) {
	tests := []struct {
		algo func() string
		want string
	}{
		{(&HMACSigner{Hash: "sha256"}).Algorithm, "hmac-sha256"},
		{(&HMACSigner{Hash: "sha512"}).Algorithm, "hmac-sha512"},
		{(&RSASigner{}).Algorithm, "rsa-sha256"},
		{(&ECDSASigner{}).Algorithm, "ecdsa-sha256"},
		{(&Algo1Signer{Cipher: "aes"}).Algorithm, "iso9797-algo1-aes"},
		{(&Algo1Signer{Cipher: "des"}).Algorithm, "iso9797-algo1-des"},
		{(&Algo3Signer{}).Algorithm, "iso9797-algo3-des"},
	}
	for _, tt := range tests {
		if got := tt.algo(); got != tt.want {
			t.Errorf("Algorithm() = %q, want %q", got, tt.want)
		}
	}
}

// ── Benchmarks ──────────────────────────────────────────────────────────────────

func BenchmarkHMAC_Sign_1KB(b *testing.B) {
	s := &HMACSigner{Key: []byte("bench-key-16byte"), Hash: "sha256"}
	data := make([]byte, 1024)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		s.Sign(data)
	}
}

func BenchmarkAlgo1_AES_Sign_1KB(b *testing.B) {
	s := &Algo1Signer{Key: make([]byte, 16), Cipher: "aes", Padder: padding.Get("iso9797-2")}
	data := make([]byte, 1024)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		s.Sign(data)
	}
}

func BenchmarkAlgo3_Sign_1KB(b *testing.B) {
	s := &Algo3Signer{
		Key1:   []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF},
		Key2:   []byte{0xFE, 0xDC, 0xBA, 0x98, 0x76, 0x54, 0x32, 0x10},
		Padder: padding.Get("iso9797-2"),
	}
	data := make([]byte, 1024)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		s.Sign(data)
	}
}

func BenchmarkECDSA_Sign_P256(b *testing.B) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	s := &ECDSASigner{PrivateKey: key}
	data := []byte("ECDSA benchmark message")
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		s.Sign(data)
	}
}

func BenchmarkRSA_Sign_2K(b *testing.B) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	s := &RSASigner{PrivateKey: key}
	data := []byte("RSA benchmark")
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		s.Sign(data)
	}
}

func BenchmarkHMAC_Verify_1KB(b *testing.B) {
	s := &HMACSigner{Key: []byte("bench-key-16byte"), Hash: "sha256"}
	data := make([]byte, 1024)
	sig, _ := s.Sign(data)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		s.Verify(data, sig)
	}
}

func BenchmarkAlgo1_AES_Verify_1KB(b *testing.B) {
	s := &Algo1Signer{Key: make([]byte, 16), Cipher: "aes", Padder: padding.Get("iso9797-2")}
	data := make([]byte, 1024)
	sig, _ := s.Sign(data)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		s.Verify(data, sig)
	}
}
