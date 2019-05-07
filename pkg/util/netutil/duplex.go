package netutil

import (
	"encoding/binary"
	"errors"
	"io"
	"log"
	"net"
)

// PrefixedConn will inherit the net.Conn interface.
type PrefixedConn struct {
	net.Conn
	prefix   byte
	name     string
	readChan chan []byte
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
	clientCh := make(chan []byte)
	serverCh := make(chan []byte)

	// PrefixedConn implements net.Conn and assigned it to d.clientConn and d.serverConn
	if initiator {
		d.clientConn = &PrefixedConn{Conn: conn, prefix: 0, name: "clientConn", readChan: clientCh}
		d.serverConn = &PrefixedConn{Conn: conn, prefix: 1, name: "serverConn", readChan: serverCh}
	} else {
		d.clientConn = &PrefixedConn{Conn: conn, prefix: 1, name: "clientConn", readChan: clientCh}
		d.serverConn = &PrefixedConn{Conn: conn, prefix: 0, name: "serverConn", readChan: serverCh}
	}
	d.conn = conn

	return &d
}

// ReadHeader reads the first bytes of the data and returns the prefix of the packet
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

// Serve calls Forward() in a loop and would wait
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

// Forward forwards one packet from Original conn to PrefixedConn based on the packet's prefix
func (d *RPCDuplex) Forward() error {

	// Reads 1st byte of prefixed connection to determine which conn to forward to
	prefix, size := d.ReadHeader()

	data := make([]byte, size)
	_, err := d.conn.Read(data)
	if err != nil {
		if err != io.EOF {
			return err
		}
		return nil
	}

	// Push data from original conn into prefixedConn's chan []byte
	go func() {
		if prefix == d.serverConn.prefix {
			d.serverConn.readChan <- data
		} else if prefix == d.clientConn.prefix {
			d.clientConn.readChan <- data
		} else {
			log.Fatalln(errors.New("error encountered while forwarding packets"))
		}
	}()

	return nil

}

// Read reads in data from original conn through PrefixedConn's chan []byte
func (pc *PrefixedConn) Read(b []byte) (n int, err error) {

	// Reads in data from chan []byte pushed from Original Conn
	data := <-pc.readChan

	// Compare length of data with b to check if a longer buffer is required
	if len(data) > len(b) {
		err = io.ErrShortBuffer
		return n, err
	}

	return copy(b, data), nil
}

// Write prefixes a connection with either 0 or 1 and writes this prefixed data stream
// back to the original conn
func (pc *PrefixedConn) Write(b []byte) (n int, err error) {

	buf := make([]byte, 3)
	buf[0] = byte(pc.prefix)
	binary.BigEndian.PutUint16(buf[1:3], uint16(len(b)))

	n, err = pc.Conn.Write(append(buf, b...))

	return n - 3, err
}
