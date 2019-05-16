package netutil

import (
	"encoding/binary"
	"errors"
	"io"
	"log"
	"net"
	"net/rpc"
)

// branchConn will inherit the net.Conn interface.
type branchConn struct {
	net.Conn
	prefix byte
	readCh chan []byte
}

// RPCDuplex holds the basic structure of two prefixed connections and the original connection
type RPCDuplex struct {
	conn       net.Conn
	clientConn *branchConn
	serverConn *branchConn
	rpcS       *rpc.Server
	rpcC       *rpc.Client
}

// NewRPCDuplex initiates a new RPCDuplex struct
func NewRPCDuplex(conn net.Conn, srv *rpc.Server, initiator bool) *RPCDuplex {
	var d RPCDuplex
	clientCh := make(chan []byte, 1)
	serverCh := make(chan []byte, 1)

	// branchConn implements net.Conn and assigned it to d.clientConn and d.serverConn
	if initiator {
		d.clientConn = &branchConn{Conn: conn, prefix: 0, readCh: clientCh}
		d.serverConn = &branchConn{Conn: conn, prefix: 1, readCh: serverCh}
	} else {
		d.clientConn = &branchConn{Conn: conn, prefix: 1, readCh: clientCh}
		d.serverConn = &branchConn{Conn: conn, prefix: 0, readCh: serverCh}
	}
	d.conn = conn
	d.rpcS = srv

	return &d
}

// Client returns the internal RPC Client.
func (d *RPCDuplex) Client() *rpc.Client { return d.rpcC }

// ReadHeader reads the first three bytes of the data and returns the prefix and size of the packet
func (d *RPCDuplex) ReadHeader() (byte, uint16) {

	var b = make([]byte, 3)
	var size uint16

	// Read the first three byte of packet
	// 1st byte is prefix and the 2nd and 3rd byte is the encoded size
	_, err := d.conn.Read(b[:3])
	if err != nil {
		log.Fatalln("error reading header from conn", err)
	}

	prefix := b[0]
	size = binary.BigEndian.Uint16(b[1:3])

	return prefix, size
}

// Serve is a blocking function that serves the RPC server and runs the event loop that forwards data to branchConns.
func (d *RPCDuplex) Serve() error {

	for {
		err := d.Forward()
		switch err {
		case nil:
			continue
		case io.EOF:
			return nil
		default:
			log.Fatalln(err)
		}
	}
}

// A ---> B (A is initiator, B is responder)
// Prefix read/writes with either 0 (talk to RPC server) or 1 (talk to RPC client).

// A -> B (0-prefixed) : Talk to B's RPC server.
// A -> B (1-prefixed) : Talk to B's RPC client.
// A <- B (0-prefixed) : Talk to A's RPC client.
// A <- B (1-prefixed) : Talk to A's RPC server.
// 0 prefix (net.Conn), 1 prefix (net.Conn)

// Forward forwards one packet from Original conn to appropriate branchConn based on the packet's prefix
func (d *RPCDuplex) Forward() error {

	// Reads 1st byte of prefixed connection to determine which conn to forward to
	prefix, size := d.ReadHeader()

	// Reads packet with size 'size' from Original Conn into data
	data := make([]byte, size)
	_, err := d.conn.Read(data)
	if err != nil {
		if err != io.EOF {
			return err
		}
		return nil
	}

	// Push data from original conn into branchConn's chan []byte
	if prefix == d.serverConn.prefix {

		// Serve conn and init new branch client
		go d.rpcS.ServeConn(d.clientConn)
		d.rpcC = rpc.NewClient(d.serverConn)

		d.serverConn.readCh <- data
	} else if prefix == d.clientConn.prefix {

		// Serve conn and init new branch client
		go d.rpcS.ServeConn(d.serverConn)
		d.rpcC = rpc.NewClient(d.clientConn)

		d.clientConn.readCh <- data
	} else {
		log.Fatalln(errors.New("error encountered while forwarding packets"))
	}

	return nil
}

// Read reads in data from original conn through branchConn's chan [].
// If read []byte is smaller than length of packet, io.ErrShortBuffer
// is raised.
func (pc *branchConn) Read(b []byte) (n int, err error) {

	// Reads in data from chan []byte pushed from Original Conn
	data := <-pc.readCh

	// Compare length of data with b to check if a longer buffer is required
	if len(data) > len(b) {
		err = io.ErrShortBuffer
		return n, err
	}

	return copy(b, data), nil
}

// Write prefixes a connection with either 0 or 1 and writes this prefixed data stream
// back to the original conn
func (pc *branchConn) Write(b []byte) (n int, err error) {

	buf := make([]byte, 3)
	buf[0] = byte(pc.prefix)
	binary.BigEndian.PutUint16(buf[1:3], uint16(len(b)))

	n, err = pc.Conn.Write(append(buf, b...))
	return n - 3, err
}
