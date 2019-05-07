package noise

import (
	"errors"
	"io"
	"sync"
	"time"

	"github.com/skycoin/skywire/internal/ioutil"
	"github.com/skycoin/skywire/pkg/cipher"
)

// ReadWriter implements noise encrypted read writer.
type ReadWriter struct {
	lrw *ioutil.LenReadWriter
	ns  *Noise

	rMx sync.Mutex
	wMx sync.Mutex
}

// NewReadWriter constructs a new ReadWriter.
func NewReadWriter(rw io.ReadWriter, ns *Noise) *ReadWriter {
	return &ReadWriter{
		lrw: ioutil.NewLenReadWriter(rw),
		ns:  ns,
	}
}

func (rw *ReadWriter) Read(p []byte) (int, error) {
	rw.rMx.Lock()
	defer rw.rMx.Unlock()

	ciphertext, err := rw.lrw.ReadPacket()
	if err != nil {
		return 0, err
	}
	plaintext, err := rw.ns.DecryptUnsafe(ciphertext)
	if err != nil {
		return 0, err
	}
	if len(plaintext) > len(p) {
		return 0, io.ErrShortBuffer
	}
	return copy(p, plaintext), nil
}

func (rw *ReadWriter) Write(p []byte) (n int, err error) {
	rw.wMx.Lock()
	defer rw.wMx.Unlock()

	ciphertext := rw.ns.EncryptUnsafe(p)
	n, err = rw.lrw.Write(ciphertext)
	if n != len(ciphertext) {
		err = io.ErrShortWrite
		return
	}
	return len(p), err
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

		if _, err := rw.lrw.Write(msg); err != nil {
			return err
		}

		if rw.ns.HandshakeFinished() {
			break
		}

		res, err := rw.lrw.ReadPacket()
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
		msg, err := rw.lrw.ReadPacket()
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

		if _, err := rw.lrw.Write(res); err != nil {
			return err
		}

		if rw.ns.HandshakeFinished() {
			break
		}
	}

	return nil
}
