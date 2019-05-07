package netutil

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"sync"
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
	{description: "bDuplex's serverConn sends 10 bytes msg to aDuplex's clientConn (initiator)",
		msg:          "helloworld",
		initiatorA:   false,
		initiatorB:   true,
		whoWrites:    "bDuplex",
		branchConn:   "serverConn",
		expectedMsg:  "helloworld",
		expectedSize: uint16(10),
		expectedConn: "clientConn",
	},
	{description: "bDuplex's serverConn sends 20 bytes msg to aDuplex's clientConn (initiator)",
		msg:          "helloworld. Skycoin is best coin!",
		initiatorA:   false,
		initiatorB:   true,
		whoWrites:    "bDuplex",
		branchConn:   "serverConn",
		expectedMsg:  "helloworld. Skycoin is best coin!",
		expectedSize: uint16(33),
		expectedConn: "clientConn",
	},
	// {description: "aDuplex's serverConn (initiator) sends empty string to bDuplex's clientConn",
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
			bs := make([]byte, 256)

			connA, connB := net.Pipe()

			connA.SetDeadline(time.Now().Add(time.Second * 10))
			connB.SetDeadline(time.Now().Add(time.Second * 10))

			// Create two instances of RPCDuplex
			aDuplex := NewRPCDuplex(connA, tt.initiatorA)
			bDuplex := NewRPCDuplex(connB, tt.initiatorB)

			// Determine who is writing the msg and who is reading the msg from test input
			msgReader, msgWriter := whoWrites(tt.whoWrites, aDuplex, bDuplex)

			// Determine who is writing and reading from clientConn or serverConn from test input
			pcWriter, pcReader := fromWhichBranch(tt.branchConn, msgWriter, msgReader)

			msg := []byte(tt.msg)

			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				n, err := pcWriter.Write([]byte(msg))
				assert.Nil(err)
				assert.Equal(tt.expectedSize, uint16(n), "length of message written should be equal")
				wg.Done()
			}()

			// Read prefix and forward to the appropriate channel
			go func() {
				err := msgReader.Serve()
				assert.Nil(err)
			}()

			// Read from one of the PrefixedConn; either serverConn or clientConn
			n, err := pcReader.Read(bs)

			// Close channel
			wg.Wait()
			close(pcReader.readChan)
			close(pcWriter.readChan)

			assert.Nil(err)
			assert.Equal(tt.expectedConn, pcReader.name, "msg forwarded to wrong channel")
			assert.Equal(tt.expectedSize, uint16(n), "length of message read from prefixedConn.Read() should be equal")
			assert.Equal(tt.expectedMsg, string(bs[:n]), "message content should be equal")
		})
	}
}

// Test sending multiple messages in a single connection and recieving them
// in the appropriate branchConn.
// Test Case: aDuplex's clientConn sends n consecutive message to bDuplex's serverConn
func TestNewRPCDuplex_MultipleMessages(t *testing.T) {

	bs := make([]byte, 256)
	assert := assert.New(t)
	expectedMsgCount := 10000

	connA, connB := net.Pipe()

	aDuplex := NewRPCDuplex(connA, true)
	bDuplex := NewRPCDuplex(connB, false)

	var wg sync.WaitGroup
	wg.Add(expectedMsgCount)
	go func() {
		for i := 0; i < expectedMsgCount; i++ {
			msg := fmt.Sprintf("foo%d", i)
			n, err := aDuplex.clientConn.Write([]byte(msg))
			// log.Println(msg, n)
			assert.Nil(err)
			assert.Equal(len(msg), n)
			wg.Done()
			time.Sleep(time.Nanosecond * 250)
		}
	}()

	// Read prefix and forward to the appropriate channel
	go func() {
		err := bDuplex.Serve()
		assert.Nil(err)
	}()

	// Read all the message sent to bDuplex's serverConn
	for i := 0; i < expectedMsgCount; i++ {
		n, err := bDuplex.serverConn.Read(bs)
		// log.Println(string(bs[:n]), n)
		assert.Nil(err)
		assert.Equal("serverConn", bDuplex.serverConn.name, "msg forwarded to wrong channel")
		assert.Equal(len(fmt.Sprintf("foo%d", i)), n, "message content should be equal")
		assert.Equal(fmt.Sprintf("foo%d", i), string(bs[:n]), "message content should be equal")
	}

	// Close channel
	wg.Wait()
	close(aDuplex.clientConn.readChan)
	close(bDuplex.serverConn.readChan)
}

// TestRPCDuplex_Forward forwards one packet Original conn to PrefixedConn
// based on the packet's prefix
func TestRPCDuplex_Forward(t *testing.T) {

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
	}()

	err := bDuplex.Forward()

	bs := <-bDuplex.serverConn.readChan

	require.NoError(t, err)
	assert.Equal(t, []byte("foo"), bs)
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
// data stream and appends it with a 1 byte prefix and
// 2 byte encoded length of the packet.
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
	assert.Equal(t, 6, len(msg))
	assert.Equal(t, string([]byte("\x00\x00\x03foo")), string(msg))
}

// TestReadHeader reads the first-three bytes of the data and
// returns the prefix and size of the packet
func TestRPCDuplex_ReadHeader(t *testing.T) {

	// Case Tested: aDuplex's clientConn is the initiator and has
	// a prefix of 0 and wrote a msg "foo" with size 3
	// 1) prefix: Want(0) -- Got(0)
	// 1) size: Want(3) -- Got(3)
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

		prefix, size := bDuplex.ReadHeader()

		assert.Equal(t, byte(0), prefix)
		assert.Equal(t, uint16(3), size)
	})

	// Case Tested: aDuplex's serverConn is the initiator and has
	// a prefix of 1 and wrote a msg "hello" of size 5
	// 1) prefix: Want(1) -- Got(1)
	// 2) size: Want(5) -- Got(5)
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

		prefix, size := bDuplex.ReadHeader()

		assert.Equal(t, byte(1), prefix)
		assert.Equal(t, uint16(5), size)
	})
}
