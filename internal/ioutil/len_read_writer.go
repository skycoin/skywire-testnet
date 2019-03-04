package ioutil

import (
	"bytes"
	"encoding/binary"
	"io"
)

// LenReadWriter writes len prepended packets and always reads whole
// packet. If read buffer is smaller than packet, LenReadWriter will
// buffer unread part and will return it first in subsequent reads.
type LenReadWriter struct {
	io.ReadWriter
	buf *bytes.Buffer
}

// NewLenReadWriter constructs a new LenReadWriter.
func NewLenReadWriter(rw io.ReadWriter) *LenReadWriter {
	return &LenReadWriter{rw, new(bytes.Buffer)}
}

// ReadPacket returns single received len prepended packet.
func (rw *LenReadWriter) ReadPacket() (data []byte, err error) {
	var size uint16
	if err = binary.Read(rw.ReadWriter, binary.BigEndian, &size); err != nil {
		return
	}

	data = make([]byte, size)
	_, err = io.ReadFull(rw.ReadWriter, data)
	return data, err
}

func (rw *LenReadWriter) Read(p []byte) (n int, err error) {
	if rw.buf.Len() != 0 {
		return rw.buf.Read(p)
	}

	var data []byte
	data, err = rw.ReadPacket()
	if err != nil {
		return
	}

	if len(data) > len(p) {
		if _, err := rw.buf.Write(data[len(p):]); err != nil {
			return 0, io.ErrShortBuffer
		}

		return copy(p, data[:len(p)]), nil
	}

	return copy(p, data), nil
}

func (rw *LenReadWriter) Write(p []byte) (n int, err error) {
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, uint16(len(p)))
	n, err = rw.ReadWriter.Write(append(buf, p...))
	return n - 2, err
}
