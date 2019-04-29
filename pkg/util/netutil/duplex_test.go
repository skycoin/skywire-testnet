package netutil

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDuplex(t *testing.T) {
	sConn, cConn := net.Pipe()
	defer func() {
		require.NoError(t, sConn.Close())
		require.NoError(t, cConn.Close())
	}()

	aDuplex := NewRPCDuplex(cConn, true)
	bDuplex := NewRPCDuplex(sConn, false)

	t.Run("prefixedConn client can communicate with server", func(t *testing.T) {

		go func() {
			n, err := aDuplex.clientConn.Write([]byte("foo"))
			require.NoError(t, err)
			assert.Equal(t, 3, n)
		}()

		// Make a []byte with size of 4 because Write appends a 0 or 1
		buf := make([]byte, 4)
		n, err := sConn.Read(buf)
		require.NoError(t, err)
		assert.Equal(t, 4, n)
		assert.Equal(t, []byte("\000foo"), buf)
	})

	t.Run("prefixedConn server can communicate with client", func(t *testing.T) {

		go func() {
			n, err := bDuplex.serverConn.Write([]byte("foo"))
			require.NoError(t, err)
			assert.Equal(t, 3, n)
		}()

		buf := make([]byte, 4)
		n, err := cConn.Read(buf)
		require.NoError(t, err)
		assert.Equal(t, 4, n)
		assert.Equal(t, []byte("\000foo"), buf)
	})

	t.Run("prefixedConn client can communicate prefixedConn server", func(t *testing.T) {

		go func() {
			n, err := aDuplex.clientConn.Write([]byte("foo"))
			require.NoError(t, err)
			assert.Equal(t, 3, n)
		}()

		buf := make([]byte, 4)
		n, err := bDuplex.serverConn.Read(buf)
		require.NoError(t, err)
		assert.Equal(t, 4, n)
		assert.Equal(t, []byte("\000foo"), buf)
	})

}
