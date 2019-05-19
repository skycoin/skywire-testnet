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

// RPCDuplex holds the basic structure of two prefixed connections and the original connection
type RPCDuplex struct {
	conn       net.Conn
	clientConn *branchConn
	serverConn *branchConn
	rpcS       *rpc.Server
	rpcC       *rpc.Client
	rpcUse     bool
}

// NewRPCDuplex initiates a new RPCDuplex struct
func NewRPCDuplex(conn net.Conn, srv *rpc.Server, initiator bool, rpcUse bool) *RPCDuplex {
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

	// Right now, the duplex can call RPC methods under the following condition:
	// Init rpc.NewClient(d.clientConn) and serve rpc.ServeConn(d.serverConn)
	// or
	// Init rpc.NewClient(d.serverConn) and serve rpc.ServeConn(d.clientConn)
	//
	// However, branchConn read/write will hang if a new rpc client is created
	// or a rpc connection with either one of the branch connection is served.
	//
	// It will only be able to read/write message under the following condition:
	// 1) client writes multiple messages to server
	// Init rpc.NewClient(d.clientConn) and serve rpc.ServeConn(d.clientConn)
	//
	// 2) server writes multiple messages to client
	// Init rpc.NewClient(d.serverConn) and serve rpc.ServeConn(d.serverConn)
	//
	// 3) We don't create any new rpc client and we don't serve the rpc servers

	// This means that the duplex can either call RPC methods or
	// read/write packets with branchConn. It cannot do both at the same time.
	// My only way to circumvent this issue at the moment is to simply declare at start
	// whether we are creating the duplex for calling RPC only. Further investigation
	// required in order to over come this issue
	if rpcUse == true {
		d.rpcUse = rpcUse
		d.rpcC = rpc.NewClient(d.clientConn)
		// d.rpcC = rpc.NewClient(d.serverConn)
	}

	return &d
}

// Client returns the internal RPC Client.
func (d *RPCDuplex) Client() *rpc.Client { return d.rpcC }

// readHeader reads the first three bytes of the data and returns the prefix and size of the packet
func (d *RPCDuplex) readHeader() (byte, uint16) {

	var b = make([]byte, 3)

	// Read the first three byte of packet
	// 1st byte is prefix and the 2nd and 3rd byte is the encoded size
	_, err := d.conn.Read(b[:3])
	if err != nil {
		log.Fatalln("error reading header from conn", err)
	}

	prefix := b[0]
	size := binary.BigEndian.Uint16(b[1:3])

	return prefix, size
}

// forward forwards one packet from Original conn to appropriate branchConn based on the packet's prefix
func (d *RPCDuplex) forward() error {

	// Reads 1st byte of prefixed connection to determine which conn to forward to
	prefix, size := d.readHeader()

	// Reads packet with size 'size' from Original Conn into data
	data := make([]byte, size)
	// n, err := d.conn.Read(data)
	_, err := d.conn.Read(data)
	if err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}

	// Push data from original conn into branchConn's chan []byte
	switch {
	case prefix == d.serverConn.prefix:
		// log.Println("serverConn", string(data[:n]))
		d.serverConn.readCh <- data
	case prefix == d.clientConn.prefix:
		// log.Println("clientConn", string(data[:n]))
		d.clientConn.readCh <- data
	default:
		log.Fatalln(errors.New("error encountered while forwarding packets, data header contains incorrect or empty prefix"))
	}

	return nil
}

// Serve is a blocking function that serves the RPC server and runs the event loop that forwards data to branchConns.
func (d *RPCDuplex) Serve() error {

	if d.rpcUse == true {
		go d.rpcS.ServeConn(d.serverConn)
		// go d.rpcS.ServeConn(d.clientConn)
	}

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
