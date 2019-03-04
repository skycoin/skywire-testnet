package ioutil

import (
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/pkg/cipher"
)

func TestAckReadWriter(t *testing.T) {
	in, out := net.Pipe()
	rw1 := NewAckReadWriter(in, 100*time.Millisecond)
	rw2 := NewAckReadWriter(out, 100*time.Millisecond)

	errCh := make(chan error)
	go func() {
		_, err := rw1.Write([]byte("foo"))
		errCh <- err
	}()

	buf := make([]byte, 3)
	n, err := rw2.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Equal(t, []byte("foo"), buf)

	errCh2 := make(chan error)
	go func() {
		_, err = rw2.Write([]byte("bar"))
		errCh2 <- err
	}()

	require.NoError(t, <-errCh)

	buf = make([]byte, 3)
	n, err = rw1.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Equal(t, []byte("bar"), buf)

	require.NoError(t, rw1.Close())
	require.NoError(t, <-errCh2)
	require.NoError(t, rw2.Close())
}

func TestAckReadWriterCRCFailure(t *testing.T) {
	in, out := net.Pipe()
	rw1 := NewAckReadWriter(in, 100*time.Millisecond)
	rw2 := NewAckReadWriter(out, 100*time.Millisecond)

	errCh := make(chan error)
	go func() {
		_, err := rw1.Write([]byte("foo"))
		errCh <- err
	}()

	buf := make([]byte, 3)
	n, err := rw2.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Equal(t, []byte("foo"), buf)

	rw2.rcvAcks.set(0, &ack{nil, cipher.SumSHA256([]byte("bar"))})

	go rw2.Write([]byte("bar")) // nolint: errcheck

	err = <-errCh
	require.Error(t, err)
	assert.Equal(t, "invalid CRC", err.Error())

	buf = make([]byte, 3)
	n, err = rw1.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Equal(t, []byte("bar"), buf)

	require.NoError(t, rw1.Close())
	require.NoError(t, rw2.Close())
}

func TestAckReadWriterFlushOnClose(t *testing.T) {
	in, out := net.Pipe()
	rw1 := NewAckReadWriter(in, 100*time.Millisecond)
	rw2 := NewAckReadWriter(out, 100*time.Millisecond)

	errCh := make(chan error)
	go func() {
		_, err := rw1.Write([]byte("foo"))
		errCh <- err
	}()

	buf := make([]byte, 3)
	n, err := rw2.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Equal(t, []byte("foo"), buf)

	require.NoError(t, rw2.Close())
	require.NoError(t, <-errCh)

	require.NoError(t, rw1.Close())
}

func TestAckReadWriterPartialRead(t *testing.T) {
	in, out := net.Pipe()
	rw1 := NewAckReadWriter(in, 100*time.Millisecond)
	rw2 := NewAckReadWriter(out, 100*time.Millisecond)

	errCh := make(chan error)
	go func() {
		_, err := rw1.Write([]byte("foo"))
		errCh <- err
	}()

	buf := make([]byte, 2)
	n, err := rw2.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, 2, n)
	assert.Equal(t, []byte("fo"), buf)

	n, err = rw2.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, []byte("o"), buf[:n])

	require.NoError(t, rw2.Close())
	require.NoError(t, rw1.Close())
}

func TestAckReadWriterReadError(t *testing.T) {
	in, out := net.Pipe()
	rw := NewAckReadWriter(in, 100*time.Millisecond)

	errCh := make(chan error)
	go func() {
		_, err := rw.Read([]byte{})
		errCh <- err
	}()

	require.NoError(t, out.Close())

	err := <-errCh
	require.Error(t, err)
	assert.Equal(t, io.EOF, err)

	require.NoError(t, rw.Close())
}
