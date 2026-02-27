package crypto

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
)

// Decryptor holds an RSA private key for decrypting Tibia login payloads.
type Decryptor struct {
	key *rsa.PrivateKey
}

// LoadPrivateKey parses a PEM-encoded RSA private key and returns a Decryptor.
func LoadPrivateKey(pemData []byte) (*Decryptor, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("rsa: failed to decode PEM block")
	}

	if block.Type != "RSA PRIVATE KEY" {
		return nil, fmt.Errorf("rsa: unexpected PEM block type %q", block.Type)
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("rsa: failed to parse PKCS1 private key: %w", err)
	}

	return &Decryptor{key: privateKey}, nil
}

// Decrypt decrypts RSA-1024 PKCS1v15 ciphertext. A nil random reader is used
// because Tibia's protocol does not require blinding.
func (d *Decryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	plaintext, err := rsa.DecryptPKCS1v15(nil, d.key, ciphertext)
	if err != nil {
		return nil, fmt.Errorf("rsa: decryption failed: %w", err)
	}

	return plaintext, nil
}
