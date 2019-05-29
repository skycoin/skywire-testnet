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

func TestBranchConn_Read(t *testing.T) {

	// It is expected that a 'branchConn' can Read data that is pushed in it's 'readCh' successfully.
	// In such a scenario, 'branchConn.Read()' should return 'err == nil', 'n == 3' and 'b=[]byte("foo")'.
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

	// It is expected that 'branchConn' handles empty packets (read via 'readCh') correctly.
	// In such a scenario, 'branchConn.Read()' should return 'err == nil' and 'n == 0'.
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

// TestbranchConn_Writes tests for the following:
// Ensure len(msg) is equal to len(payload) + 3.
// Ensure msg[0] is the right prefix.
// Ensure msg[1:3] decodes to len(payload).
// Ensure msg[3:] is equal to payload.
func TestBranchConn_Write(t *testing.T) {

	var tablesWrite = []struct {
		// Inputs
		description string
		payload     string
		inputPrefix byte

		// Expected Result
		lenMsg      int
		prefix      byte
		lenPayload  uint16
		expectedMsg []byte
	}{
		{description: "len(msg)_equal_len(payload)+3",
			payload:     "foo",
			inputPrefix: byte(0),
			lenMsg:      6,
			prefix:      byte(0),
			lenPayload:  uint16(3),
			expectedMsg: []byte("foo"),
		},
		{description: "msg[0]_prefix_is_1",
			payload:     "bar",
			inputPrefix: byte(1),
			lenMsg:      6,
			prefix:      byte(1),
			lenPayload:  uint16(3),
			expectedMsg: []byte("bar"),
		},
		{description: "msg[1:3]_decodes_to_len(payload)",
			payload:     "foobar",
			inputPrefix: byte(0),
			lenMsg:      9,
			prefix:      byte(0),
			lenPayload:  uint16(6),
			expectedMsg: []byte("foobar"),
		},
		{description: "msg[3:]_equal_to_payload)",
			payload:     "helloworld",
			inputPrefix: byte(0),
			lenMsg:      13,
			prefix:      byte(0),
			lenPayload:  uint16(10),
			expectedMsg: []byte("helloworld"),
		},
	}

	for _, tt := range tablesWrite {
		t.Run(tt.description, func(t *testing.T) {
			var err error
			var ch chan []byte

			connA, connB := net.Pipe()
			defer connB.Close()

			bc := &branchConn{Conn: connA, prefix: tt.inputPrefix, readCh: ch}
			payload := tt.payload

			done := make(chan struct{})

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
			assert.Equal(t, tt.lenMsg, len(payload)+3, "len(msg) should equal to len(payload)+3")
			assert.Equal(t, tt.prefix, msg[0], "incorrect prefix")
			assert.Equal(t, tt.lenPayload, lenPayload, "decoded length should equal to length of payload")
			assert.Equal(t, tt.expectedMsg, msg[3:], "message content should be equal")

			close(done)
		})
	}
}

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
	{description: "bDuplex's_serverConn_sends_multiple_20_bytes_msg_to_aDuplex's_clientConn_(initiator)",
		msg:         "helloworld. Skycoin is best coin!%d",
		initiatorA:  false,
		initiatorB:  true,
		whoWrites:   "bDuplex",
		branchConn:  "serverConn",
		expectedMsg: "helloworld. Skycoin is best coin!%d",
	},
}

// TestNewRPCDuplex test communication between two RPC Duplexes' branch connections. When both Duplexes are created and
// duplex.forward() is called, it is expected that multiple packets written by one of the duplex's branchConn
// will be read and matched in terms of length and content by another end of the second Duplex's branhConn.
// e.g (aDuplex's clientConn --> bDuplex's serverConn or bDuplex's serverConn --> aDuplex's clientConn)
//
// Example: Given that aDuplex's clientConn writes 'foo' five times, it is expected that the msg 'foo' with length of 3
// be forwarded correctly to bDuplex's serverConn all five times.
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
					if err := aDuplex.forward(); err != nil {
						if err == io.EOF {
							err = nil
						}
						errChA <- err
					}
				}
			}()
			close(errChA)

			errChB := make(chan error, 1)
			go func() {
				for {
					if err := bDuplex.forward(); err != nil {
						if err == io.EOF {
							err = nil
						}
						errChB <- err
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

		})
	}
}

// Given a prefixed packet, it is expected that duplex.forward() will decoded the prefixed packet,read in the data,
// and forward the packet to the correct branchConn. In such a scenario, it is expected that the msg []byte("foo")
// is push to bDuplex's serverConn's 'readCh' successfully
func TestRPCDuplex_Forward(t *testing.T) {

	connA, connB := net.Pipe()
	defer connB.Close()

	bDuplex := NewRPCDuplex(connB, nil, false)

	p := append([]byte{0, 0, 3}, []byte("foo")...)
	errCh := make(chan error, 1)
	go func() {
		_, err := connA.Write(p)
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
// the rpc method 'double' from the remote's rpc.Server
func TestRPCDuplex_Serve(t *testing.T) {
	cA, cB := net.Pipe()

	serverA := rpc.NewServer()
	require.NoError(t, serverA.RegisterName("RPC", new(RPC)))

	serverB := rpc.NewServer()
	require.NoError(t, serverB.RegisterName("RPC", new(RPC)))

	errChA := make(chan error, 1)
	dA := NewRPCDuplex(cA, serverA, true)
	go func() { errChA <- dA.Serve() }()

	errChB := make(chan error, 1)
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

	close(errChA)
	close(errChB)
	assert.Nil(t, <-errChA)
	assert.Nil(t, <-errChB)

}

func TestRPCDuplex_Close(t *testing.T) {

	// It is expected that 'err==nil' when duplex.Close() is called and that calling duplex.forward()
	// after duplex.Close() will result in 'panic: send on closed channel'
	t.Run("branchConns_should_close_when_RPCDuplex_closes", func(t *testing.T) {

		cA, cB := net.Pipe()

		dA := NewRPCDuplex(cA, nil, true)
		dB := NewRPCDuplex(cB, nil, false)

		errChA := make(chan error, 1)
		go func() {
			_, err := dA.clientConn.Write([]byte("foo"))
			errChA <- err
		}()

		assert.Nil(t, dB.Close())
		assert.Panics(t, func() {
			err := dB.forward()
			log.Println(err)
		})
		assert.Nil(t, <-errChA)

	})

	// Closing any branchConn or RPCDuplex multiple times shouldn't result in panic
	t.Run("closing_branchConn_multiple_times", func(t *testing.T) {

		cA, cB := net.Pipe()

		serverA := rpc.NewServer()
		require.NoError(t, serverA.RegisterName("RPC", new(RPC)))

		serverB := rpc.NewServer()
		require.NoError(t, serverB.RegisterName("RPC", new(RPC)))

		errChA := make(chan error, 1)
		dA := NewRPCDuplex(cA, serverA, true)
		go func() { errChA <- dA.Serve() }()

		errChB := make(chan error, 1)
		dB := NewRPCDuplex(cB, serverB, false)
		go func() { errChB <- dB.Serve() }()

		for i := 0; i < 3; i++ {
			assert.Nil(t, dA.Close())
			assert.Nil(t, dB.Close())
			assert.Nil(t, dA.clientConn.Close())
			assert.Nil(t, dB.clientConn.Close())
			assert.Nil(t, dA.serverConn.Close())
			assert.Nil(t, dB.serverConn.Close())
		}

		close(errChA)
		close(errChB)
		assert.Nil(t, <-errChA)
		assert.Nil(t, <-errChB)
	})

}
