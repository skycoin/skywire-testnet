package netutil

import (
	"encoding/binary"
	"errors"
	"io"
	"log"
	"net"
	"net/rpc"
	"sync"
)

const bufferSize = 8

// branchConn will inherit the net.Conn interface.
type branchConn struct {
	duplex *RPCDuplex
	net.Conn
	prefix byte
	readCh chan []byte
}

// Close calls Duplex.Close to close internal readCh for both branchConn
func (bc *branchConn) Close() error {
	return bc.duplex.Close()
}

// Read reads in data from original conn through branchConn's chan [].
// If read []byte is smaller than length of packet, io.ErrShortBuffer
// is raised.
func (bc *branchConn) Read(b []byte) (int, error) {

	// Reads in data from chan []byte pushed from Original Conn
	data, ok := <-bc.readCh
	if !ok {
		return 0, io.ErrClosedPipe
	}

	// Compare length of data with b to check if a longer buffer is required
	if len(data) > len(b) {
		return 0, io.ErrShortBuffer
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

// RPCDuplex holds the basic structure of two prefixed connections and the original connection
type RPCDuplex struct {
	conn       net.Conn
	clientConn *branchConn
	serverConn *branchConn
	rpcS       *rpc.Server
	rpcC       *rpc.Client
	closeOnce  sync.Once
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

	d.clientConn.duplex = &d
	d.serverConn.duplex = &d

	return &d
}

// Client returns the internal RPC Client.
func (d *RPCDuplex) Client() *rpc.Client { return d.rpcC }

// closeDone close both branchConn's readCh
func (d *RPCDuplex) closeDone() {
	close(d.clientConn.readCh)
	close(d.serverConn.readCh)
}

// Close calls closeDone which close both branchConn's readCh
func (d *RPCDuplex) Close() error {

	var err error

	// closeOnce ensures that branchConn's readCh channel is closed once
	d.closeOnce.Do(func() {
		d.closeDone()
	})

	// Convert panic into error
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	return err
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
		if err := d.forward(); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}
