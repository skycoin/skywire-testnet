package netutil

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
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
	ch := make(chan []byte)

	// PrefixedConn implements net.Conn and assigned it to d.clientConn and d.serverConn
	if initiator {
		d.clientConn = &PrefixedConn{Conn: conn, prefix: 0, name: "clientConn", readChan: ch}
		d.serverConn = &PrefixedConn{Conn: conn, prefix: 1, name: "serverConn", readChan: ch}
	} else {
		d.clientConn = &PrefixedConn{Conn: conn, prefix: 1, name: "clientConn", readChan: ch}
		d.serverConn = &PrefixedConn{Conn: conn, prefix: 0, name: "serverConn", readChan: ch}
	}
	d.conn = conn

	return &d
}

// ReadHeader reads the first bytes of the data and returns the prefix of the packet
func (d *RPCDuplex) ReadHeader() byte {

	var bs = make([]byte, 1)

	// Read the 1st byte from prefix
	_, err := d.conn.Read(bs[:1])
	if err != nil {
		log.Fatalln("error reading prefix from conn", err)
	}

	return bs[0]
}

// Forward forwards one packet from Original conn to PrefixedConn based on the packet's prefix
func (d *RPCDuplex) Forward() error {

	// Reads 1st byte of prefixed connection to determine which conn to forward to
	prefix := d.ReadHeader()

	// Reads into data from conn until original conn is close
	data, err := ioutil.ReadAll(d.conn)
	if err != nil {
		if err != io.EOF {
			fmt.Println("read error:", err)
		}
		panic(err)
	}

	// //or ???
	// Isn't it an issue when data exceeds fixed byte size limit. Won't I still need the len of data being read?
	// data := make([]byte, 256)
	// _, err := d.conn.Read(data)
	// if err != nil {
	// 	return err
	// }

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

	defer close(pc.readChan)

	// Reads in data from chan []byte pushed from Original Conn
	data := <-pc.readChan

	// Compare length of data with b to check if a longer buffer is required
	if len(data) > len(b) {
		err = io.ErrShortBuffer
		return
	}

	return copy(b, data), nil
}

// Write prefixes a connection with either 0 or 1 and writes this prefixed data stream
// back to the original conn
func (pc *PrefixedConn) Write(b []byte) (n int, err error) {
	n, err = pc.Conn.Write(append([]byte{pc.prefix}, b...))
	return n - 1, err
}
