package crypto_test

import (
	stdadler32 "hash/adler32"
	"testing"

	"github.com/MutterPedro/otserver/internal/crypto"
)

// TestAdler32_KnownValues validates the Tibia Adler32 against known input/output
// pairs to prevent regressions.
func TestAdler32_KnownValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []byte
		expected uint32
	}{
		{
			name:     "single byte 0x01",
			input:    []byte{0x01},
			expected: stdadler32.Checksum([]byte{0x01}),
		},
		{
			name:     "ASCII hello",
			input:    []byte("Hello"),
			expected: stdadler32.Checksum([]byte("Hello")),
		},
		{
			name:     "all zeros 8 bytes",
			input:    make([]byte, 8),
			expected: stdadler32.Checksum(make([]byte, 8)),
		},
		{
			name:     "sequential bytes",
			input:    []byte{0, 1, 2, 3, 4, 5, 6, 7},
			expected: stdadler32.Checksum([]byte{0, 1, 2, 3, 4, 5, 6, 7}),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := crypto.Adler32(tc.input)
			if got != tc.expected {
				t.Errorf("Adler32(%q) = 0x%08X, want 0x%08X", tc.input, got, tc.expected)
			}
		})
	}
}

// TestAdler32_EmptyInput verifies that an empty byte slice returns the initial
// Adler32 value (1), matching the standard algorithm's starting state.
func TestAdler32_EmptyInput(t *testing.T) {
	t.Parallel()

	got := crypto.Adler32([]byte{})
	if got != 1 {
		t.Errorf("Adler32(empty) = 0x%08X, want 0x00000001", got)
	}
}

// TestAdler32RFC1950_Mismatch explicitly proves that the Tibia Adler32
// returns 0 for oversized input (exceeding the network message limit), while
// the standard library hash/adler32 computes a valid checksum. This prevents
// future engineers from "refactoring" to use the stdlib.
func TestAdler32RFC1950_Mismatch(t *testing.T) {
	t.Parallel()

	// Tibia's max network message size is 24590 bytes.
	oversized := make([]byte, 24591)
	for i := range oversized {
		oversized[i] = byte(i % 256)
	}

	tibiaResult := crypto.Adler32(oversized)
	stdResult := stdadler32.Checksum(oversized)

	if tibiaResult != 0 {
		t.Errorf("Tibia Adler32 on oversized input = 0x%08X, want 0 (reject oversized)", tibiaResult)
	}

	if stdResult == 0 {
		t.Error("stdlib Adler32 on oversized input returned 0; expected a valid checksum")
	}

	if tibiaResult == stdResult {
		t.Error("Tibia Adler32 matches stdlib for oversized input; constraint is not enforced")
	}
}

// TestAdler32_NilInput verifies that a nil slice is treated as empty.
func TestAdler32_NilInput(t *testing.T) {
	t.Parallel()

	got := crypto.Adler32(nil)
	if got != 1 {
		t.Errorf("Adler32(nil) = 0x%08X, want 0x00000001", got)
	}
}

// TestAdler32_LargeValidInput verifies that a large (but valid) input within
// the Tibia network message size limit produces a correct checksum.
func TestAdler32_LargeValidInput(t *testing.T) {
	t.Parallel()

	data := make([]byte, 24590)
	for i := range data {
		data[i] = byte(i % 256)
	}

	got := crypto.Adler32(data)
	expected := stdadler32.Checksum(data)

	if got != expected {
		t.Errorf("Adler32(%d bytes) = 0x%08X, want 0x%08X", len(data), got, expected)
	}
}
