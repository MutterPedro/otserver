package buffer_test

import (
	"testing"

	"github.com/MutterPedro/otserver/internal/buffer"
)

// TestNetworkMessage_WriteByte writes a single byte and reads it back.
func TestNetworkMessage_WriteByte(t *testing.T) {
	t.Parallel()

	msg := buffer.GetNetworkMessage()
	defer msg.Release()

	msg.WriteByte(0x42)

	got, err := msg.ReadByte()
	if err != nil {
		t.Fatalf("ReadByte: %v", err)
	}

	if got != 0x42 {
		t.Errorf("ReadByte = 0x%02X, want 0x42", got)
	}
}

// TestNetworkMessage_WriteUint16 writes a uint16 in little-endian and reads it back.
func TestNetworkMessage_WriteUint16(t *testing.T) {
	t.Parallel()

	msg := buffer.GetNetworkMessage()
	defer msg.Release()

	msg.WriteUint16(0xBEEF)

	got, err := msg.ReadUint16()
	if err != nil {
		t.Fatalf("ReadUint16: %v", err)
	}

	if got != 0xBEEF {
		t.Errorf("ReadUint16 = 0x%04X, want 0xBEEF", got)
	}
}

// TestNetworkMessage_WriteUint32 writes a uint32 in little-endian and reads it back.
func TestNetworkMessage_WriteUint32(t *testing.T) {
	t.Parallel()

	msg := buffer.GetNetworkMessage()
	defer msg.Release()

	msg.WriteUint32(0xDEADBEEF)

	got, err := msg.ReadUint32()
	if err != nil {
		t.Fatalf("ReadUint32: %v", err)
	}

	if got != 0xDEADBEEF {
		t.Errorf("ReadUint32 = 0x%08X, want 0xDEADBEEF", got)
	}
}

// TestNetworkMessage_MultipleWritesThenReads verifies correct sequential
// reading of multiple values written in order.
func TestNetworkMessage_MultipleWritesThenReads(t *testing.T) {
	t.Parallel()

	msg := buffer.GetNetworkMessage()
	defer msg.Release()

	msg.WriteByte(0x0A)
	msg.WriteUint16(0x1234)
	msg.WriteUint32(0xCAFEBABE)

	b, err := msg.ReadByte()
	if err != nil {
		t.Fatalf("ReadByte: %v", err)
	}
	if b != 0x0A {
		t.Errorf("ReadByte = 0x%02X, want 0x0A", b)
	}

	u16, err := msg.ReadUint16()
	if err != nil {
		t.Fatalf("ReadUint16: %v", err)
	}
	if u16 != 0x1234 {
		t.Errorf("ReadUint16 = 0x%04X, want 0x1234", u16)
	}

	u32, err := msg.ReadUint32()
	if err != nil {
		t.Fatalf("ReadUint32: %v", err)
	}
	if u32 != 0xCAFEBABE {
		t.Errorf("ReadUint32 = 0x%08X, want 0xCAFEBABE", u32)
	}
}

// TestNetworkMessage_ReadPastEnd verifies that reading beyond written data
// returns an error.
func TestNetworkMessage_ReadPastEnd(t *testing.T) {
	t.Parallel()

	msg := buffer.GetNetworkMessage()
	defer msg.Release()

	msg.WriteByte(0xFF)

	// First read succeeds.
	_, err := msg.ReadByte()
	if err != nil {
		t.Fatalf("first ReadByte: %v", err)
	}

	// Second read should fail — no more data.
	_, err = msg.ReadByte()
	if err == nil {
		t.Error("expected error reading past end of message, got nil")
	}
}

// TestNetworkMessage_PoolReusability verifies that a released message can be
// reused from the pool without data leaks from the previous user.
func TestNetworkMessage_PoolReusability(t *testing.T) {
	t.Parallel()

	// Write data, then release.
	msg1 := buffer.GetNetworkMessage()
	msg1.WriteByte(0xAA)
	msg1.WriteUint32(0xDEADBEEF)
	msg1.Release()

	// Get a new message from the pool — should be clean.
	msg2 := buffer.GetNetworkMessage()
	defer msg2.Release()

	// Reading should fail since the message is empty (reset).
	_, err := msg2.ReadByte()
	if err == nil {
		t.Error("expected error reading from fresh pooled message, got nil (data leak)")
	}
}

// TestNetworkMessage_Length verifies that Length returns the number of written bytes.
func TestNetworkMessage_Length(t *testing.T) {
	t.Parallel()

	msg := buffer.GetNetworkMessage()
	defer msg.Release()

	if msg.Length() != 0 {
		t.Errorf("initial Length = %d, want 0", msg.Length())
	}

	msg.WriteByte(0x01)
	if msg.Length() != 1 {
		t.Errorf("after WriteByte Length = %d, want 1", msg.Length())
	}

	msg.WriteUint16(0x0102)
	if msg.Length() != 3 {
		t.Errorf("after WriteUint16 Length = %d, want 3", msg.Length())
	}

	msg.WriteUint32(0x01020304)
	if msg.Length() != 7 {
		t.Errorf("after WriteUint32 Length = %d, want 7", msg.Length())
	}
}

// TestNetworkMessage_Bytes returns the raw written data.
func TestNetworkMessage_Bytes(t *testing.T) {
	t.Parallel()

	msg := buffer.GetNetworkMessage()
	defer msg.Release()

	msg.WriteByte(0x0A)
	msg.WriteUint16(0x0102)

	got := msg.Bytes()
	want := []byte{0x0A, 0x02, 0x01} // 0x0102 in LE is [0x02, 0x01]

	if len(got) != len(want) {
		t.Fatalf("Bytes length = %d, want %d", len(got), len(want))
	}

	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Bytes[%d] = 0x%02X, want 0x%02X", i, got[i], want[i])
		}
	}
}

// TestNetworkMessage_ReadUint16PastEnd verifies that reading a uint16 when
// fewer than 2 bytes remain returns an error.
func TestNetworkMessage_ReadUint16PastEnd(t *testing.T) {
	t.Parallel()

	msg := buffer.GetNetworkMessage()
	defer msg.Release()

	msg.WriteByte(0xFF) // Only 1 byte available.

	_, err := msg.ReadUint16()
	if err == nil {
		t.Error("expected error reading uint16 with only 1 byte remaining, got nil")
	}
}

// TestNetworkMessage_ReadUint32PastEnd verifies that reading a uint32 when
// fewer than 4 bytes remain returns an error.
func TestNetworkMessage_ReadUint32PastEnd(t *testing.T) {
	t.Parallel()

	msg := buffer.GetNetworkMessage()
	defer msg.Release()

	msg.WriteUint16(0x1234) // Only 2 bytes available.

	_, err := msg.ReadUint32()
	if err == nil {
		t.Error("expected error reading uint32 with only 2 bytes remaining, got nil")
	}
}

// TestNetworkMessage_BoundaryValues verifies correct handling of minimum
// and maximum representable values for each type.
func TestNetworkMessage_BoundaryValues(t *testing.T) {
	t.Parallel()

	msg := buffer.GetNetworkMessage()
	defer msg.Release()

	msg.WriteByte(0x00)
	msg.WriteByte(0xFF)
	msg.WriteUint16(0x0000)
	msg.WriteUint16(0xFFFF)
	msg.WriteUint32(0x00000000)
	msg.WriteUint32(0xFFFFFFFF)

	tests := []struct {
		name string
		read func() (uint32, error)
		want uint32
	}{
		{"byte min", func() (uint32, error) { v, e := msg.ReadByte(); return uint32(v), e }, 0x00},
		{"byte max", func() (uint32, error) { v, e := msg.ReadByte(); return uint32(v), e }, 0xFF},
		{"uint16 min", func() (uint32, error) { v, e := msg.ReadUint16(); return uint32(v), e }, 0x0000},
		{"uint16 max", func() (uint32, error) { v, e := msg.ReadUint16(); return uint32(v), e }, 0xFFFF},
		{"uint32 min", func() (uint32, error) { v, e := msg.ReadUint32(); return uint32(v), e }, 0x00000000},
		{"uint32 max", func() (uint32, error) { v, e := msg.ReadUint32(); return uint32(v), e }, 0xFFFFFFFF},
	}

	for _, tc := range tests {
		got, err := tc.read()
		if err != nil {
			t.Errorf("%s: unexpected error: %v", tc.name, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%s: got 0x%X, want 0x%X", tc.name, got, tc.want)
		}
	}
}

// TestNetworkMessage_BytesReturnsEmptyForFreshMessage verifies that Bytes
// returns an empty slice for a message with no writes.
func TestNetworkMessage_BytesReturnsEmptyForFreshMessage(t *testing.T) {
	t.Parallel()

	msg := buffer.GetNetworkMessage()
	defer msg.Release()

	got := msg.Bytes()
	if len(got) != 0 {
		t.Errorf("Bytes() on fresh message has length %d, want 0", len(got))
	}
}

// TestNetworkMessage_ReadFromEmptyMessage verifies all read methods return
// errors when the message has no data.
func TestNetworkMessage_ReadFromEmptyMessage(t *testing.T) {
	t.Parallel()

	msg := buffer.GetNetworkMessage()
	defer msg.Release()

	if _, err := msg.ReadByte(); err == nil {
		t.Error("ReadByte on empty message: expected error, got nil")
	}

	if _, err := msg.ReadUint16(); err == nil {
		t.Error("ReadUint16 on empty message: expected error, got nil")
	}

	if _, err := msg.ReadUint32(); err == nil {
		t.Error("ReadUint32 on empty message: expected error, got nil")
	}
}
