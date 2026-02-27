// Package buffer provides fixed-size, pool-friendly byte buffers for reading
// and writing Tibia network packets without heap allocations in steady state.
package buffer

import (
	"encoding/binary"
	"errors"
	"sync"

	"github.com/MutterPedro/otserver/internal/protocol"
)

// errReadPastEnd is returned when a read operation exceeds the written data boundary.
var errReadPastEnd = errors.New("buffer: read past end of message")

// pool is a sync.Pool of NetworkMessage values to eliminate allocations on the hot path.
var pool = sync.Pool{
	New: func() any {
		return &NetworkMessage{}
	},
}

// NetworkMessage is a fixed-size, pool-friendly buffer for reading and writing
// Tibia network packets without heap allocations in steady state.
type NetworkMessage struct {
	buf    [protocol.MaxNetworkMessageSize]byte
	pos    int
	length int
}

// GetNetworkMessage returns a NetworkMessage from the pool, ready for use.
func GetNetworkMessage() *NetworkMessage {
	return pool.Get().(*NetworkMessage)
}

// Release resets the message and returns it to the pool.
func (m *NetworkMessage) Release() {
	m.pos = 0
	m.length = 0
	pool.Put(m)
}

// WriteByte writes a single byte at the write cursor and advances length.
// It implements [io.ByteWriter] and always returns a nil error.
func (m *NetworkMessage) WriteByte(v byte) error {
	m.buf[m.length] = v
	m.length++
	return nil
}

// WriteUint16 writes a uint16 in little-endian byte order and advances length by 2.
func (m *NetworkMessage) WriteUint16(v uint16) {
	binary.LittleEndian.PutUint16(m.buf[m.length:], v)
	m.length += 2
}

// WriteUint32 writes a uint32 in little-endian byte order and advances length by 4.
func (m *NetworkMessage) WriteUint32(v uint32) {
	binary.LittleEndian.PutUint32(m.buf[m.length:], v)
	m.length += 4
}

// ReadByte reads a single byte at the read cursor and advances pos.
func (m *NetworkMessage) ReadByte() (byte, error) {
	if m.pos >= m.length {
		return 0, errReadPastEnd
	}
	v := m.buf[m.pos]
	m.pos++
	return v, nil
}

// ReadUint16 reads a uint16 in little-endian byte order and advances pos by 2.
func (m *NetworkMessage) ReadUint16() (uint16, error) {
	if m.pos+2 > m.length {
		return 0, errReadPastEnd
	}
	v := binary.LittleEndian.Uint16(m.buf[m.pos:])
	m.pos += 2
	return v, nil
}

// ReadUint32 reads a uint32 in little-endian byte order and advances pos by 4.
func (m *NetworkMessage) ReadUint32() (uint32, error) {
	if m.pos+4 > m.length {
		return 0, errReadPastEnd
	}
	v := binary.LittleEndian.Uint32(m.buf[m.pos:])
	m.pos += 4
	return v, nil
}

// Length returns the number of bytes written to the message.
func (m *NetworkMessage) Length() int {
	return m.length
}

// Bytes returns a slice of the internal buffer containing the written data.
func (m *NetworkMessage) Bytes() []byte {
	return m.buf[:m.length]
}
