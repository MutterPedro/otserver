// Package crypto implements cryptographic primitives used by the Tibia game
// protocol, including the XTEA block cipher with CBC-like chaining.
package crypto

import (
	"encoding/binary"
	"fmt"
)

const (
	xteaBlockSize = 8
	xteaRounds    = 32
	xteaDelta     = uint32(0x9E3779B9)
)

// xteaSumInit is delta*rounds computed at init time to avoid constant overflow.
var xteaSumInit = func() uint32 {
	var s uint32
	for range xteaRounds {
		s += xteaDelta
	}
	return s
}()

// Cipher holds the key state for XTEA encryption and decryption.
type Cipher struct {
	key [4]uint32
}

// NewCipher creates a new XTEA cipher from a 4-element uint32 key.
func NewCipher(key []uint32) *Cipher {
	c := &Cipher{}
	copy(c.key[:], key)
	return c
}

// EncryptInPlace encrypts data in-place using Tibia's XTEA with CBC-like
// chaining. The data length must be a multiple of 8. Empty data is a no-op.
func (c *Cipher) EncryptInPlace(data []byte) error {
	n := len(data)
	if n == 0 {
		return nil
	}

	if n%xteaBlockSize != 0 {
		return fmt.Errorf("xtea: data length %d is not a multiple of %d", n, xteaBlockSize)
	}

	var prevV0, prevV1 uint32

	for i := 0; i < n; i += xteaBlockSize {
		block := data[i : i+xteaBlockSize]

		v0 := binary.LittleEndian.Uint32(block[0:4])
		v1 := binary.LittleEndian.Uint32(block[4:8])

		// CBC: XOR plaintext block with previous ciphertext block.
		v0 ^= prevV0
		v1 ^= prevV1

		// Standard XTEA encryption with 32 rounds.
		var sum uint32
		for range xteaRounds {
			v0 += ((v1<<4 ^ v1>>5) + v1) ^ (sum + c.key[sum&3])
			sum += xteaDelta
			v1 += ((v0<<4 ^ v0>>5) + v0) ^ (sum + c.key[(sum>>11)&3])
		}

		binary.LittleEndian.PutUint32(block[0:4], v0)
		binary.LittleEndian.PutUint32(block[4:8], v1)

		prevV0 = v0
		prevV1 = v1
	}

	return nil
}

// DecryptInPlace decrypts data in-place using Tibia's XTEA with CBC-like
// chaining. The data length must be a multiple of 8. Empty data is a no-op.
func (c *Cipher) DecryptInPlace(data []byte) error {
	n := len(data)
	if n == 0 {
		return nil
	}

	if n%xteaBlockSize != 0 {
		return fmt.Errorf("xtea: data length %d is not a multiple of %d", n, xteaBlockSize)
	}

	var prevV0, prevV1 uint32

	for i := 0; i < n; i += xteaBlockSize {
		block := data[i : i+xteaBlockSize]

		cipherV0 := binary.LittleEndian.Uint32(block[0:4])
		cipherV1 := binary.LittleEndian.Uint32(block[4:8])

		// Standard XTEA decryption with 32 rounds.
		v0 := cipherV0
		v1 := cipherV1
		sum := xteaSumInit

		for range xteaRounds {
			v1 -= ((v0<<4 ^ v0>>5) + v0) ^ (sum + c.key[(sum>>11)&3])
			sum -= xteaDelta
			v0 -= ((v1<<4 ^ v1>>5) + v1) ^ (sum + c.key[sum&3])
		}

		// CBC: XOR decrypted block with previous ciphertext block.
		v0 ^= prevV0
		v1 ^= prevV1

		binary.LittleEndian.PutUint32(block[0:4], v0)
		binary.LittleEndian.PutUint32(block[4:8], v1)

		prevV0 = cipherV0
		prevV1 = cipherV1
	}

	return nil
}
