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

// whoWrites determines which Duplex writes the msg and which one is reading the msg from test input
func whoWrites(whoWrites string, aDuplex *RPCDuplex, bDuplex *RPCDuplex) (*RPCDuplex, *RPCDuplex) {

	if whoWrites == "aDuplex" {
		msgWriter := aDuplex
		msgReader := bDuplex
		return msgReader, msgWriter
	}

	return bDuplex, aDuplex
}

// fromWhichBranch determines which branch writes the msg and which one is reading the msg from test input
func fromWhichBranch(branchConn string, msgWriter *RPCDuplex, msgReader *RPCDuplex) (*PrefixedConn, *PrefixedConn) {
	if branchConn == "clientConn" {
		return msgWriter.clientConn, msgReader.serverConn
	}
	return msgWriter.serverConn, msgReader.clientConn
}

var tables = []struct {
	// Inputs
	description string
	msg         string
	initiatorA  bool
	initiatorB  bool
	whoWrites   string
	branchConn  string

	// Expected Result
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
			bs := make([]byte, 20)

			connA, connB := net.Pipe()
			defer connA.Close()
			defer connB.Close()

			connA.SetDeadline(time.Now().Add(time.Second * 3))
			connB.SetDeadline(time.Now().Add(time.Second * 3))

			// Create two instances of RPCDuplex
			aDuplex := NewRPCDuplex(connA, tt.initiatorA)
			bDuplex := NewRPCDuplex(connB, tt.initiatorB)

			// Determine who is writing the msg and who is reading the msg from test input
			msgReader, msgWriter := whoWrites(tt.whoWrites, aDuplex, bDuplex)

			// Determine who is writing and reading from clientConn or serverConn from test input
			pcWriter, pcReader := fromWhichBranch(tt.branchConn, msgWriter, msgReader)

			msg := []byte(tt.msg)

			go func() {

				n, err := pcWriter.Write([]byte(msg))
				if err != nil {
					log.Fatalln("Error writing from conn", err)
				}

				assert.Equal(tt.expectedSize, uint16(n), "length of message written should be equal")

			}()

			time.Sleep(time.Millisecond * 1)

			msgReader.Serve()
			n, err := pcReader.Read(bs)

			// Trim Null characters after bs
			bs = bytes.Trim(bs, "\x00")

			assert.Nil(err)
			assert.Equal(tt.expectedConn, pcReader.name, "msg forwarded to wrong channel")
			assert.Equal(tt.expectedSize, uint16(n), "length of message read from prefixedConn.Read() should be equal")
			assert.Equal(tt.expectedMsg, string(bs), "message content should be equal")

		})
	}
}

// TestPrefixedConn_Read reads data from the buffer pushed in by
// the Original connection
func TestPrefixedConn_Read(t *testing.T) {

	// Want(3, "foo") - Get(3, "foo")
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

	// Want(0, nil) - Get(0, nil)
	t.Run("len(bs) is zero prefixedConn read", func(t *testing.T) {
		var c io.Writer
		var readBuf bytes.Buffer

		pc := &PrefixedConn{prefix: 0, writeConn: c, readBuf: readBuf}

		var bs []byte
		n, err := pc.Read(bs)

		require.NoError(t, err)
		assert.Equal(t, 0, n)
		assert.Equal(t, []byte(nil), bs)
	})

	// // Want(io.EOF; loop forever) - Get(io.EOF; loop forever)
	// t.Run("empty buffer prefixedConn read", func(t *testing.T) {
	// 	var c io.Writer
	// 	var readBuf bytes.Buffer

	// 	pc := &PrefixedConn{prefix: 0, writeConn: c, readBuf: readBuf}

	// 	bs := make([]byte, 3)
	// 	n, err := pc.Read(bs)

	// 	require.NoError(t, err)
	// 	assert.Equal(t, 0, n)
	// })

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
func TestRPCDuplex_ReadHeader(t *testing.T) {

	// Case Tested: aDuplex's clientConn is the initiator and has
	// a prefix of 0 and wrote a msg "foo" with byte size of 3.
	// 1) Prefix: Want(0) -- Got(0)
	// 2) size: Want(3) -- Got(3)
	t.Run("successfully read prefix and size from clientConn to serverConn", func(t *testing.T) {
		connA, connB := net.Pipe()

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

		assert.Equal(t, byte(0), prefix)
		assert.Equal(t, uint16(len(msg)), size)
	})

	// Case Tested: aDuplex's serverConn is the initiator and has
	// a prefix of 1 and wrote a msg "hello" with byte size of 5.
	// 1) Prefix: Want(1) -- Got(1)
	// 2) size: Want(5) -- Got(5)
	t.Run("successfully read prefix and size from serverConn to clientConn", func(t *testing.T) {
		connA, connB := net.Pipe()

		aDuplex := NewRPCDuplex(connA, true)
		bDuplex := NewRPCDuplex(connB, false)

		msg := []byte("hello")

		go func() {
			_, err := aDuplex.serverConn.Write(msg)
			if err != nil {
				log.Fatalln("Error writing from conn", err)
			}
		}()

		prefix, size := bDuplex.ReadHeader()

		assert.Equal(t, byte(1), prefix)
		assert.Equal(t, uint16(5), size)
	})
}

// TestRPCDuplex_Forward forwards one packet from Original conn to
// PrefixedConn given the prefix and size of payload. If reading from
// Original Conn returns an err; that err is returned.
func TestRPCDuplex_Forward(t *testing.T) {

	connA, connB := net.Pipe()

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
	err := bDuplex.Forward(prefix, size)
	require.NoError(t, err)
}

// TestRPCDuplex_Serve continuously calls forward in a loop alongside with ReadHeader.
// Case Tested: aDuplex's clientConn writes a msg to bDuplex's serverConn
func TestRPCDuplex_Serve(t *testing.T) {

	t.Run("successfully serve and read once from prefixedConn", func(t *testing.T) {
		bs := make([]byte, 3)

		connA, connB := net.Pipe()

		aDuplex := NewRPCDuplex(connA, true)
		bDuplex := NewRPCDuplex(connB, false)

		msg := []byte("foo")
		go func() {
			_, err := aDuplex.clientConn.Write(msg)
			if err != nil {
				log.Fatalln("Error writing from conn", err)
			}
		}()

		bDuplex.Serve()
		n, err := bDuplex.serverConn.Read(bs)

		require.NoError(t, err)
		assert.Equal(t, 3, n)
		assert.Equal(t, "foo", string(bs))
	})

	t.Run("successfully serve and read multiple times from prefixedConn", func(t *testing.T) {
		bs := make([]byte, 3)

		connA, connB := net.Pipe()

		aDuplex := NewRPCDuplex(connA, true)
		bDuplex := NewRPCDuplex(connB, false)

		msg := []byte("fooBar")
		go func() {
			_, err := aDuplex.clientConn.Write(msg)
			if err != nil {
				log.Fatalln("Error writing from conn", err)
			}
		}()

		bDuplex.Serve()
		// Read once from buffer
		n, err := bDuplex.serverConn.Read(bs)

		require.NoError(t, err)
		assert.Equal(t, 3, n)
		assert.Equal(t, "foo", string(bs))

		// Read remaining from buffer
		n, err = bDuplex.serverConn.Read(bs)
		require.NoError(t, err)
		assert.Equal(t, 3, n)
		assert.Equal(t, "Bar", string(bs))
	})

	// t.Run("successfully serve and read forever from prefixedConn", func(t *testing.T) {
	// 	bs := make([]byte, 3)

	// 	connA, connB := net.Pipe()

	// 	aDuplex := NewRPCDuplex(connA, true)
	// 	bDuplex := NewRPCDuplex(connB, false)

	// 	msg := []byte("fooBar")
	// 	go func() {
	// 		_, err := aDuplex.clientConn.Write(msg)
	// 		if err != nil {
	// 			log.Fatalln("Error writing from conn", err)
	// 		}
	// 	}()

	// 	time.Sleep(time.Millisecond * 1)

	// 	go func() {
	// 		_, err := aDuplex.clientConn.Write([]byte("hi"))
	// 		if err != nil {
	// 			log.Println(nil)
	// 		}
	// 	}()

	// 	time.Sleep(time.Millisecond * 1)

	// 	bDuplex.Serve()

	// 	for {
	// 		bDuplex.serverConn.Read(bs)
	// 		log.Println(string(bs))
	// 	}
	// })

}

// TO DO: FIX RACE CONDITION -  go test -race -v -run NewRPCDuplex
// time.Sleep(time.Millisecond * 1)

// for i := 0; i < expectedMsgCount; i++ {
// 	msg := fmt.Sprintf("Hello world %d!", i)
// 	_, err := initConn.Send(1, []byte(msg))
// 	require.NoError(t, err)
// }
