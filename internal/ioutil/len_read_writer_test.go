package ioutil

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLenReadWriter(t *testing.T) {
	in, out := net.Pipe()
	rwIn := NewLenReadWriter(in)
	rwOut := NewLenReadWriter(out)

	errCh := make(chan error)
	go func() {
		_, err := rwIn.Write([]byte("foo"))
		errCh <- err
	}()

	buf := make([]byte, 2)
	n, err := rwOut.Read(buf)
	require.NoError(t, err)
	require.NoError(t, <-errCh)
	assert.Equal(t, 2, n)
	assert.Equal(t, []byte("fo"), buf)

	buf = make([]byte, 2)
	n, err = rwOut.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, []byte("o"), buf[:n])

	go func() {
		_, err := rwIn.Write([]byte("foo"))
		errCh <- err
	}()

	packet, err := rwOut.ReadPacket()
	require.NoError(t, err)
	require.NoError(t, <-errCh)
	assert.Equal(t, []byte("foo"), packet)

	go func() {
		_, err := rwOut.ReadPacket()
		errCh <- err
	}()

	n, err = rwIn.Write([]byte("bar"))
	require.NoError(t, err)
	require.NoError(t, <-errCh)
	assert.Equal(t, 3, n)
}
