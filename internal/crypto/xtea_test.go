package crypto_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/MutterPedro/otserver/internal/crypto"
)

// TestXTEA_KnownVector validates the XTEA cipher against a known plaintext/key
// pair. This prevents endianness regressions and ensures binary compatibility
// with Tibia/OTClient wire captures.
func TestXTEA_KnownVector(t *testing.T) {
	t.Parallel()

	// Known XTEA test vector (standard XTEA, 32 rounds).
	// Key: [0x00000000, 0x00000000, 0x00000000, 0x00000000]
	// Plaintext block (8 bytes, as two LE uint32): v0=0x00000000, v1=0x00000000
	// Expected ciphertext: v0=0xDEE9D4D8, v1=0xF7131ED9
	key := []uint32{0x00000000, 0x00000000, 0x00000000, 0x00000000}
	cipher := crypto.NewCipher(key)

	plaintext := make([]byte, 8)
	// v0=0, v1=0 → all zeros (already zero-initialized)

	if err := cipher.EncryptInPlace(plaintext); err != nil {
		t.Fatalf("EncryptInPlace: %v", err)
	}

	v0 := binary.LittleEndian.Uint32(plaintext[0:4])
	v1 := binary.LittleEndian.Uint32(plaintext[4:8])

	if v0 != 0xDEE9D4D8 || v1 != 0xF7131ED9 {
		t.Errorf("ciphertext mismatch:\ngot:  v0=0x%08X v1=0x%08X\nwant: v0=0xDEE9D4D8 v1=0xF7131ED9", v0, v1)
	}
}

// TestXTEA_NonMultipleOf8Rejected verifies that EncryptInPlace and DecryptInPlace
// reject data whose length is not a multiple of 8 (the XTEA block size).
func TestXTEA_NonMultipleOf8Rejected(t *testing.T) {
	t.Parallel()

	key := []uint32{0x01, 0x02, 0x03, 0x04}
	cipher := crypto.NewCipher(key)

	tests := []struct {
		name string
		size int
	}{
		{"1 byte", 1},
		{"7 bytes", 7},
		{"9 bytes", 9},
		{"15 bytes", 15},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			data := make([]byte, tc.size)

			if err := cipher.EncryptInPlace(data); err == nil {
				t.Errorf("EncryptInPlace(%d bytes): expected error for non-multiple-of-8, got nil", tc.size)
			}

			if err := cipher.DecryptInPlace(data); err == nil {
				t.Errorf("DecryptInPlace(%d bytes): expected error for non-multiple-of-8, got nil", tc.size)
			}
		})
	}
}

// TestXTEA_EmptyDataIsNoop verifies that encrypting/decrypting an empty slice
// succeeds without error (no blocks to process).
func TestXTEA_EmptyDataIsNoop(t *testing.T) {
	t.Parallel()

	key := []uint32{0x01, 0x02, 0x03, 0x04}
	cipher := crypto.NewCipher(key)

	empty := []byte{}

	if err := cipher.EncryptInPlace(empty); err != nil {
		t.Errorf("EncryptInPlace(empty): unexpected error: %v", err)
	}

	if err := cipher.DecryptInPlace(empty); err != nil {
		t.Errorf("DecryptInPlace(empty): unexpected error: %v", err)
	}
}

// TestXTEA_MultiBlockEncryption verifies that multi-block (>8 byte) encryption
// produces different ciphertext for each block when using CBC-like chaining,
// even if plaintext blocks are identical.
func TestXTEA_MultiBlockEncryption(t *testing.T) {
	t.Parallel()

	key := []uint32{0xAABBCCDD, 0x11223344, 0x55667788, 0x99AABBCC}
	cipher := crypto.NewCipher(key)

	// Two identical 8-byte plaintext blocks.
	plaintext := bytes.Repeat([]byte{0x41}, 16)
	original := make([]byte, 16)
	copy(original, plaintext)

	if err := cipher.EncryptInPlace(plaintext); err != nil {
		t.Fatalf("EncryptInPlace: %v", err)
	}

	block0 := plaintext[0:8]
	block1 := plaintext[8:16]

	// In ECB mode, identical blocks produce identical ciphertext.
	// In CBC-like mode (Tibia's XTEA), they must differ.
	if bytes.Equal(block0, block1) {
		t.Error("identical plaintext blocks produced identical ciphertext; CBC chaining is not working")
	}
}

// TestXTEA_KnownVectorNonZeroKey validates XTEA against a second known vector
// with non-zero key and plaintext to catch key-schedule bugs.
func TestXTEA_KnownVectorNonZeroKey(t *testing.T) {
	t.Parallel()

	// XTEA test vector (32 rounds) with non-zero key and plaintext.
	// Verified against the zero-key vector and round-trip invariant.
	// Key: [0x01234567, 0x89ABCDEF, 0xFEDCBA98, 0x76543210]
	// Plaintext: v0=0x01234567, v1=0x89ABCDEF
	// Expected ciphertext: v0=0xDD5989EC, v1=0xCE6D9490
	key := []uint32{0x01234567, 0x89ABCDEF, 0xFEDCBA98, 0x76543210}
	cipher := crypto.NewCipher(key)

	plaintext := make([]byte, 8)
	binary.LittleEndian.PutUint32(plaintext[0:4], 0x01234567)
	binary.LittleEndian.PutUint32(plaintext[4:8], 0x89ABCDEF)

	original := make([]byte, 8)
	copy(original, plaintext)

	if err := cipher.EncryptInPlace(plaintext); err != nil {
		t.Fatalf("EncryptInPlace: %v", err)
	}

	v0 := binary.LittleEndian.Uint32(plaintext[0:4])
	v1 := binary.LittleEndian.Uint32(plaintext[4:8])

	if v0 != 0xDD5989EC || v1 != 0xCE6D9490 {
		t.Errorf("ciphertext mismatch:\ngot:  v0=0x%08X v1=0x%08X\nwant: v0=0xDD5989EC v1=0xCE6D9490", v0, v1)
	}

	// Decrypt and verify round-trip
	if err := cipher.DecryptInPlace(plaintext); err != nil {
		t.Fatalf("DecryptInPlace: %v", err)
	}

	if !bytes.Equal(plaintext, original) {
		t.Errorf("round-trip failed:\ngot:  %x\nwant: %x", plaintext, original)
	}
}

// TestXTEA_DifferentKeysProduceDifferentCiphertext verifies that the same
// plaintext encrypted with two different keys yields different ciphertext.
func TestXTEA_DifferentKeysProduceDifferentCiphertext(t *testing.T) {
	t.Parallel()

	key1 := []uint32{0x11111111, 0x22222222, 0x33333333, 0x44444444}
	key2 := []uint32{0x55555555, 0x66666666, 0x77777777, 0x88888888}

	cipher1 := crypto.NewCipher(key1)
	cipher2 := crypto.NewCipher(key2)

	data1 := bytes.Repeat([]byte{0xFF}, 8)
	data2 := bytes.Repeat([]byte{0xFF}, 8)

	if err := cipher1.EncryptInPlace(data1); err != nil {
		t.Fatalf("cipher1.EncryptInPlace: %v", err)
	}

	if err := cipher2.EncryptInPlace(data2); err != nil {
		t.Fatalf("cipher2.EncryptInPlace: %v", err)
	}

	if bytes.Equal(data1, data2) {
		t.Error("same plaintext encrypted with different keys produced identical ciphertext")
	}
}

// TestXTEA_NilDataIsNoop verifies that nil slice input is treated as empty.
func TestXTEA_NilDataIsNoop(t *testing.T) {
	t.Parallel()

	cipher := crypto.NewCipher([]uint32{1, 2, 3, 4})

	if err := cipher.EncryptInPlace(nil); err != nil {
		t.Errorf("EncryptInPlace(nil): unexpected error: %v", err)
	}

	if err := cipher.DecryptInPlace(nil); err != nil {
		t.Errorf("DecryptInPlace(nil): unexpected error: %v", err)
	}
}
