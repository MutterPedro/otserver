package crypto_test

import (
	"bytes"
	"testing"

	"github.com/MutterPedro/otserver/internal/crypto"
)

// FuzzXTEARoundTrip ensures that the XTEA cipher never panics on arbitrary
// input and always round-trips correctly for valid (multiple-of-8) data.
func FuzzXTEARoundTrip(f *testing.F) {
	// Seed corpus with interesting inputs.
	f.Add([]byte{0, 0, 0, 0, 0, 0, 0, 0})
	f.Add([]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF})
	f.Add(bytes.Repeat([]byte{0x41}, 16))
	f.Add(make([]byte, 64))

	key := []uint32{0x12345678, 0x9ABCDEF0, 0xDEADBEEF, 0xCAFEBABE}
	cipher := crypto.NewCipher(key)

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data)%8 != 0 || len(data) == 0 {
			// Only test valid block-aligned data.
			err := cipher.EncryptInPlace(data)
			if len(data)%8 != 0 && len(data) > 0 && err == nil {
				t.Error("expected error for non-aligned data")
			}
			return
		}

		original := make([]byte, len(data))
		copy(original, data)

		if err := cipher.EncryptInPlace(data); err != nil {
			t.Fatalf("EncryptInPlace(%d bytes): %v", len(data), err)
		}

		if err := cipher.DecryptInPlace(data); err != nil {
			t.Fatalf("DecryptInPlace(%d bytes): %v", len(data), err)
		}

		if !bytes.Equal(data, original) {
			t.Fatalf("round-trip mismatch for %d bytes", len(data))
		}
	})
}
