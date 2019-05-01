package netutil

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"
	"net"
	"time"
)

// By default bufio.Writer uses 4096 bytes long buffer as per
// https://github.com/golang/go/blob/c6c0f47e92771c9b4fced87b94c04f66e5d6eba5/src/bufio/bufio.go#L18
const defaultByteSize = 4096

// PrefixedConn will inherit the net.Conn interface.
type PrefixedConn struct {
	prefix    byte
	writeConn io.Writer    // Original connection. net.Conn has a write method therefore it implements the writer interface.
	readBuf   bytes.Buffer // Read data from original connection. It is RPCDuplex's responsibility to push to here.
}

// RPCDuplex holds the basic structure of two prefixed connections and the original connection
type RPCDuplex struct {
	conn       net.Conn
	clientConn *PrefixedConn
	serverConn *PrefixedConn
}

// NewRPCDuplex initiates a new RPCDuplex struct
func NewRPCDuplex(conn net.Conn, initiator bool) *RPCDuplex {
	var d RPCDuplex
	var buf bytes.Buffer

	// PrefixedConn implements net.Conn and assigned it to d.clientConn and d.serverConn
	if initiator {
		d.clientConn = &PrefixedConn{prefix: 0, writeConn: conn, readBuf: buf}
		d.serverConn = &PrefixedConn{prefix: 1, writeConn: conn, readBuf: buf}
	} else {
		d.clientConn = &PrefixedConn{prefix: 1, writeConn: conn, readBuf: buf}
		d.serverConn = &PrefixedConn{prefix: 0, writeConn: conn, readBuf: buf}
	}

	d.conn = conn

	return &d
}

// ReadHeader reads the first three bytes of the data and returns the prefix and size of the packets
func (d *RPCDuplex) ReadHeader() (byte, uint16) {

	var bs = make([]byte, defaultByteSize)
	var size uint16

	// Read the 1st byte from prefix
	_, err := d.conn.Read(bs[:1])
	if err != nil {
		log.Fatalln("error reading prefix from conn", err)
	}
	prefix := bs[0]

	// Reads the encoded size from the 2nd and 3rd byte
	_, err = d.conn.Read(bs[:2])
	if err != nil {
		log.Fatalln("error reading size from conn", err)
	}

	size = binary.BigEndian.Uint16(bs)

	return prefix, size
}

// Forward forwards data from Original conn to PrefixedConn given the prefix
func (d *RPCDuplex) Forward(prefix byte, size uint16) []byte {

	// Using the original conn to push data into the buffer
	buf := make([]byte, defaultByteSize)
	_, err := d.conn.Read(buf)
	if err != nil {
		log.Fatalln(err)
	}

	// ====================
	// FORWARDING LOGIC
	// ====================
	// A Initiator (0-prefixed) -> B (0-prefixed): A talks to B's RPC server
	// A Initiator (1-prefixed) -> B (1-prefixed): A talks to B's RPC client
	// A (0-prefixed) <- B Initiator (0-prefixed): B talks to A's RPC server
	// A (1-prefixed) <- B Initiator (1-prefixed): B talks to A's RPC client

	if (prefix == 0 && d.serverConn.prefix == 0) || (prefix == 1 && d.serverConn.prefix == 1) {
		d.serverConn.readBuf.Write(buf)
		_, err := d.serverConn.Read(buf[:size])
		if err != nil {
			log.Fatalln(err)
		}

		// log.Println("PrefixedConn.serverConn:", n)
		// log.Println("PrefixedConn.serverConn:", string(buf))

	} else if (prefix == 1 && d.clientConn.prefix == 1) || (prefix == 0 && d.clientConn.prefix == 0) {
		d.clientConn.readBuf.Write(buf)
		_, err := d.clientConn.Read(buf[:size])

		if err != nil {
			log.Fatalln(err)
		}

		// log.Println("PrefixedConn.clientConn:", n)
		// log.Println("PrefixedConn.clientConn:", string(buf))
	}

	return buf[:size]

}

// Read reads in prefixed data from root connection
func (pc *PrefixedConn) Read(b []byte) (n int, err error) {

	// The bytes.Buffer readBuf reads data into b
	n, err = pc.readBuf.Read(b)

	if err != nil {
		log.Fatalln(err)
	}

	return n, err
}

// Write prefixes data to the connection and then writes this prefixed data to the root connection.
func (pc *PrefixedConn) Write(b []byte) (n int, err error) {

	buf := make([]byte, 3)
	buf[0] = byte(pc.prefix)
	binary.BigEndian.PutUint16(buf[1:3], uint16(len(b)))

	n, err = pc.writeConn.Write(append(buf, b...))

	// Write returns the number of bytes written from p (0 <= n <= len(p))
	// and any error encountered that caused the write to stop early.
	// Write must return a non-nil error if it returns n < len(p).
	if n > 0 {
		n = n - 3
	}

	return n, err
}

// Close closes the connection.
func (pc *PrefixedConn) Close() error {
	return nil
}

// LocalAddr returns the local network address.
func (pc *PrefixedConn) LocalAddr() net.Addr {
	var addr net.Addr
	return addr
}

// RemoteAddr returns the remote network address.
func (pc *PrefixedConn) RemoteAddr() net.Addr {
	var addr net.Addr
	return addr
}

// SetDeadline sets the read
func (pc *PrefixedConn) SetDeadline(t time.Time) error {
	return nil
}

// SetReadDeadline sets the deadline
func (pc *PrefixedConn) SetReadDeadline(t time.Time) error {
	return nil
}

// SetWriteDeadline sets the deadline for future Write calls
func (pc *PrefixedConn) SetWriteDeadline(t time.Time) error {
	return nil
}
