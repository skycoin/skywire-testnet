package netutil

import (
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/rpc"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type chReadWrite struct {
	i   int
	n   int
	err error
	msg string
}

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
func fromWhichBranch(branchConn string, msgWriter *RPCDuplex, msgReader *RPCDuplex) (*branchConn, *branchConn) {
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
	expectedMsg string
}{
	{description: "aDuplex's_clientConn_(initiator)_sends_multiple_msg_to_bDuplex's_serverConn",
		msg:         "foo%d",
		initiatorA:  true,
		initiatorB:  false,
		whoWrites:   "aDuplex",
		branchConn:  "clientConn",
		expectedMsg: "foo%d",
	},
	{description: "aDuplex's_serverConn_(initiator)_sends_multiple_msg_to_bDuplex's_clientConn",
		msg:         "foo%d",
		initiatorA:  true,
		initiatorB:  false,
		whoWrites:   "aDuplex",
		branchConn:  "serverConn",
		expectedMsg: "foo%d",
	},
	{description: "bDuplex's_clientConn_(initiator)_sends_multiple_msg_to_aDuplex's_serverConn",
		msg:         "foo%d",
		initiatorA:  false,
		initiatorB:  true,
		whoWrites:   "bDuplex",
		branchConn:  "clientConn",
		expectedMsg: "foo%d",
	},
	{description: "bDuplex's_serverConn_(initiator)_sends_multiple_msg_to_aDuplex's_clientConn",
		msg:         "foo%d",
		initiatorA:  false,
		initiatorB:  true,
		whoWrites:   "bDuplex",
		branchConn:  "serverConn",
		expectedMsg: "foo%d",
	},
	{description: "aDuplex's_clientConn_sends_multiple_msg_to_bDuplex's_serverConn_(initiator)",
		msg:         "bar%d",
		initiatorA:  false,
		initiatorB:  true,
		whoWrites:   "aDuplex",
		branchConn:  "clientConn",
		expectedMsg: "bar%d",
	},
	{description: "aDuplex's_serverConn_sends_multiple_msg_to_bDuplex's_clientConn_(initiator)",
		msg:         "bar%d",
		initiatorA:  false,
		initiatorB:  true,
		whoWrites:   "aDuplex",
		branchConn:  "serverConn",
		expectedMsg: "bar%d",
	},
	{description: "bDuplex's_clientConn_sends_multiple_msg_to_aDuplex's_serverConn_(initiator)",
		msg:         "bar%d",
		initiatorA:  false,
		initiatorB:  true,
		whoWrites:   "bDuplex",
		branchConn:  "clientConn",
		expectedMsg: "bar%d",
	},
	{description: "bDuplex's_serverConn_sends_multiple_msg_to_aDuplex's_clientConn_(initiator)",
		msg:         "bar%d",
		initiatorA:  false,
		initiatorB:  true,
		whoWrites:   "bDuplex",
		branchConn:  "serverConn",
		expectedMsg: "bar%d",
	},
	{description: "bDuplex's_serverConn_sends_multiple_10_bytes_msg_to_aDuplex's_clientConn_(initiator)",
		msg:         "helloworld%d",
		initiatorA:  false,
		initiatorB:  true,
		whoWrites:   "bDuplex",
		branchConn:  "serverConn",
		expectedMsg: "helloworld%d",
	},
	{description: "bDuplex's_serverConn_sends_multiple_20_bytes_msg_to_aDuplex's_clientConn_(initiator)",
		msg:         "helloworld. Skycoin is best coin!%d",
		initiatorA:  false,
		initiatorB:  true,
		whoWrites:   "bDuplex",
		branchConn:  "serverConn",
		expectedMsg: "helloworld. Skycoin is best coin!%d",
	},
}

// Test sending multiple messages in a single connection and receiving them
// in the appropriate branchConn with multiple test cases
func TestNewRPCDuplex(t *testing.T) {
	for _, tt := range tables {
		t.Run(tt.description, func(t *testing.T) {

			b := make([]byte, 256)
			assert := assert.New(t)
			expectedMsgCount := 10
			chWrite := make(chan chReadWrite, expectedMsgCount)
			chRead := make(chan chReadWrite, expectedMsgCount)

			connA, connB := net.Pipe()

			assert.Nil(connA.SetDeadline(time.Now().Add(time.Second * 10)))
			assert.Nil(connB.SetDeadline(time.Now().Add(time.Second * 10)))

			// Create two instances of RPCDuplex and rpcServer
			aDuplex := NewRPCDuplex(connA, nil, true)
			bDuplex := NewRPCDuplex(connB, nil, false)

			// Determine who is writing the msg and who is reading the msg from test input
			msgReader, msgWriter := whoWrites(tt.whoWrites, aDuplex, bDuplex)

			// Determine who is writing and reading from clientConn or serverConn from test input
			bcWriter, bcReader := fromWhichBranch(tt.branchConn, msgWriter, msgReader)

			go func() {
				for i := 0; i < expectedMsgCount; i++ {
					msg := fmt.Sprintf(tt.msg, i)
					n, err := bcWriter.Write([]byte(msg))
					chWrite <- chReadWrite{i: i, n: n, err: err, msg: msg}
				}
				close(chWrite)
			}()

			// Read prefix and forward to the appropriate channel
			errChA := make(chan error, 1)
			go func() {
				for {
					err := aDuplex.forward()
					switch err {
					case nil:
						continue
					case io.EOF:
						errChA <- err
					default:
						errChA <- err
					}
				}
			}()
			close(errChA)

			errChB := make(chan error, 1)
			go func() {
				for {
					err := bDuplex.forward()
					switch err {
					case nil:
						continue
					case io.EOF:
						errChB <- err
					default:
						errChA <- err
					}
				}
			}()
			close(errChB)

			// Loop through write channel
			for i := range chWrite {
				// Read from one of the branchConn; either serverConn or clientConn
				n, err := bcReader.Read(b)

				// Send struct to chRead
				chRead <- chReadWrite{i: i.i, n: n, err: err, msg: string(b[:n])}

				// Assert variables from chWrite
				// log.Println("chWrite:", i)
				assert.Nil(i.err)
				assert.Equal(len(i.msg), i.n, "message length written should be equal")
				assert.Equal(fmt.Sprintf(tt.expectedMsg, i.i), i.msg, "message content written should be equal")
			}
			close(chRead)

			// Assert variables from chRead
			for i := range chRead {
				// log.Println("chRead: ", i)
				assert.Nil(i.err)
				assert.Equal(len(i.msg), i.n, "message length read from branchConn.Read() should be equal")
				assert.Equal(fmt.Sprintf(tt.expectedMsg, i.i), i.msg, "message content read from branchConn.Read() should be equal")
			}

			// Assert error from msgReader.Forward()
			assert.Nil(<-errChA)
			assert.Nil(<-errChB)

			// Close channel
			close(bcReader.readCh)
			close(bcWriter.readCh)
		})
	}
}

// TestRPCDuplex_Forward forwards one packet Original conn to branchConn
// based on the packet's prefix
func TestRPCDuplex_Forward(t *testing.T) {

	connA, connB := net.Pipe()
	defer connB.Close()

	bDuplex := NewRPCDuplex(connB, nil, false)

	msg := []byte("\x00\x00\x03foo")
	errCh := make(chan error, 1)
	go func() {
		_, err := connA.Write(msg)
		connA.Close()
		errCh <- err
	}()

	err := bDuplex.forward()
	require.NoError(t, err)
	assert.Equal(t, []byte("foo"), <-bDuplex.serverConn.readCh)
	require.NoError(t, <-errCh)

}

// RPC is a receiver which we will use Register to publishes the receiver's methods in the DefaultServer.
type RPC struct{}

// RPC methods must look schematically like: func (t *T) MethodName(argType T1, replyType *T2) error
func (RPC) Double(i int, reply *int) error {
	*reply = i * 2
	return nil
}

// TestRPCDuplex_Serve test communication between two RPC Duplex. When both Duplex
// is created and served, both duplex's client should be able to successfully call
// the rpc method 'double' from rpc.server
func TestRPCDuplex_Serve(t *testing.T) {
	cA, cB := net.Pipe()

	serverA := rpc.NewServer()
	require.NoError(t, serverA.RegisterName("RPC", new(RPC)))

	serverB := rpc.NewServer()
	require.NoError(t, serverB.RegisterName("RPC", new(RPC)))

	errChA := make(chan error)
	dA := NewRPCDuplex(cA, serverA, true)
	go func() { errChA <- dA.Serve() }()

	errChB := make(chan error)
	dB := NewRPCDuplex(cB, serverB, false)
	go func() { errChB <- dB.Serve() }()

	var r int
	for i := 0; i < 10; i++ {
		require.NoError(t, dA.Client().Call("RPC.Double", i, &r))
		require.Equal(t, i*2, r)
		// log.Println("aDuplex:", r)

		require.NoError(t, dB.Client().Call("RPC.Double", i, &r))
		require.Equal(t, i*2, r)
		// log.Println("bDuplex:", r)
	}
}

func TestBranchConn_Read(t *testing.T) {

	// Read reads in data from original conn through branchConn's readCh which is
	// a type of chan [].
	t.Run("successful_branchConn_read", func(t *testing.T) {

		ch := make(chan []byte, 1)
		bc := &branchConn{prefix: 0, readCh: ch}
		msg := []byte("foo")

		bc.readCh <- msg

		b := make([]byte, 3)
		n, err := bc.Read(b)

		require.NoError(t, err)
		assert.Equal(t, 3, n)
		assert.Equal(t, []byte("foo"), b)
	})

	// Read reads in empty data from original conn through branchConn's readCh which is
	// a type of chan []. It should return 0 length read and b should be nil
	t.Run("empty_branchConn_read", func(t *testing.T) {

		ch := make(chan []byte, 1)
		bc := &branchConn{prefix: 0, readCh: ch}
		msg := []byte("")

		bc.readCh <- msg

		var b []byte
		n, err := bc.Read(b)

		require.NoError(t, err)
		assert.Equal(t, 0, n)
		assert.Equal(t, []byte(nil), b)
	})
}

var tablesWrite = []struct {
	// Inputs
	description string

	// Expected Result
	lenMsg      int
	prefix      byte
	lenPayload  uint16
	expectedMsg []byte
}{
	// TestbranchConn_Writes writes len(p) bytes from p to
	// data stream and appends it with a 1 byte prefix and
	// 2 byte encoded length of the packet. Thus, length
	// of message should equal to length of payload + 3
	{description: "len(msg)_equal_len(payload)+3",
		lenMsg:      6,
		prefix:      byte(0),
		lenPayload:  uint16(3),
		expectedMsg: []byte("foo"),
	},
	// The first byte of the message is the prefix that
	// will be used to direct packets to their respective
	// branchConn. The prefix is a byte of either 0 or 1.
	{description: "msg[0]_prefix_is_0",
		lenMsg:      6,
		prefix:      byte(0),
		lenPayload:  uint16(3),
		expectedMsg: []byte("foo"),
	},
	// The 2nd and 3rd byte is the encoded length of the payload
	// with the binary package. The decoded length of the payload
	// will be in uint16 and is 3.
	{description: "msg[1:3]_decodes_to_len(payload)",
		lenMsg:      6,
		prefix:      byte(0),
		lenPayload:  uint16(3),
		expectedMsg: []byte("foo"),
	},
	// The bytes after the 3rd byte will be the initial message
	// itself. It should equal to the payload. In this case,
	// the message "foo" should equal to "foo"
	{description: "msg[3:]_equal_to_payload)",
		lenMsg:      6,
		prefix:      byte(0),
		lenPayload:  uint16(3),
		expectedMsg: []byte("foo"),
	},
}

func TestBranchConn_Write(t *testing.T) {

	for _, tt := range tablesWrite {
		t.Run(tt.description, func(t *testing.T) {
			var err error
			var ch chan []byte

			connA, connB := net.Pipe()
			defer connB.Close()

			bc := &branchConn{Conn: connA, readCh: ch}
			payload := "foo"

			done := make(chan struct{})
			defer close(done)

			go func() {
				_, err = bc.Write([]byte(payload))
				bc.Conn.Close()
				done <- struct{}{}
			}()

			msg, err := ioutil.ReadAll(connB)
			if err != nil {
				log.Println(err)
			}

			lenPayload := binary.BigEndian.Uint16(msg[1:3])
			<-done

			assert.NoError(t, err)
			assert.Equal(t, tt.lenMsg, len(payload)+3)
			assert.Equal(t, tt.prefix, msg[0])
			assert.Equal(t, tt.lenPayload, lenPayload)
			assert.Equal(t, tt.expectedMsg, msg[3:])
		})
	}
}
