package netutil

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"time"
	"unicode/utf8"
)

// PrefixedConn will inherit net.Conn from the interface.
type PrefixedConn struct {
	prefix    byte
	writeConn io.Writer    // Original connection. net.Conn has a write method therefore it implements the writer interface.
	readBuf   bytes.Buffer // Read data from original connection. It is RPCDuplex's responsibility to push to here.
}

// RPCDuplex holds the basic structure of two prefixed connections
type RPCDuplex struct {
	clientConn *PrefixedConn
	serverConn *PrefixedConn
}

// NewRPCDuplex initiates a new RPCDuplex struct and reads in the
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

	return &d
}

// removeAtBytes remove a rune from a bytes.Buffer at ith location
func removeAtBytes(p []byte, i int) []byte {
	j := 0
	k := 0
	for k < len(p) {
		_, n := utf8.DecodeRune(p[k:])
		if i == j {
			p = p[:k+copy(p[k:], p[k+n:])]
		}
		j++
		k += n
	}
	return p
}

// Read reads in prefixed data from root connection and reads it into the appropriate branch connection
func (pc *PrefixedConn) Read(b []byte) (n int, err error) {

	// Remove the first byte from the bytes.Buffer
	pc.readBuf.Truncate(len(removeAtBytes(pc.readBuf.Bytes(), 0)))

	// The bytes.Buffer readBuf reads data into b
	n, err = pc.readBuf.Read(b)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println("Bytes read", n)
	fmt.Print("Bytes not read ", pc.readBuf.Len(), "\n")

	return n, err
}

// Write prefixes data to the connection and then writes this prefixed data to the root connection.
func (pc *PrefixedConn) Write(b []byte) (n int, err error) {

	n, err = pc.writeConn.Write(append([]byte{pc.prefix}, b...))
	if n > 0 {
		n--
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
