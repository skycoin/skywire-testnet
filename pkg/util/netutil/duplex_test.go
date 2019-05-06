package netutil

import (
	"bytes"
	"io/ioutil"
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
	{description: "aDuplex's serverConn (initiator) sends empty string to bDuplex's clientConn",
		msg:          "",
		initiatorA:   true,
		initiatorB:   false,
		whoWrites:    "aDuplex",
		branchConn:   "clientConn",
		expectedMsg:  "",
		expectedSize: uint16(0),
		expectedConn: "serverConn",
	},
}

func TestNewRPCDuplex(t *testing.T) {
	for _, tt := range tables {
		t.Run(tt.description, func(t *testing.T) {

			assert := assert.New(t)
			bs := make([]byte, 10)

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
				pcWriter.Close()
			}()

			err := msgReader.Forward()
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

// TestRPCDuplex_Forward forwards one packet Original conn to PrefixedConn based on the packet's prefix
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
		aDuplex.conn.Close()
	}()

	err := bDuplex.Forward()

	bs := <-bDuplex.serverConn.readChan
	bs = bytes.Trim(bs, "\x00")
	require.NoError(t, err)
	assert.Equal(t, []byte("foo"), bs)

	bDuplex.conn.Close()
}

// TestPrefixedConn_Read reads data pushed in by
// the Original connection
func TestPrefixedConn_Read(t *testing.T) {

	t.Run("successful prefixedConn read", func(t *testing.T) {

		ch := make(chan []byte)
		pc := &PrefixedConn{prefix: 0, readChan: ch}

		msg := []byte("foo")

		go func() {
			pc.readChan <- msg
		}()

		bs := make([]byte, 3)
		n, err := pc.Read(bs)

		require.NoError(t, err)
		assert.Equal(t, 3, n)
		assert.Equal(t, []byte("foo"), bs)
	})

	t.Run("empty prefixedConn read", func(t *testing.T) {

		ch := make(chan []byte)
		pc := &PrefixedConn{prefix: 0, readChan: ch}

		msg := []byte("")

		go func() {
			pc.readChan <- msg
		}()

		var bs []byte
		n, err := pc.Read(bs)

		require.NoError(t, err)
		assert.Equal(t, 0, n)
		assert.Equal(t, []byte(nil), bs)
	})
}

// TestPrefixedConn_Writes writes len(p) bytes from p to
// data stream and appends it with a prefix of 1 byte
func TestPrefixedConn_Write(t *testing.T) {

	connA, connB := net.Pipe()
	var ch chan []byte
	defer connB.Close()

	pc := &PrefixedConn{Conn: connA, readChan: ch}

	go func() {
		n, err := pc.Write([]byte("foo"))
		require.NoError(t, err)
		assert.Equal(t, 3, n)
		connA.Close()
	}()

	msg, err := ioutil.ReadAll(connB)
	require.NoError(t, err)
	assert.Equal(t, 4, len(msg))
	assert.Equal(t, string([]byte("\x00foo")), string(msg))
}

// TestReadHeader reads the first bytes of the data and
// returns the prefix of the packet
func TestRPCDuplex_ReadHeader(t *testing.T) {

	// Case Tested: aDuplex's clientConn is the initiator and has
	// a prefix of 0 and wrote a msg "foo".
	// 1) Prefix: Want(0) -- Got(0)
	t.Run("successfully read prefix from clientConn to serverConn", func(t *testing.T) {
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

		prefix := bDuplex.ReadHeader()

		assert.Equal(t, byte(0), prefix)
	})

	// Case Tested: aDuplex's serverConn is the initiator and has
	// a prefix of 1 and wrote a msg "hello".
	// 1) Prefix: Want(1) -- Got(1)
	t.Run("successfully read prefix from serverConn to clientConn", func(t *testing.T) {
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

		prefix := bDuplex.ReadHeader()
		assert.Equal(t, byte(1), prefix)
	})
}

// func TestRandomTest(t *testing.T) {

// 	connA, connB := net.Pipe()

// 	aDuplex := NewRPCDuplex(connA, true)
// 	bDuplex := NewRPCDuplex(connB, false)

// 	msg := []byte("aaaafoo")
// 	go func() {
// 		_, err := aDuplex.clientConn.Write(msg)
// 		// fmt.Fprintf(aDuplex.clientConn, "Bar")
// 		if err != nil {
// 			log.Fatalln("Error writing from conn", err)
// 		}

// 		// aDuplex.conn.Close()
// 	}()

// 	for {
// 		err := bDuplex.Forward()
// 		bs := <-bDuplex.serverConn.readChan
// 		bs = bytes.Trim(bs, "\x00")

// 		log.Println(string(bs))

// 		assert.Nil(t, err)
// 		assert.Equal(t, []byte("fooBaar"), bs)
// 		assert.Equal(t, 6, len(bs))
// 	}

// ======================

// // errCh := make(chan error)
// ch := make(chan []byte)

// go func() {
// 	for i := 0; i < 2; i++ {
// 		log.Println("Code got here")
// 		// err := bDuplex.Forward()
// 		bDuplex.Forward()
// 		bs := <-bDuplex.serverConn.readChan
// 		bs = bytes.Trim(bs, "\x00")

// 		// errCh <- err
// 		ch <- bs
// 		log.Println(string(bs))
// 		log.Println("Code finish")
// 	}
// 	close(ch)
// 	// close(errCh)
// }()

// for bs := range ch {
// 	assert.Equal(t, []byte("afoo"), bs)
// }

// for err := range errCh {
// 	assert.Nil(t, err)
// }

// }
