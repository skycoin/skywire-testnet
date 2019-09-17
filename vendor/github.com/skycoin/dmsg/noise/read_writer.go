package noise

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/SkycoinProject/dmsg/cipher"
	"github.com/SkycoinProject/dmsg/ioutil"
)

// ReadWriter implements noise encrypted read writer.
type ReadWriter struct {
	origin io.ReadWriter
	ns     *Noise
	rBuf   bytes.Buffer
	rMx    sync.Mutex
	wMx    sync.Mutex
}

// NewReadWriter constructs a new ReadWriter.
func NewReadWriter(rw io.ReadWriter, ns *Noise) *ReadWriter {
	return &ReadWriter{
		origin: rw,
		ns:     ns,
	}
}

func (rw *ReadWriter) Read(p []byte) (int, error) {
	rw.rMx.Lock()
	defer rw.rMx.Unlock()

	if rw.rBuf.Len() > 0 {
		return rw.rBuf.Read(p)
	}

	ciphertext, err := rw.readPacket()
	if err != nil {
		return 0, err
	}
	plaintext, err := rw.ns.DecryptUnsafe(ciphertext)
	if err != nil {
		return 0, err
	}
	return ioutil.BufRead(&rw.rBuf, plaintext, p)
}

func (rw *ReadWriter) readPacket() ([]byte, error) {
	h := make([]byte, 2)
	if _, err := io.ReadFull(rw.origin, h); err != nil {
		return nil, err
	}
	data := make([]byte, binary.BigEndian.Uint16(h))
	_, err := io.ReadFull(rw.origin, data)
	return data, err
}

func (rw *ReadWriter) Write(p []byte) (int, error) {
	rw.wMx.Lock()
	defer rw.wMx.Unlock()

	ciphertext := rw.ns.EncryptUnsafe(p)

	if err := rw.writePacket(ciphertext); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (rw *ReadWriter) writePacket(p []byte) error {
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, uint16(len(p)))
	_, err := rw.origin.Write(append(buf, p...))
	return err
}

// Handshake performs a Noise handshake using the provided io.ReadWriter.
func (rw *ReadWriter) Handshake(hsTimeout time.Duration) error {
	doneChan := make(chan error)
	go func() {
		if rw.ns.init {
			doneChan <- rw.initiatorHandshake()
		} else {
			doneChan <- rw.responderHandshake()
		}
	}()

	select {
	case err := <-doneChan:
		return err
	case <-time.After(hsTimeout):
		return errors.New("timeout")
	}
}

// LocalStatic returns the local static public key.
func (rw *ReadWriter) LocalStatic() cipher.PubKey {
	return rw.ns.LocalStatic()
}

// RemoteStatic returns the remote static public key.
func (rw *ReadWriter) RemoteStatic() cipher.PubKey {
	return rw.ns.RemoteStatic()
}

func (rw *ReadWriter) initiatorHandshake() error {
	for {
		msg, err := rw.ns.HandshakeMessage()
		if err != nil {
			return err
		}

		if err := rw.writePacket(msg); err != nil {
			return err
		}

		if rw.ns.HandshakeFinished() {
			break
		}

		res, err := rw.readPacket()
		if err != nil {
			return err
		}

		if err = rw.ns.ProcessMessage(res); err != nil {
			return err
		}

		if rw.ns.HandshakeFinished() {
			break
		}
	}

	return nil
}

func (rw *ReadWriter) responderHandshake() error {
	for {
		msg, err := rw.readPacket()
		if err != nil {
			return err
		}

		if err := rw.ns.ProcessMessage(msg); err != nil {
			return err
		}

		if rw.ns.HandshakeFinished() {
			break
		}

		res, err := rw.ns.HandshakeMessage()
		if err != nil {
			return err
		}

		if err := rw.writePacket(res); err != nil {
			return err
		}

		if rw.ns.HandshakeFinished() {
			break
		}
	}

	return nil
}
