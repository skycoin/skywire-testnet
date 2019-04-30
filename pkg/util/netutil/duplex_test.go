package netutil

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPrefixedConn_Read reads in a slice of bytes with a prefix and removes it
func TestPrefixedConn_Read(t *testing.T) {
	var c io.Writer
	var readBuf bytes.Buffer

	pc := &PrefixedConn{prefix: 0, writeConn: c, readBuf: readBuf}

	pc.readBuf.WriteString("\x00\x00\x03foo")

	bs := make([]byte, 3)
	n, err := pc.Read(bs)

	require.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Equal(t, []byte("foo"), bs)
}

// TestPrefixedConn_Writes writes len(p) bytes from p to underlying data stream and appends it with a prefix
func TestPrefixedConn_Write(t *testing.T) {

	buffer := bytes.Buffer{}
	var readBuf bytes.Buffer

	pc := &PrefixedConn{prefix: 0, writeConn: &buffer, readBuf: readBuf}
	n, err := pc.Write([]byte("foo"))

	require.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Equal(t, string([]byte("\x00\x00\x03foo")), buffer.String())
}
