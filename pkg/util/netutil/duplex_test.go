package netutil

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrefixedConn_Read(t *testing.T) {
	sConn, cConn := net.Pipe()
	defer func() {
		require.NoError(t, sConn.Close())
		require.NoError(t, cConn.Close())
	}()

	sDuplex := NewRPCDuplex(sConn, true)
	// cDuplex := NewRPCDuplex(cConn, false)

	sDuplex.serverConn.readBuf.Write([]byte("\x00foo")) // Passed
	// sDuplex.clientConn.readBuf.Write([]byte("foo")) // Failed
	// cDuplex.serverConn.readBuf.Write([]byte("foo")) // Failed
	// cDuplex.clientConn.readBuf.Write([]byte("foo")) // Failed

	// Make a []byte with size of 4 because read removes a prefix
	buf := make([]byte, 3)
	n, err := sDuplex.serverConn.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Equal(t, []byte("foo"), buf)
}

func TestPrefixedConn_Write(t *testing.T) {
	sConn, cConn := net.Pipe()
	defer func() {
		require.NoError(t, sConn.Close())
		require.NoError(t, cConn.Close())
	}()

	cDuplex := NewRPCDuplex(cConn, true)

	go func() {
		n, err := cDuplex.clientConn.Write([]byte("foo"))
		require.NoError(t, err)
		assert.Equal(t, 3, n)
	}()

	// Make a []byte with size of 4 because Write appends a prefix
	buf := make([]byte, 4)
	n, err := sConn.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, 4, n)
	assert.Equal(t, []byte("\x00foo"), buf)
}
