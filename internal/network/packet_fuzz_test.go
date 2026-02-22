package network_test

import (
	"testing"

	"github.com/MutterPedro/otserver/internal/network"
)

// FuzzPacketDecoder ensures that the packet decoder never panics on arbitrary
// input — a critical property for a public-facing server that accepts raw bytes
// from untrusted clients.
func FuzzPacketDecoder(f *testing.F) {
	// Seed corpus: valid-ish packet shapes the decoder might see.
	f.Add([]byte{})
	f.Add([]byte{0x00})
	// 2-byte length header with valid payload.
	f.Add([]byte{0x02, 0x00, 0xFF, 0xFF})
	// Zero-length packet.
	f.Add([]byte{0x00, 0x00})
	// 1-byte payload.
	f.Add([]byte{0x01, 0x00, 0x0A})
	// Length header claims more bytes than are present.
	f.Add([]byte{0xFF, 0xFF, 0x00, 0x01, 0x02, 0x03, 0x04})

	f.Fuzz(func(t *testing.T, data []byte) {
		// The decoder must never panic, regardless of input.
		// Return values (packet, error) are ignored; only panic matters.
		_ = network.DecodePacket(data)
	})
}
