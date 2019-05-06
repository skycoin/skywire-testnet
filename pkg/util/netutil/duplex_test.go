package netutil

import (
	"bytes"
	"io"
	"log"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPrefixedConn_Read reads data from the butter pushed in by
// the Original connection
func TestPrefixedConn_Read(t *testing.T) {

	t.Run("successful prefixedConn read", func(t *testing.T) {
		var c io.Writer
		var readBuf bytes.Buffer

		pc := &PrefixedConn{prefix: 0, writeConn: c, readBuf: readBuf}

		pc.readBuf.WriteString("foo")

		bs := make([]byte, 3)
		n, err := pc.Read(bs)

		require.NoError(t, err)
		assert.Equal(t, 3, n)
		assert.Equal(t, []byte("foo"), bs)
	})

	t.Run("empty buffer prefixedConn read", func(t *testing.T) {
		var c io.Writer
		var readBuf bytes.Buffer

		pc := &PrefixedConn{prefix: 0, writeConn: c, readBuf: readBuf}

		var bs []byte
		n, err := pc.Read(bs)

		require.NoError(t, err)
		assert.Equal(t, 0, n)
		assert.Equal(t, []byte(nil), bs)
	})

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
func TestRPCDuplex_ReadHeader(t *testing.T) {
	// errCh := make(chan error)
	buf := make([]byte, 3)
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

	// To prevent read/write on closed pipe
	connB.Read(buf)
}

// TestRPCDuplex_Forward forwards data based on the prefix to appropriate prefixedConn
func TestRPCDuplex_Forward(t *testing.T) {

	t.Run("aDuplex's clientConn is Initiator", func(t *testing.T) {

		connA, connB := net.Pipe()

		var bs []byte

		aDuplex := NewRPCDuplex(connA, true)
		bDuplex := NewRPCDuplex(connB, false)

		msg := []byte("foo")
		go func() {
			_, err := aDuplex.clientConn.Write(msg)
			if err != nil {
				log.Fatalln("Error writing from conn", err)
			}
		}()

		prefix, size := bDuplex.ReadHeader()
		pc, bs := bDuplex.Forward(prefix, size)
		assert.Equal(t, "serverConn", pc.name)
		assert.Equal(t, "foo", string(bs))

	})
}

func whoWrites(whoWrites string, aDuplex *RPCDuplex, bDuplex *RPCDuplex) (*RPCDuplex, *RPCDuplex) {

	if whoWrites == "aDuplex" {

		msgWriter := aDuplex
		msgReader := bDuplex

		return msgReader, msgWriter
	}

	return bDuplex, aDuplex
}

func fromWhichBranch(branchConn string, msgWriter *RPCDuplex) *PrefixedConn {

	if branchConn == "clientConn" {
		return msgWriter.clientConn
	}

	return msgWriter.serverConn
}

var tables = []struct {
	description  string
	msg          string
	whoWrites    string
	initiatorA   bool
	initiatorB   bool
	branchConn   string
	expectedMsg  string
	expectedSize uint16
	expectedConn string
}{
	{description: "aDuplex's clientConn (initiator) sends msg to bDuplex's serverConn",
		msg:          "foo",
		initiatorA:   true,
		initiatorB:   false,
		whoWrites:    "aDuplex",
		branchConn:   "clientConn",
		expectedMsg:  "foo",
		expectedSize: uint16(3),
		expectedConn: "serverConn",
	},
	{description: "aDuplex's serverConn (initiator) sends msg to bDuplex's clientConn",
		msg:          "foo",
		initiatorA:   true,
		initiatorB:   false,
		whoWrites:    "aDuplex",
		branchConn:   "serverConn",
		expectedMsg:  "foo",
		expectedSize: uint16(3),
		expectedConn: "clientConn",
	},
	{description: "bDuplex's clientConn (initiator) sends msg to aDuplex's serverConn",
		msg:          "foo",
		initiatorA:   false,
		initiatorB:   true,
		whoWrites:    "bDuplex",
		branchConn:   "clientConn",
		expectedMsg:  "foo",
		expectedSize: uint16(3),
		expectedConn: "serverConn",
	},
	{description: "bDuplex's serverConn (initiator) sends msg to aDuplex's clientConn",
		msg:          "foo",
		initiatorA:   false,
		initiatorB:   true,
		whoWrites:    "bDuplex",
		branchConn:   "serverConn",
		expectedMsg:  "foo",
		expectedSize: uint16(3),
		expectedConn: "clientConn",
	},
	{description: "aDuplex's clientConn sends msg to bDuplex's serverConn (initiator)",
		msg:          "bar",
		initiatorA:   false,
		initiatorB:   true,
		whoWrites:    "aDuplex",
		branchConn:   "clientConn",
		expectedMsg:  "bar",
		expectedSize: uint16(3),
		expectedConn: "serverConn",
	},
	{description: "aDuplex's serverConn sends msg to bDuplex's clientConn (initiator)",
		msg:          "bar",
		initiatorA:   false,
		initiatorB:   true,
		whoWrites:    "aDuplex",
		branchConn:   "serverConn",
		expectedMsg:  "bar",
		expectedSize: uint16(3),
		expectedConn: "clientConn",
	},
	{description: "bDuplex's clientConn sends msg to aDuplex's serverConn (initiator)",
		msg:          "bar",
		initiatorA:   false,
		initiatorB:   true,
		whoWrites:    "bDuplex",
		branchConn:   "clientConn",
		expectedMsg:  "bar",
		expectedSize: uint16(3),
		expectedConn: "serverConn",
	},
	{description: "bDuplex's serverConn sends msg to aDuplex's clientConn (initiator)",
		msg:          "bar",
		initiatorA:   false,
		initiatorB:   true,
		whoWrites:    "bDuplex",
		branchConn:   "serverConn",
		expectedMsg:  "bar",
		expectedSize: uint16(3),
		expectedConn: "clientConn",
	},
	{description: "aDuplex's serverConn (initiator) sends 10 bytes msg to bDuplex's clientConn",
		msg:          "Helloworld",
		initiatorA:   true,
		initiatorB:   false,
		whoWrites:    "aDuplex",
		branchConn:   "clientConn",
		expectedMsg:  "Helloworld",
		expectedSize: uint16(10),
		expectedConn: "serverConn",
	},
	// {description: "aDuplex's serverConn (initiator) sends no msg to bDuplex's clientConn",
	// 	msg:          "",
	// 	initiatorA:   true,
	// 	initiatorB:   false,
	// 	whoWrites:    "aDuplex",
	// 	branchConn:   "clientConn",
	// 	expectedMsg:  "",
	// 	expectedSize: uint16(0),
	// 	expectedConn: "serverConn",
	// },
}

func TestNewRPCDuplex(t *testing.T) {
	for _, tt := range tables {
		t.Run(tt.description, func(t *testing.T) {

			assert := assert.New(t)
			connA, connB := net.Pipe()
			defer connA.Close()
			defer connB.Close()

			// connA.SetReadDeadline(time.Now().Add(time.Second * 3))
			connA.SetDeadline(time.Now().Add(time.Second * 3))
			connB.SetDeadline(time.Now().Add(time.Second * 3))

			// Create two instances of RPCDuplex
			aDuplex := NewRPCDuplex(connA, tt.initiatorA)
			bDuplex := NewRPCDuplex(connB, tt.initiatorB)

			// Assign to variables as to who is writing the msg and who is reading the msg
			msgReader, msgWriter := whoWrites(tt.whoWrites, aDuplex, bDuplex)

			// Assign to pcWriter as to whether writer is writing from clientConn or serverConn
			pcWriter := fromWhichBranch(tt.branchConn, msgWriter)

			msg := []byte(tt.msg)

			// TO DO: Send multiple packets instead of one...
			// TO DO: Fix prefixedConn's read method... review logic for prefixedConn.Read(), RPCDuplex.ReadHeader() and RPCDuplex.Forward()
			// Look at Len_read_writer.go and link.go as examples.
			go func() {

				n, err := pcWriter.Write([]byte(msg))
				if err != nil {
					log.Fatalln("Error writing from conn", err)
				}

				assert.Equal(tt.expectedSize, uint16(n), "length of message written should be equal")

			}()

			// TO DO: FIX RACE CONDITION -  go test -race -v -run NewRPCDuplex
			time.Sleep(time.Millisecond * 1)

			prefix, size := msgReader.ReadHeader()

			assert.Equal(tt.expectedSize, size, "length of message read from header should be equal")

			pc, bs := msgReader.Forward(prefix, size)

			assert.Equal(tt.expectedConn, pc.name, "sent to incorrect branchConn")

			n, err := pc.Read(bs)

			assert.Nil(err)
			assert.Equal(tt.expectedSize, uint16(n), "length of message read from prefixedConn.Read() should be equal")
			assert.Equal(tt.expectedMsg, string(bs), "message content should be equal")

		})
	}
}
