package netutil

import (
	"encoding/binary"
	"errors"
	"io"
	"log"
	"net"
)

const readChSize int = 2

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
}

// NewRPCDuplex initiates a new RPCDuplex struct
func NewRPCDuplex(conn net.Conn, initiator bool) *RPCDuplex {
	var d RPCDuplex
	clientCh := make(chan []byte, readChSize)
	serverCh := make(chan []byte, readChSize)

	// branchConn implements net.Conn and assigned it to d.clientConn and d.serverConn
	if initiator {
		d.clientConn = &branchConn{Conn: conn, prefix: 0, readCh: clientCh}
		d.serverConn = &branchConn{Conn: conn, prefix: 1, readCh: serverCh}
	} else {
		d.clientConn = &branchConn{Conn: conn, prefix: 1, readCh: clientCh}
		d.serverConn = &branchConn{Conn: conn, prefix: 0, readCh: serverCh}
	}
	d.conn = conn

	return &d
}

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

// Serve calls Forward() in a loop and direct the packets to the appropriate branchConn based on it's prefix
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
		d.serverConn.readCh <- data
	} else if prefix == d.clientConn.prefix {
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
