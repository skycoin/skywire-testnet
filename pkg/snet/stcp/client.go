package stcp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/SkycoinProject/dmsg"
	"github.com/SkycoinProject/dmsg/cipher"
	"github.com/SkycoinProject/skycoin/src/util/logging"
)

// Conn wraps an underlying net.Conn and modifies various methods to integrate better with the 'network' package.
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
		if freePort != nil {
			freePort()
		}
		return nil, err
	}
	return &Conn{Conn: conn, lAddr: lAddr, rAddr: rAddr, freePort: freePort}, nil
}

// LocalAddr implements net.Conn
func (c *Conn) LocalAddr() net.Addr {
	return c.lAddr
}

// RemoteAddr implements net.Conn
func (c *Conn) RemoteAddr() net.Addr {
	return c.rAddr
}

// Close implements net.Conn
func (c *Conn) Close() error {
	if c.freePort != nil {
		c.freePort()
	}
	return c.Conn.Close()
}

// Listener implements net.Listener
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

// Introduce is used by stcp.Client to introduce stcp.Conn to Listener.
func (l *Listener) Introduce(conn *Conn) error {
	select {
	case <-l.done:
		return io.ErrClosedPipe
	default:
		l.mx.Lock()
		defer l.mx.Unlock()

		select {
		case l.accept <- conn:
			return nil
		case <-l.done:
			return io.ErrClosedPipe
		}
	}
}

// Accept implements net.Listener
func (l *Listener) Accept() (net.Conn, error) {
	conn, ok := <-l.accept
	if !ok {
		return nil, io.ErrClosedPipe
	}
	return conn, nil
}

// Close implements net.Listener
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

// Addr implements net.Listener
func (l *Listener) Addr() net.Addr {
	return l.lAddr
}

// Client is the central control for incoming and outgoing 'stcp.Conn's.
type Client struct {
	log *logging.Logger

	lPK cipher.PubKey
	lSK cipher.SecKey
	t   PKTable
	p   *Porter

	lTCP net.Listener
	lMap map[uint16]*Listener // key: lPort
	mx   sync.Mutex

	done chan struct{}
	once sync.Once
}

// NewClient creates a net Client.
func NewClient(log *logging.Logger, pk cipher.PubKey, sk cipher.SecKey, t PKTable) *Client {
	if log == nil {
		log = logging.MustGetLogger("stcp")
	}
	return &Client{
		log:  log,
		lPK:  pk,
		lSK:  sk,
		t:    t,
		p:    newPorter(PorterMinEphemeral),
		lMap: make(map[uint16]*Listener),
		done: make(chan struct{}),
	}
}

// Serve serves the listening portion of the client.
func (c *Client) Serve(tcpAddr string) error {
	if c.lTCP != nil {
		return errors.New("already listening")
	}

	lTCP, err := net.Listen("tcp", tcpAddr)
	if err != nil {
		return err
	}
	c.lTCP = lTCP
	c.log.Infof("listening on tcp addr: %v", lTCP.Addr())

	go func() {
		for {
			if err := c.acceptTCPConn(); err != nil {
				c.log.Warnf("failed to accept incoming connection: %v", err)
				if !IsHandshakeError(err) {
					c.log.Warnf("stopped serving stcp")
					return
				}
			}
		}
	}()

	return nil
}

func (c *Client) acceptTCPConn() error {
	if c.isClosed() {
		return io.ErrClosedPipe
	}

	tcpConn, err := c.lTCP.Accept()
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

// Dial dials a new stcp.Conn to specified remote public key and port.
func (c *Client) Dial(ctx context.Context, rPK cipher.PubKey, rPort uint16) (*Conn, error) {
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

	lPort, freePort, err := c.p.ReserveEphemeral(ctx)
	if err != nil {
		return nil, err
	}
	hs := InitiatorHandshake(c.lSK, dmsg.Addr{PK: c.lPK, Port: lPort}, dmsg.Addr{PK: rPK, Port: rPort})
	return newConn(conn, time.Now().Add(HandshakeTimeout), hs, freePort)
}

// Listen creates a new listener for stcp.
// The created Listener cannot actually accept remote connections unless Serve is called beforehand.
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

// Close closes the Client.
func (c *Client) Close() error {
	if c == nil {
		return nil
	}
	c.once.Do(func() {
		close(c.done)

		c.mx.Lock()
		defer c.mx.Unlock()

		if c.lTCP != nil {
			_ = c.lTCP.Close() //nolint:errcheck
		}

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
