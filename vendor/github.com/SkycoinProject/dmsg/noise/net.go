package noise

import (
	"errors"
	"io"
	"math"
	"net"
	"net/rpc"
	"sync"
	"time"

	"github.com/flynn/noise"

	"github.com/SkycoinProject/dmsg/cipher"
)

var (
	// ErrAlreadyServing is returned when an operation fails due to an operation
	// that is currently running.
	ErrAlreadyServing = errors.New("already serving")

	// ErrPacketTooBig occurs when data is too large.
	ErrPacketTooBig = errors.New("data too large to contain within a packet")

	// HandshakeXK is the XK handshake pattern.
	HandshakeXK = noise.HandshakeXK

	// HandshakeKK is the KK handshake pattern.
	HandshakeKK = noise.HandshakeKK

	// AcceptHandshakeTimeout determines how long a noise hs should take.
	AcceptHandshakeTimeout = time.Second * 10
)

// RPCClientDialer attempts to redial to a remotely served RPCClient.
// It exposes an RPCServer to the remote server.
// The connection is encrypted via noise.
type RPCClientDialer struct {
	config  Config
	pattern noise.HandshakePattern
	addr    string
	conn    net.Conn
	mu      sync.Mutex
	done    chan struct{} // nil: loop is not running, non-nil: loop is running.
}

// NewRPCClientDialer creates a new RPCClientDialer.
func NewRPCClientDialer(addr string, pattern noise.HandshakePattern, config Config) *RPCClientDialer {
	return &RPCClientDialer{config: config, pattern: pattern, addr: addr}
}

// Run repeatedly dials to remote until a successful connection is established.
// It exposes a RPC Server.
// It will return if Close is called or crypto fails.
func (d *RPCClientDialer) Run(srv *rpc.Server, retry time.Duration) error {
	if ok := d.setDone(); !ok {
		return ErrAlreadyServing
	}
	for {
		if err := d.establishConn(); err != nil {
			// Only return if not network error.
			if _, ok := err.(net.Error); !ok {
				return err
			}
		} else {
			// Only serve when then dial succeeds.
			srv.ServeConn(d.conn)
			d.setConn(nil)
		}
		select {
		case <-d.done:
			d.clearDone()
			return nil
		case <-time.After(retry):
		}
	}
}

// Close closes the handler.
func (d *RPCClientDialer) Close() (err error) {
	if d == nil {
		return nil
	}
	d.mu.Lock()
	if d.done != nil {
		close(d.done)
	}
	if d.conn != nil {
		err = d.conn.Close()
	}
	d.mu.Unlock()
	return
}

// This operation should be atomic, hence protected by mutex.
func (d *RPCClientDialer) establishConn() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	conn, err := net.Dial("tcp", d.addr)
	if err != nil {
		return err
	}
	ns, err := New(d.pattern, d.config)
	if err != nil {
		return err
	}
	conn, err = WrapConn(conn, ns, time.Second*5)
	if err != nil {
		return err
	}
	d.conn = conn
	return nil
}

func (d *RPCClientDialer) setConn(conn net.Conn) {
	d.mu.Lock()
	d.conn = conn
	d.mu.Unlock()
}

func (d *RPCClientDialer) setDone() (ok bool) {
	d.mu.Lock()
	if ok = d.done == nil; ok {
		d.done = make(chan struct{})
	}
	d.mu.Unlock()
	return
}

func (d *RPCClientDialer) clearDone() {
	d.mu.Lock()
	d.done = nil
	d.mu.Unlock()
}

// Addr is the address of a either an AppNode or ManagerNode.
type Addr struct {
	PK   cipher.PubKey
	Addr net.Addr
}

// Network returns the network type.
func (a Addr) Network() string {
	return "noise"
}

// String implements fmt.Stringer
func (a Addr) String() string {
	return a.Addr.String() + "(" + a.PK.Hex() + ")"
}

// Conn wraps a net.Conn and encrypts the connection with noise.
type Conn struct {
	net.Conn
	ns *ReadWriter
}

// WrapConn wraps a provided net.Conn with noise.
func WrapConn(conn net.Conn, ns *Noise, hsTimeout time.Duration) (*Conn, error) {
	rw := NewReadWriter(conn, ns)
	if err := rw.Handshake(hsTimeout); err != nil {
		return nil, err
	}
	return &Conn{Conn: conn, ns: rw}, nil
}

// Read reads from the noise-encrypted connection.
func (c *Conn) Read(b []byte) (int, error) {
	return c.ns.Read(b)
}

// Write writes to the noise-encrypted connection.
func (c *Conn) Write(b []byte) (int, error) {
	if len(b) > math.MaxUint16 {
		return 0, io.ErrShortWrite
	}
	return c.ns.Write(b)
}

// LocalAddr returns the local address of the connection.
func (c *Conn) LocalAddr() net.Addr {
	return &Addr{
		PK:   c.ns.LocalStatic(),
		Addr: c.Conn.LocalAddr(),
	}
}

// RemoteAddr returns the remote address of the connection.
func (c *Conn) RemoteAddr() net.Addr {
	return &Addr{
		PK:   c.ns.RemoteStatic(),
		Addr: c.Conn.RemoteAddr(),
	}
}

// Listener accepts incoming connections and encrypts with noise.
type Listener struct {
	net.Listener
	pk      cipher.PubKey
	sk      cipher.SecKey
	init    bool
	pattern noise.HandshakePattern
}

// WrapListener wraps a listener and encrypts incoming connections with noise.
func WrapListener(lis net.Listener, pk cipher.PubKey, sk cipher.SecKey, init bool, pattern noise.HandshakePattern) *Listener {
	return &Listener{Listener: lis, pk: pk, sk: sk, init: init, pattern: pattern}
}

// Accept calls Accept from the underlying net.Listener and encrypts the
// obtained net.Conn with noise.
func (ml *Listener) Accept() (net.Conn, error) {
	for {
		conn, err := ml.Listener.Accept()
		if err != nil {
			return nil, err
		}
		ns, err := New(ml.pattern, Config{
			LocalPK:   ml.pk,
			LocalSK:   ml.sk,
			Initiator: ml.init,
		})
		if err != nil {
			return nil, err
		}
		rw := NewReadWriter(conn, ns)
		if err := rw.Handshake(AcceptHandshakeTimeout); err != nil {
			noiseLogger.WithError(err).Warn("accept: noise handshake failed.")
			continue
		}
		noiseLogger.Infoln("accepted:", rw.RemoteStatic())
		return &Conn{Conn: conn, ns: rw}, nil
	}
}

// Addr returns the local address of the noise-encrypted Listener.
func (ml *Listener) Addr() net.Addr {
	return &Addr{
		PK:   ml.pk,
		Addr: ml.Listener.Addr(),
	}
}
