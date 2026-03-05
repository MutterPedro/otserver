package iomap_test

import (
	"testing"

	"github.com/MutterPedro/otserver/internal/iomap"
)

// TestPropStreamReadWriteRoundTrip verifies that values written with PropWriter
// can be read back with PropStream and produce identical results.
func TestPropStreamReadWriteRoundTrip(t *testing.T) {
	t.Parallel()

	w := iomap.NewPropWriter()
	w.WriteUint8(0x42)
	w.WriteUint16(0xBEEF)
	w.WriteUint32(0xDEADBEEF)
	w.WriteString("Hello")

	r := iomap.NewPropStream(w.Bytes())

	gotU8, err := r.ReadUint8()
	if err != nil {
		t.Fatalf("ReadUint8: %v", err)
	}
	if gotU8 != 0x42 {
		t.Errorf("ReadUint8 = 0x%02X, want 0x42", gotU8)
	}

	gotU16, err := r.ReadUint16()
	if err != nil {
		t.Fatalf("ReadUint16: %v", err)
	}
	if gotU16 != 0xBEEF {
		t.Errorf("ReadUint16 = 0x%04X, want 0xBEEF", gotU16)
	}

	gotU32, err := r.ReadUint32()
	if err != nil {
		t.Fatalf("ReadUint32: %v", err)
	}
	if gotU32 != 0xDEADBEEF {
		t.Errorf("ReadUint32 = 0x%08X, want 0xDEADBEEF", gotU32)
	}

	gotStr, err := r.ReadString()
	if err != nil {
		t.Fatalf("ReadString: %v", err)
	}
	if gotStr != "Hello" {
		t.Errorf("ReadString = %q, want %q", gotStr, "Hello")
	}

	if r.Remaining() != 0 {
		t.Errorf("Remaining = %d, want 0", r.Remaining())
	}
}

// TestPropStreamReadBeyondEnd verifies that reading past the end of the buffer
// returns an error rather than panicking.
func TestPropStreamReadBeyondEnd(t *testing.T) {
	t.Parallel()

	r := iomap.NewPropStream([]byte{0x01}) // only 1 byte

	// First read should succeed
	_, err := r.ReadUint8()
	if err != nil {
		t.Fatalf("first ReadUint8: %v", err)
	}

	// Second read should fail — buffer exhausted
	_, err = r.ReadUint8()
	if err == nil {
		t.Error("expected error on ReadUint8 past end, got nil")
	}

	// ReadUint16 on empty stream
	r2 := iomap.NewPropStream([]byte{})
	_, err = r2.ReadUint16()
	if err == nil {
		t.Error("expected error on ReadUint16 from empty stream, got nil")
	}

	// ReadUint32 on short stream
	r3 := iomap.NewPropStream([]byte{0x01, 0x02})
	_, err = r3.ReadUint32()
	if err == nil {
		t.Error("expected error on ReadUint32 from short stream, got nil")
	}
}

// TestPropStreamReadBytesOutOfBounds verifies that requesting more bytes than
// available returns an error.
func TestPropStreamReadBytesOutOfBounds(t *testing.T) {
	t.Parallel()

	r := iomap.NewPropStream([]byte{0x01, 0x02, 0x03})

	_, err := r.ReadBytes(10)
	if err == nil {
		t.Error("expected error on ReadBytes(10) from 3-byte stream, got nil")
	}
}

// TestPropStreamEmptyString verifies that a zero-length string round-trips correctly.
func TestPropStreamEmptyString(t *testing.T) {
	t.Parallel()

	w := iomap.NewPropWriter()
	w.WriteString("")

	r := iomap.NewPropStream(w.Bytes())
	got, err := r.ReadString()
	if err != nil {
		t.Fatalf("ReadString: %v", err)
	}
	if got != "" {
		t.Errorf("ReadString = %q, want empty string", got)
	}
}

// TestPropStreamLargeString verifies that a 1000-byte string round-trips correctly.
func TestPropStreamLargeString(t *testing.T) {
	t.Parallel()

	// Build a 1000-byte string
	data := make([]byte, 1000)
	for i := range data {
		data[i] = byte(i % 256)
	}
	s := string(data)

	w := iomap.NewPropWriter()
	w.WriteString(s)

	r := iomap.NewPropStream(w.Bytes())
	got, err := r.ReadString()
	if err != nil {
		t.Fatalf("ReadString: %v", err)
	}
	if got != s {
		t.Errorf("ReadString length = %d, want 1000", len(got))
	}
}

// TestPropStreamStringLengthOutOfBounds verifies that a string length prefix
// exceeding the remaining buffer returns an error.
func TestPropStreamStringLengthOutOfBounds(t *testing.T) {
	t.Parallel()

	// Craft a buffer with a uint16 length prefix of 100 but only 2 data bytes
	buf := []byte{
		0x64, 0x00, // length = 100 (little-endian)
		0x41, 0x42, // only 2 bytes of data
	}

	r := iomap.NewPropStream(buf)
	_, err := r.ReadString()
	if err == nil {
		t.Error("expected error on ReadString with length exceeding buffer, got nil")
	}
}

// TestPropStreamInt32RoundTrip verifies int32 read/write.
func TestPropStreamInt32RoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		val  int32
	}{
		{"zero", 0},
		{"positive", 12345},
		{"negative", -12345},
		{"max int32", 0x7FFFFFFF},
		{"min int32", -0x80000000},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			w := iomap.NewPropWriter()
			w.WriteInt32(tc.val)

			r := iomap.NewPropStream(w.Bytes())
			got, err := r.ReadInt32()
			if err != nil {
				t.Fatalf("ReadInt32: %v", err)
			}
			if got != tc.val {
				t.Errorf("ReadInt32 = %d, want %d", got, tc.val)
			}
		})
	}
}

// TestPropStreamInt64RoundTrip verifies int64 read/write.
func TestPropStreamInt64RoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		val  int64
	}{
		{"zero", 0},
		{"positive", 123456789012},
		{"negative", -123456789012},
		{"max int64", 0x7FFFFFFFFFFFFFFF},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			w := iomap.NewPropWriter()
			w.WriteInt64(tc.val)

			r := iomap.NewPropStream(w.Bytes())
			got, err := r.ReadInt64()
			if err != nil {
				t.Fatalf("ReadInt64: %v", err)
			}
			if got != tc.val {
				t.Errorf("ReadInt64 = %d, want %d", got, tc.val)
			}
		})
	}
}

// TestPropStreamSkip verifies that Skip advances the position correctly.
func TestPropStreamSkip(t *testing.T) {
	t.Parallel()

	w := iomap.NewPropWriter()
	w.WriteUint8(0xAA)
	w.WriteUint16(0xBBBB)
	w.WriteUint8(0xCC)

	r := iomap.NewPropStream(w.Bytes())

	// Skip the uint8 + uint16 (3 bytes)
	if err := r.Skip(3); err != nil {
		t.Fatalf("Skip(3): %v", err)
	}

	got, err := r.ReadUint8()
	if err != nil {
		t.Fatalf("ReadUint8 after skip: %v", err)
	}
	if got != 0xCC {
		t.Errorf("ReadUint8 after skip = 0x%02X, want 0xCC", got)
	}
}

// TestPropStreamSkipOutOfBounds verifies that Skip returns an error when
// skipping past the buffer end.
func TestPropStreamSkipOutOfBounds(t *testing.T) {
	t.Parallel()

	r := iomap.NewPropStream([]byte{0x01, 0x02})
	if err := r.Skip(10); err == nil {
		t.Error("expected error on Skip(10) from 2-byte stream, got nil")
	}
}

// TestPropWriterBytes verifies that Bytes returns the correct byte slice.
func TestPropWriterBytes(t *testing.T) {
	t.Parallel()

	w := iomap.NewPropWriter()
	if len(w.Bytes()) != 0 {
		t.Errorf("empty writer Bytes() length = %d, want 0", len(w.Bytes()))
	}

	w.WriteUint8(0xFF)
	b := w.Bytes()
	if len(b) != 1 || b[0] != 0xFF {
		t.Errorf("Bytes() = %v, want [0xFF]", b)
	}
}
