package netutil

import (
	"fmt"
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
			rpcSrvA := rpc.NewServer()
			rpcSrvB := rpc.NewServer()

			aDuplex := NewRPCDuplex(connA, rpcSrvA, true, false)
			bDuplex := NewRPCDuplex(connB, rpcSrvB, false, false)
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
			errChA := make(chan error)
			go func() { errChA <- aDuplex.Serve() }()
			close(errChA)

			errChB := make(chan error)
			go func() { errChB <- bDuplex.Serve() }()
			close(errChB)

			// Loop through write channel
			for i := range chWrite {
				// Read from one of the branchConn; either serverConn or clientConn
				n, err := bcReader.Read(b)

				// Send struct to chRead
				chRead <- chReadWrite{i: i.i, n: n, err: err, msg: string(b[:n])}

				// Assert variables from chWrite
				log.Println("chWrite:", i)
				assert.Nil(i.err)
				assert.Equal(len(i.msg), i.n, "message length written should be equal")
				assert.Equal(fmt.Sprintf(tt.expectedMsg, i.i), i.msg, "message content written should be equal")
			}
			close(chRead)

			// Assert variables from chRead
			for i := range chRead {
				log.Println("chRead: ", i)
				assert.Nil(i.err)
				assert.Equal(len(i.msg), i.n, "message length read from branchConn.Read() should be equal")
				assert.Equal(fmt.Sprintf(tt.expectedMsg, i.i), i.msg, "message content read from branchConn.Read() should be equal")
			}

			// Assert error from msgReader.Serve()
			assert.Nil(<-errChA)
			assert.Nil(<-errChB)

			// Close channel
			close(bcReader.readCh)
			close(bcWriter.readCh)
		})
	}
}

// TestReadHeader reads the first-three bytes of the data and
// returns the prefix and size of the packet
func TestRPCDuplex_ReadHeader(t *testing.T) {

	// Case Tested: aDuplex's clientConn is the initiator and has
	// a prefix of 0 and wrote a msg "foo" with size 3
	// 1) prefix: Want(0) -- Got(0)
	// 1) size: Want(3) -- Got(3)
	t.Run("successfully_read_prefix_from_clientConn_to_serverConn", func(t *testing.T) {
		connA, connB := net.Pipe()

		rpcSrvA := rpc.NewServer()
		rpcSrvB := rpc.NewServer()

		aDuplex := NewRPCDuplex(connA, rpcSrvA, true, false)
		bDuplex := NewRPCDuplex(connB, rpcSrvB, false, false)

		msg := []byte("foo")

		errCh := make(chan error)
		go func() {
			_, err := aDuplex.clientConn.Write(msg)
			aDuplex.conn.Close()
			errCh <- err
		}()

		prefix, size := bDuplex.readHeader()

		assert.Equal(t, byte(0), prefix)
		assert.Equal(t, uint16(3), size)

		// Read packets so that channel doesn't block
		b := make([]byte, size)
		n, err := bDuplex.conn.Read(b)
		require.NoError(t, err)
		require.NoError(t, <-errCh)
		assert.Equal(t, 3, n)
		assert.Equal(t, string([]byte("foo")), string(b))

	})

	// Case Tested: aDuplex's serverConn is the initiator and has
	// a prefix of 1 and wrote a msg "hello" of size 5
	// 1) prefix: Want(1) -- Got(1)
	// 2) size: Want(5) -- Got(5)
	t.Run("successfully_read_prefix_from_serverConn_to_clientConn", func(t *testing.T) {
		connA, connB := net.Pipe()

		rpcSrvA := rpc.NewServer()
		rpcSrvB := rpc.NewServer()

		aDuplex := NewRPCDuplex(connA, rpcSrvA, true, false)
		bDuplex := NewRPCDuplex(connB, rpcSrvB, false, false)

		msg := []byte("hello")

		errCh := make(chan error)
		go func() {
			_, err := aDuplex.serverConn.Write(msg)
			aDuplex.conn.Close()
			errCh <- err
		}()

		prefix, size := bDuplex.readHeader()

		assert.Equal(t, byte(1), prefix)
		assert.Equal(t, uint16(5), size)

		// Read packets so that channel doesn't block
		b := make([]byte, size)
		n, err := bDuplex.conn.Read(b)
		require.NoError(t, err)
		require.NoError(t, <-errCh)
		assert.Equal(t, 5, n)
		assert.Equal(t, string([]byte("hello")), string(b))
	})
}

// TestRPCDuplex_Forward forwards one packet Original conn to branchConn
// based on the packet's prefix
func TestRPCDuplex_Forward(t *testing.T) {

	connA, connB := net.Pipe()
	defer connB.Close()

	rpcSrvA := rpc.NewServer()
	rpcSrvB := rpc.NewServer()

	aDuplex := NewRPCDuplex(connA, rpcSrvA, true, false)
	bDuplex := NewRPCDuplex(connB, rpcSrvB, false, false)

	msg := []byte("foo")
	errCh := make(chan error, 1)
	go func() {
		_, err := aDuplex.clientConn.Write(msg)
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

func TestRPCDuplex_Serve(t *testing.T) {
	cA, cB := net.Pipe()

	serverA := rpc.NewServer()
	require.NoError(t, serverA.RegisterName("RPC", new(RPC)))

	serverB := rpc.NewServer()
	require.NoError(t, serverB.RegisterName("RPC", new(RPC)))

	errChA := make(chan error)
	dA := NewRPCDuplex(cA, serverA, true, true)
	go func() { errChA <- dA.Serve() }()

	errChB := make(chan error)
	dB := NewRPCDuplex(cB, serverB, false, true)
	go func() { errChB <- dB.Serve() }()

	var r int
	for i := 0; i < 10; i++ {
		require.NoError(t, dA.Client().Call("RPC.Double", i, &r))
		require.Equal(t, i*2, r)
		log.Println("aDuplex:", r)

		require.NoError(t, dB.Client().Call("RPC.Double", i, &r))
		require.Equal(t, i*2, r)
		log.Println("bDuplex:", r)
	}
}

// TestbranchConn_Read reads data pushed in by
// the Original connection
func TestBranchConn_Read(t *testing.T) {

	t.Run("successful_branchConn_read", func(t *testing.T) {

		ch := make(chan []byte)
		bc := &branchConn{prefix: 0, readCh: ch}
		msg := []byte("foo")

		go func() {
			bc.readCh <- msg
		}()

		b := make([]byte, 3)
		n, err := bc.Read(b)

		require.NoError(t, err)
		assert.Equal(t, 3, n)
		assert.Equal(t, []byte("foo"), b)
	})

	t.Run("empty_branchConn_read", func(t *testing.T) {

		ch := make(chan []byte)
		bc := &branchConn{prefix: 0, readCh: ch}
		msg := []byte("")

		go func() {
			bc.readCh <- msg
		}()

		var b []byte
		n, err := bc.Read(b)

		require.NoError(t, err)
		assert.Equal(t, 0, n)
		assert.Equal(t, []byte(nil), b)
	})
}

// TestbranchConn_Writes writes len(p) bytes from p to
// data stream and appends it with a 1 byte prefix and
// 2 byte encoded length of the packet.
func TestBranchConn_Write(t *testing.T) {

	connA, connB := net.Pipe()
	var ch chan []byte
	defer connB.Close()

	bc := &branchConn{Conn: connA, readCh: ch}

	nCh := make(chan int, 1)
	errCh := make(chan error, 1)
	go func() {
		n, err := bc.Write([]byte("foo"))
		bc.Conn.Close()
		nCh <- n
		errCh <- err
	}()

	msg, err := ioutil.ReadAll(connB)

	assert.Equal(t, 3, <-nCh)
	require.NoError(t, <-errCh)
	require.NoError(t, err)
	assert.Equal(t, 6, len(msg))
	assert.Equal(t, string([]byte("\x00\x00\x03foo")), string(msg))
}
