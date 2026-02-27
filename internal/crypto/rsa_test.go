package crypto_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"

	tibiacrypto "github.com/MutterPedro/otserver/internal/crypto"
)

func generateTestPEM(t *testing.T) []byte {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	der := x509.MarshalPKCS1PrivateKey(privateKey)
	block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: der}

	return pem.EncodeToMemory(block)
}

func TestRSA_LoadPrivateKey(t *testing.T) {
	t.Parallel()

	pemData := generateTestPEM(t)

	d, err := tibiacrypto.LoadPrivateKey(pemData)
	if err != nil {
		t.Fatalf("LoadPrivateKey: unexpected error: %v", err)
	}

	if d == nil {
		t.Fatal("LoadPrivateKey returned nil decryptor")
	}
}

func TestRSA_LoadPrivateKeyInvalidPEM(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		pemData []byte
	}{
		{"empty", []byte{}},
		{"garbage", []byte("not a PEM block")},
		{"wrong block type", pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: []byte("fake")})},
		{"corrupt DER", pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: []byte("corrupt")})},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := tibiacrypto.LoadPrivateKey(tc.pemData)
			if err == nil {
				t.Error("expected error for invalid PEM, got nil")
			}
		})
	}
}

func TestRSA_EncryptDecryptRoundTrip(t *testing.T) {
	t.Parallel()

	pemData := generateTestPEM(t)
	d, err := tibiacrypto.LoadPrivateKey(pemData)
	if err != nil {
		t.Fatalf("LoadPrivateKey: %v", err)
	}

	block, _ := pem.Decode(pemData)
	privateKey, _ := x509.ParsePKCS1PrivateKey(block.Bytes)

	// Tibia login payloads are 128 bytes (RSA-1024 block size).
	// PKCS1v15 overhead is 11 bytes, so max plaintext is 128-11=117 bytes.
	plaintext := make([]byte, 117)
	for i := range plaintext {
		plaintext[i] = byte(i)
	}

	ciphertext, err := rsa.EncryptPKCS1v15(rand.Reader, &privateKey.PublicKey, plaintext)
	if err != nil {
		t.Fatalf("rsa.EncryptPKCS1v15: %v", err)
	}

	decrypted, err := d.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	if len(decrypted) != len(plaintext) {
		t.Fatalf("decrypted length = %d, want %d", len(decrypted), len(plaintext))
	}

	for i := range plaintext {
		if decrypted[i] != plaintext[i] {
			t.Fatalf("decrypted[%d] = 0x%02X, want 0x%02X", i, decrypted[i], plaintext[i])
		}
	}
}

func TestRSA_DecryptInvalidCiphertext(t *testing.T) {
	t.Parallel()

	pemData := generateTestPEM(t)
	d, err := tibiacrypto.LoadPrivateKey(pemData)
	if err != nil {
		t.Fatalf("LoadPrivateKey: %v", err)
	}

	// 128 bytes of garbage (RSA-1024 block size).
	garbage := make([]byte, 128)
	for i := range garbage {
		garbage[i] = 0xFF
	}

	_, err = d.Decrypt(garbage)
	if err == nil {
		t.Error("expected error when decrypting garbage ciphertext, got nil")
	}
}
