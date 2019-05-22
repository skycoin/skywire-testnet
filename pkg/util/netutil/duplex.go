package netutil

import (
	"encoding/binary"
	"errors"
	"io"
	"log"
	"net"
	"net/rpc"
)

const bufferSize = 8

// branchConn will inherit the net.Conn interface.
type branchConn struct {
	net.Conn
	prefix byte
	readCh chan []byte
}

// Read reads in data from original conn through branchConn's chan [].
// If read []byte is smaller than length of packet, io.ErrShortBuffer
// is raised.
func (bc *branchConn) Read(b []byte) (n int, err error) {

	// Reads in data from chan []byte pushed from Original Conn
	data := <-bc.readCh

	// Compare length of data with b to check if a longer buffer is required
	if len(data) > len(b) {
		err = io.ErrShortBuffer
		return n, err
	}

	return copy(b, data), nil
}

// Write prefixes a connection with either 0 or 1 and writes this prefixed data stream
// back to the original conn
func (bc *branchConn) Write(b []byte) (n int, err error) {

	buf := make([]byte, 3)
	buf[0] = byte(bc.prefix)
	binary.BigEndian.PutUint16(buf[1:3], uint16(len(b)))

	n, err = bc.Conn.Write(append(buf, b...))
	return n - 3, err
}

// Close closes all internal readCh and branchConn
func (bc *branchConn) Close() error {
	close(bc.readCh)
	return bc.Conn.Close()
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
	clientCh := make(chan []byte, bufferSize)
	serverCh := make(chan []byte, bufferSize)

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

	if srv != nil {
		d.rpcC = rpc.NewClient(d.clientConn)
	}

	return &d
}

// Client returns the internal RPC Client.
func (d *RPCDuplex) Client() *rpc.Client { return d.rpcC }

// Close closes all opened connections and channels
func (d *RPCDuplex) Close() error {

	if err := d.clientConn.Close(); err != nil {
		return err
	}

	if err := d.serverConn.Close(); err != nil {
		return err
	}

	return d.conn.Close()
}

// forward forwards one packet from Original conn to appropriate branchConn based on the packet's prefix
func (d *RPCDuplex) forward() error {

	var b = make([]byte, 3)

	// Reads the first three bytes of the data and returns the prefix and size of the packet
	if _, err := d.conn.Read(b[:3]); err != nil {
		log.Fatalln("error reading header from conn", err)
	}

	prefix := b[0]
	size := binary.BigEndian.Uint16(b[1:3])

	// Reads the rest of the packet into data with prefixed size
	data := make([]byte, size)
	if _, err := d.conn.Read(data); err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}

	// Push data from original conn into branchConn's chan []byte
	switch prefix {
	case d.serverConn.prefix:
		d.serverConn.readCh <- data
	case d.clientConn.prefix:
		d.clientConn.readCh <- data
	default:
		log.Fatalln(errors.New("error encountered while forwarding packets, data header contains incorrect or empty prefix"))
	}

	return nil
}

// Serve is a blocking function that serves the RPC server and runs the event loop that forwards data to branchConns.
func (d *RPCDuplex) Serve() error {

	go d.rpcS.ServeConn(d.serverConn)

	for {
		err := d.forward()
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
