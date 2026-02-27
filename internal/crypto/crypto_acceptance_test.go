package crypto_test

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/MutterPedro/otserver/internal/crypto"
)

// TestAcceptance_XTEAEncryptDecryptRoundTrip verifies that encrypting and
// decrypting arbitrary data in-place with the same XTEA key produces the
// original plaintext. This ensures binary wire-protocol compatibility with
// existing Tibia/OTClient clients.
func TestAcceptance_XTEAEncryptDecryptRoundTrip(t *testing.T) {
	t.Parallel()

	key := []uint32{0x12345678, 0x9ABCDEF0, 0xDEADBEEF, 0xCAFEBABE}
	cipher := crypto.NewCipher(key)

	// Test with multiple random payloads of various sizes (all multiples of 8).
	sizes := []int{8, 16, 64, 256, 1024}
	for _, size := range sizes {
		plaintext := make([]byte, size)
		if _, err := rand.Read(plaintext); err != nil {
			t.Fatalf("rand.Read(%d): %v", size, err)
		}

		original := make([]byte, size)
		copy(original, plaintext)

		if err := cipher.EncryptInPlace(plaintext); err != nil {
			t.Fatalf("EncryptInPlace(%d bytes): %v", size, err)
		}

		if bytes.Equal(plaintext, original) {
			t.Fatalf("ciphertext identical to plaintext for %d bytes; encryption had no effect", size)
		}

		if err := cipher.DecryptInPlace(plaintext); err != nil {
			t.Fatalf("DecryptInPlace(%d bytes): %v", size, err)
		}

		if !bytes.Equal(plaintext, original) {
			t.Fatalf("round-trip failed for %d bytes:\ngot:  %x\nwant: %x", size, plaintext, original)
		}
	}
}
