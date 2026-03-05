// Package propstream provides lightweight binary serialization utilities
// for reading and writing little-endian encoded data streams.
package propstream

import (
	"encoding/binary"
	"errors"
)

var errReadPastEnd = errors.New("propstream: read past end of buffer")

// PropWriter serializes values into a byte buffer using little-endian encoding.
type PropWriter struct {
	buf []byte
}

// NewPropWriter returns a new PropWriter with an empty buffer.
func NewPropWriter() *PropWriter {
	return &PropWriter{}
}

// WriteUint8 appends a single byte to the buffer.
func (w *PropWriter) WriteUint8(v uint8) {
	w.buf = append(w.buf, v)
}

// WriteUint16 appends a little-endian uint16 to the buffer.
func (w *PropWriter) WriteUint16(v uint16) {
	w.buf = binary.LittleEndian.AppendUint16(w.buf, v)
}

// WriteUint32 appends a little-endian uint32 to the buffer.
func (w *PropWriter) WriteUint32(v uint32) {
	w.buf = binary.LittleEndian.AppendUint32(w.buf, v)
}

// WriteInt32 appends a little-endian int32 to the buffer.
func (w *PropWriter) WriteInt32(v int32) {
	w.buf = binary.LittleEndian.AppendUint32(w.buf, uint32(v))
}

// WriteInt64 appends a little-endian int64 to the buffer.
func (w *PropWriter) WriteInt64(v int64) {
	w.buf = binary.LittleEndian.AppendUint64(w.buf, uint64(v))
}

// WriteString appends a uint16 length prefix followed by the string bytes.
func (w *PropWriter) WriteString(s string) {
	w.WriteUint16(uint16(len(s)))
	w.buf = append(w.buf, s...)
}

// Bytes returns the accumulated buffer.
func (w *PropWriter) Bytes() []byte {
	return w.buf
}

// PropStream reads values from a byte buffer using little-endian encoding.
type PropStream struct {
	data []byte
	pos  int
}

// NewPropStream returns a new PropStream over the given data.
func NewPropStream(data []byte) *PropStream {
	return &PropStream{data: data}
}

// ReadUint8 reads a single byte from the stream.
func (r *PropStream) ReadUint8() (uint8, error) {
	if r.pos+1 > len(r.data) {
		return 0, errReadPastEnd
	}
	v := r.data[r.pos]
	r.pos++
	return v, nil
}

// ReadUint16 reads a little-endian uint16 from the stream.
func (r *PropStream) ReadUint16() (uint16, error) {
	if r.pos+2 > len(r.data) {
		return 0, errReadPastEnd
	}
	v := binary.LittleEndian.Uint16(r.data[r.pos:])
	r.pos += 2
	return v, nil
}

// ReadUint32 reads a little-endian uint32 from the stream.
func (r *PropStream) ReadUint32() (uint32, error) {
	if r.pos+4 > len(r.data) {
		return 0, errReadPastEnd
	}
	v := binary.LittleEndian.Uint32(r.data[r.pos:])
	r.pos += 4
	return v, nil
}

// ReadInt32 reads a little-endian int32 from the stream.
func (r *PropStream) ReadInt32() (int32, error) {
	v, err := r.ReadUint32()
	return int32(v), err
}

// ReadInt64 reads a little-endian int64 from the stream.
func (r *PropStream) ReadInt64() (int64, error) {
	if r.pos+8 > len(r.data) {
		return 0, errReadPastEnd
	}
	v := binary.LittleEndian.Uint64(r.data[r.pos:])
	r.pos += 8
	return int64(v), nil
}

// ReadString reads a uint16 length prefix followed by that many bytes as a string.
func (r *PropStream) ReadString() (string, error) {
	length, err := r.ReadUint16()
	if err != nil {
		return "", err
	}
	b, err := r.ReadBytes(int(length))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ReadBytes reads exactly n bytes from the stream.
func (r *PropStream) ReadBytes(n int) ([]byte, error) {
	if r.pos+n > len(r.data) {
		return nil, errReadPastEnd
	}
	b := r.data[r.pos : r.pos+n]
	r.pos += n
	return b, nil
}

// Skip advances the stream position by n bytes.
func (r *PropStream) Skip(n int) error {
	if r.pos+n > len(r.data) {
		return errReadPastEnd
	}
	r.pos += n
	return nil
}

// Remaining returns the number of unread bytes in the stream.
func (r *PropStream) Remaining() int {
	return len(r.data) - r.pos
}
