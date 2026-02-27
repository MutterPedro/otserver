package buffer_test

import (
	"testing"

	"github.com/MutterPedro/otserver/internal/buffer"
)

// TestAcceptance_ZeroAllocationPacketCycle verifies that the full NetworkMessage
// lifecycle (Get → Write → Read → Release) produces exactly 0 heap allocations
// in steady state. This prevents GC pauses during hot-path packet processing
// in a high-population MMORPG server.
func TestAcceptance_ZeroAllocationPacketCycle(t *testing.T) {
	// Not parallel: AllocsPerRun uses runtime.ReadMemStats which requires exclusive GOMAXPROCS.

	// Warm up the pool so subsequent Gets reuse pooled buffers.
	msg := buffer.GetNetworkMessage()
	msg.Release()

	allocs := testing.AllocsPerRun(100, func() {
		msg := buffer.GetNetworkMessage()

		msg.WriteByte(0x0A)
		msg.WriteUint16(0x1234)
		msg.WriteUint32(0xDEADBEEF)

		_, _ = msg.ReadByte()
		_, _ = msg.ReadUint16()
		_, _ = msg.ReadUint32()

		msg.Release()
	})

	if allocs != 0 {
		t.Errorf("expected 0 allocations per packet cycle, got %.0f", allocs)
	}
}
