package netutil

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"log"
	"net"
	"time"
)

// PrefixedConn will inherit the net.Conn interface.
type PrefixedConn struct {
	name      string
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
		d.clientConn = &PrefixedConn{name: "clientConn", prefix: 0, writeConn: conn, readBuf: buf}
		d.serverConn = &PrefixedConn{name: "serverConn", prefix: 1, writeConn: conn, readBuf: buf}
	} else {
		d.clientConn = &PrefixedConn{name: "clientConn", prefix: 1, writeConn: conn, readBuf: buf}
		d.serverConn = &PrefixedConn{name: "serverConn", prefix: 0, writeConn: conn, readBuf: buf}
	}

	d.conn = conn

	return &d
}

// ReadHeader reads the first three bytes of the data and returns the prefix and size of the packets
func (d *RPCDuplex) ReadHeader() (byte, uint16) {

	var bs = make([]byte, 2)
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

// Serve continuously calls forward in a loop alongside with ReadHeader
func (d *RPCDuplex) Serve() {

L:
	for {
		prefix, size := d.ReadHeader()

		err := d.Forward(prefix, size)
		switch err {
		case nil:
			break L
		default:
			log.Fatalln(err)
		}
	}
}

// Forward forwards one packet from Original conn to PrefixedConn given the prefix and size of payload
func (d *RPCDuplex) Forward(prefix byte, size uint16) error {

	data := make([]byte, size)

	_, err := d.conn.Read(data)
	if err != nil {
		return err
	}

	// Using the original conn to push data into the buffer
	if prefix == d.serverConn.prefix {
		d.serverConn.readBuf.Write(data[:size])
	} else if prefix == d.clientConn.prefix {
		d.clientConn.readBuf.Write(data[:size])
	} else {
		log.Fatalln(errors.New("error encountered while forwarding packets"))
	}

	return nil

}

// Read reads in prefixed data from root connection
func (pc *PrefixedConn) Read(b []byte) (n int, err error) {

	// log.Println("Buffer len:", pc.readBuf.Len())

	// Read reads the next len(p) bytes from the buffer or until the buffer
	// is drained. The return value n is the number of bytes read. If the
	// buffer has no data to return, err is io.EOF (unless len(p) is zero);
	// otherwise it is nil.
	for {
		n, err = pc.readBuf.Read(b)
		switch err {
		case nil:
			return n, err
		case io.EOF:
			continue
		default:
			log.Fatalln(err)
			return n, err
		}
	}

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
