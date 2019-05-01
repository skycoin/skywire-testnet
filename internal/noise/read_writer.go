package noise

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/skycoin/skywire/internal/ioutil"
	"github.com/skycoin/skywire/pkg/cipher"
)

// ReadWriter implements noise encrypted read writer.
type ReadWriter struct {
	lrw *ioutil.LenReadWriter
	ns  *Noise
}

// NewReadWriter constructs a new ReadWriter.
func NewReadWriter(rw io.ReadWriter, ns *Noise) *ReadWriter {
	return &ReadWriter{ioutil.NewLenReadWriter(rw), ns}
}

// ReadPacket returns single received len prepended packet.
func (rw *ReadWriter) ReadPacket() (data []byte, err error) {
	data, err = rw.lrw.ReadPacket()
	if err != nil {
		return
	}

	return rw.ns.Decrypt(data)
}

// ReadPacketUnsafe returns single received len prepended packet using DecryptUnsafe.
func (rw *ReadWriter) ReadPacketUnsafe() (data []byte, err error) {
	data, err = rw.lrw.ReadPacket()
	if err != nil {
		return
	}

	return rw.ns.DecryptUnsafe(data)
}

func (rw *ReadWriter) Read(p []byte) (n int, err error) {
	var data []byte
	data, err = rw.ReadPacket()
	if err != nil {
		return
	}

	if len(data) > len(p) {
		err = io.ErrShortBuffer
		return
	}

	return copy(p, data), nil
}

// WriteUnsafe implements io.Writer using EncryptUnsafe.
func (rw *ReadWriter) WriteUnsafe(p []byte) (n int, err error) {
	encrypted := rw.ns.EncryptUnsafe(p)
	n, err = rw.lrw.Write(encrypted)
	if n != len(encrypted) {
		err = io.ErrShortWrite
		return
	}
	return len(p), err
}

func (rw *ReadWriter) Write(p []byte) (n int, err error) {
	encrypted := rw.ns.Encrypt(p)
	n, err = rw.lrw.Write(encrypted)
	if n != len(encrypted) {
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

		fmt.Println("APP: Write(HS)", msg)
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
