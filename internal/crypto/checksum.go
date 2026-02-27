package crypto

import "github.com/MutterPedro/otserver/internal/protocol"

const (
	adlerMod  = 65521
	adlerNmax = 5552
)

// Adler32 computes the Tibia-variant Adler32 checksum of data.
// It returns 0 if the data exceeds the maximum network message size (24590 bytes),
// enforcing Tibia's protocol constraint. For nil or empty input it returns 1.
func Adler32(data []byte) uint32 {
	if len(data) > protocol.MaxNetworkMessageSize {
		return 0
	}

	a := uint32(1)
	b := uint32(0)

	remaining := len(data)
	offset := 0

	for remaining > 0 {
		n := adlerNmax
		if remaining < n {
			n = remaining
		}

		for i := 0; i < n; i++ {
			a += uint32(data[offset+i])
			b += a
		}

		a %= adlerMod
		b %= adlerMod

		offset += n
		remaining -= n
	}

	return (b << 16) | a
}
