package netutil

import (
	"bytes"
	"io"
	"log"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPrefixedConn_Read reads data from the butter pushed in by
// the Original connection
func TestPrefixedConn_Read(t *testing.T) {
	var c io.Writer
	var readBuf bytes.Buffer

	pc := &PrefixedConn{prefix: 0, writeConn: c, readBuf: readBuf}

	pc.readBuf.WriteString("foo")

	bs := make([]byte, 3)
	n, err := pc.Read(bs)

	require.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Equal(t, []byte("foo"), bs)

}

// TestPrefixedConn_Writes writes len(p) bytes from p to
// data stream and appends it with a header that is in total 3 bytes
func TestPrefixedConn_Write(t *testing.T) {

	buffer := bytes.Buffer{}
	var readBuf bytes.Buffer

	pc := &PrefixedConn{prefix: 0, writeConn: &buffer, readBuf: readBuf}
	n, err := pc.Write([]byte("foo"))

	require.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Equal(t, string([]byte("\x00\x00\x03foo")), buffer.String())
}

// TestReadHeader reads the first three bytes of the data and
// returns the prefix and size of the packets.
//
// Case Tested: aDuplex's clientConn is the initiator and has
// a prefix of 0 and wrote a msg "foo" with byte size of 3.
// 1) Prefix: Want(0) -- Got(0)
// 2) size: Want(3) -- Got(3)
func TestReadHeader(t *testing.T) {
	// errCh := make(chan error)
	connA, connB := net.Pipe()
	defer connA.Close()
	defer connB.Close()

	aDuplex := NewRPCDuplex(connA, true)
	bDuplex := NewRPCDuplex(connB, false)

	msg := []byte("foo")

	go func() {
		_, err := aDuplex.clientConn.Write(msg)
		if err != nil {
			log.Fatalln("Error writing from conn", err)
		}
		// errCh <- err
	}()

	prefix, size := bDuplex.ReadHeader()

	assert.Equal(t, byte(0), prefix)
	assert.Equal(t, uint16(len(msg)), size)
	// require.NoError(t, <-errCh)
}

// TestForward forwards data based on the prefix to appropriate prefixedConn
//
// Case Tested: aDuplex's clientConn is the initiator and has a prefix of 0
// and wrote a msg "foo" with byte size of 3 to bDuplex's serverConn
// 1) Msg: Want("foo") -- Got("foo")
func TestForward(t *testing.T) {

	t.Run("aDuplex's clientConn is Initiator", func(t *testing.T) {
		errCh := make(chan error)
		connA, connB := net.Pipe()
		defer connA.Close()
		defer connB.Close()

		aDuplex := NewRPCDuplex(connA, true)
		bDuplex := NewRPCDuplex(connB, false)

		msg := []byte("foo")

		go func() {
			_, err := aDuplex.clientConn.Write(msg)
			if err != nil {
				log.Fatalln("Error writing from conn", err)
			}
			errCh <- err
		}()

		prefix, size := bDuplex.ReadHeader()

		buf := bDuplex.Forward(prefix, size)
		assert.Equal(t, "foo", string(buf))
		require.NoError(t, <-errCh)
	})

}
