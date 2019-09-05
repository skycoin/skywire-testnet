package stcp

import (
	"context"
	"errors"
	"fmt"
	"github.com/skycoin/dmsg"
	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/skycoin/src/util/logging"
	"io"
	"net"
	"sync"
	"time"
)

type Conn struct {
	net.Conn
	lAddr    dmsg.Addr
	rAddr    dmsg.Addr
	freePort func()
}

func newConn(conn net.Conn, deadline time.Time, hs Handshake, freePort func()) (*Conn, error) {
	lAddr, rAddr, err := hs(conn, deadline)
	if err != nil {
		_ = conn.Close() //nolint:errcheck
		freePort()
		return nil, err
	}
	return &Conn{Conn: conn, lAddr: lAddr, rAddr: rAddr, freePort: freePort}, nil
}

func (c *Conn) LocalAddr() net.Addr {
	return c.lAddr
}

func (c *Conn) RemoteAddr() net.Addr {
	return c.rAddr
}

func (c *Conn) Close() error {
	if c.freePort != nil {
		c.freePort()
	}
	return c.Conn.Close()
}

type Listener struct {
	lAddr    dmsg.Addr
	freePort func()
	accept   chan *Conn
	done     chan struct{}
	once     sync.Once
	mx       sync.Mutex
}

func newListener(lAddr dmsg.Addr, freePort func()) *Listener {
	return &Listener{
		lAddr:    lAddr,
		freePort: freePort,
		accept:   make(chan *Conn),
		done:     make(chan struct{}),
	}
}

func (l *Listener) Introduce(conn *Conn) error {
	select {
	case <-l.done:
		return io.ErrClosedPipe
	default:
		l.mx.Lock()
		defer l.mx.Unlock()

		select {
		case l.accept <-conn:
			return nil
		case <-l.done:
			return io.ErrClosedPipe
		}
	}
}

func (l *Listener) Accept() (net.Conn, error) {
	conn, ok := <-l.accept
	if !ok {
		return nil, io.ErrClosedPipe
	}
	return conn, nil
}

func (l *Listener) Close() error {
	l.once.Do(func() {
		close(l.done)

		l.mx.Lock()
		close(l.accept)
		l.mx.Unlock()

		l.freePort()
	})
	return nil
}

func (l *Listener) Addr() net.Addr {
	return l.lAddr
}

type Client struct {
	log *logging.Logger

	lPK  cipher.PubKey
	lSK  cipher.SecKey
	t    PKTable
	p    *Porter

	lMap map[uint16]*Listener // key: lPort
	mx   sync.Mutex

	done chan struct{}
	once sync.Once
}

func (c *Client) Dial(rPK cipher.PubKey, rPort uint16) (*Conn, error) {
	if c.isClosed() {
		return nil, io.ErrClosedPipe
	}

	tcpAddr, ok := c.t.Addr(rPK)
	if !ok {
		return nil, fmt.Errorf("pk table: entry of %s does not exist", rPK)
	}
	conn, err := net.Dial("tcp", tcpAddr)
	if err != nil {
		return nil, err
	}

	lPort, freePort, err := c.p.ReserveEphemeral(context.TODO())
	if err != nil {
		return nil, err
	}
	hs := InitiatorHandshake(c.lSK, dmsg.Addr{PK: c.lPK, Port: lPort}, dmsg.Addr{PK: rPK, Port: rPort})
	return newConn(conn, time.Now().Add(HandshakeTimeout), hs, freePort)
}

func (c *Client) Listen(lPort uint16) (*Listener, error) {
	if c.isClosed() {
		return nil, io.ErrClosedPipe
	}

	ok, freePort := c.p.Reserve(lPort)
	if !ok {
		return nil, errors.New("port is already occupied")
	}

	c.mx.Lock()
	defer c.mx.Unlock()

	lAddr := dmsg.Addr{PK: c.lPK, Port: lPort}
	lis := newListener(lAddr, freePort)
	c.lMap[lPort] = lis
	return lis, nil
}

func (c *Client) AcceptLoop(tcpL net.Listener) {
	for {
		if err := c.acceptTCPConn(tcpL); err != nil {
			c.log.Warnf("failed to accept incoming connection: %v", err)
			if !IsHandshakeError(err) {
				return
			}
		}
	}
}

func (c *Client) acceptTCPConn(tcpL net.Listener) error {
	if c.isClosed() {
		return io.ErrClosedPipe
	}

	tcpConn, err := tcpL.Accept()
	if err != nil {
		return err
	}
	var lis *Listener
	hs := ResponderHandshake(func(f2 Frame2) error {
		c.mx.Lock()
		defer c.mx.Unlock()
		var ok bool
		if lis, ok = c.lMap[f2.DstAddr.Port]; !ok {
			return errors.New("not listening on given port")
		}
		return nil
	})
	conn, err := newConn(tcpConn, time.Now().Add(HandshakeTimeout), hs, nil)
	if err != nil {
		return err
	}
	return lis.Introduce(conn)
}

func (c *Client) Close() error {
	c.once.Do(func() {
		close(c.done)

		c.mx.Lock()
		defer c.mx.Unlock()

		for _, lis := range c.lMap {
			_ = lis.Close() // nolint:errcheck
		}
	})
	return nil
}

func (c *Client) isClosed() bool {
	select {
	case <-c.done:
		return true
	default:
		return false
	}
}
