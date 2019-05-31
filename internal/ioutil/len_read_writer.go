package ioutil

import (
	"bytes"
	"encoding/binary"
	"io"
	"sync"
)

// LenReadWriter writes len prepended packets and always reads whole
// packet. If read buffer is smaller than packet, LenReadWriter will
// buffer unread part and will return it first in subsequent reads.
type LenReadWriter struct {
	io.ReadWriter
	buf bytes.Buffer
	mx  sync.Mutex
}

// NewLenReadWriter constructs a new LenReadWriter.
func NewLenReadWriter(rw io.ReadWriter) *LenReadWriter {
	return &LenReadWriter{ReadWriter: rw}
}

// ReadPacket returns single received len prepended packet.
func (rw *LenReadWriter) ReadPacket() ([]byte, error) {
	h := make([]byte, 2)
	if _, err := io.ReadFull(rw.ReadWriter, h); err != nil {
		return nil, err
	}
	data := make([]byte, binary.BigEndian.Uint16(h))
	_, err := io.ReadFull(rw.ReadWriter, data)
	return data, err
}

func (rw *LenReadWriter) Read(p []byte) (n int, err error) {
	rw.mx.Lock()
	defer rw.mx.Unlock()

	if rw.buf.Len() != 0 {
		return rw.buf.Read(p)
	}

	var data []byte
	data, err = rw.ReadPacket()
	if err != nil {
		return
	}

	return BufRead(&rw.buf, data, p)
}

func (rw *LenReadWriter) Write(p []byte) (n int, err error) {
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, uint16(len(p)))
	n, err = rw.ReadWriter.Write(append(buf, p...))
	return n - 2, err
}
