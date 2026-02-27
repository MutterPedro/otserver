package network

import (
	"encoding/binary"
	"fmt"
)

type Packet struct {
	Payload []byte
}

// DecodePacket parses a length-prefixed Tibia packet from raw bytes.
// Format: [uint16 LE length][payload bytes...]
// Returns an error if data is too short or the declared length exceeds available data.
// It must NEVER panic on any input.
func DecodePacket(data []byte) error {
	if len(data) < 2 {
		return fmt.Errorf("packet too short: need at least 2 bytes for length header, got %d", len(data))
	}

	length := int(binary.LittleEndian.Uint16(data[0:2]))

	if len(data) < 2+length {
		return fmt.Errorf("truncated payload: declared length %d but only %d bytes available", length, len(data)-2)
	}

	_ = data[2 : 2+length]
	return nil
}
