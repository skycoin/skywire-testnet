package messaging

import (
	"errors"
	"net"
	"sync"

	"github.com/skycoin/skywire/pkg/cipher"
)

var (
	// ErrConnExists occurs when a connection already exists.
	ErrConnExists = errors.New("connection already exists")

	// ErrAlreadyListening occurs when a pool already has a listener.
	ErrAlreadyListening = errors.New("pool is already listening")

	// ErrPoolClosed is returned by the Respond method after a call to Close.
	ErrPoolClosed = errors.New("pool closed")
)

// Pool represents a connection pool.
type Pool struct {
	config *LinkConfig
	conns  map[cipher.PubKey]*Link
	mux    sync.RWMutex
	wg     sync.WaitGroup

	listenerMutex sync.RWMutex
	listener      net.Listener

	callbacks *Callbacks
	doneChan  chan struct{}
}

// NewPool creates a new Pool.
func NewPool(config *LinkConfig, callbacks *Callbacks) *Pool {
	var pool Pool
	if config == nil {
		config = DefaultLinkConfig()
	}
	pool.config = config
	pool.conns = make(map[cipher.PubKey]*Link)
	pool.callbacks = &Callbacks{
		HandshakeComplete: pool.handshakeCompleteAction(callbacks.HandshakeComplete),
		Data:              pool.messageAction(callbacks.Data),
		Close:             pool.closeAction(callbacks.Close),
	}
	pool.doneChan = make(chan struct{})
	return &pool
}

// Initiate initiates a connection to a remote party.
func (p *Pool) Initiate(conn net.Conn, remoteID cipher.PubKey) (*Link, error) {
	if link, ok := p.Link(remoteID); ok {
		return link, ErrConnExists
	}
	link, err := NewLink(conn,
		&LinkConfig{
			Public:           p.config.Public,
			Secret:           p.config.Secret,
			HandshakeTimeout: p.config.HandshakeTimeout,
			Logger:           p.config.Logger,
			Remote:           remoteID,
			Initiator:        true,
		},
		p.callbacks)
	if err != nil {
		return nil, err
	}
	if err := link.Open(&p.wg); err != nil {
		return nil, err
	}
	return link, nil
}

// Listener returns the current listener used by the pool.
func (p *Pool) Listener() net.Listener {
	p.listenerMutex.RLock()
	defer p.listenerMutex.RUnlock()
	return p.listener
}

// Respond responds to remotely initiated connections accepting connections from the given net.Listener.
// This is a blocking call.
func (p *Pool) Respond(l net.Listener) error {
	p.listenerMutex.Lock()
	if p.listener != nil {
		p.listenerMutex.Unlock()
		return ErrAlreadyListening
	}
	p.listener = l
	p.listenerMutex.Unlock()

	for {
		c, err := p.listener.Accept()
		if err != nil {
			select {
			case <-p.doneChan:
				return ErrPoolClosed
			default:
			}

			return err
		}
		var conn *Link
		conn, err = NewLink(c,
			&LinkConfig{
				Public:           p.config.Public,
				Secret:           p.config.Secret,
				HandshakeTimeout: p.config.HandshakeTimeout,
				Logger:           p.config.Logger,
			},
			p.callbacks)
		if err != nil {
			return err
		}
		// TODO(evanlinjin): Deal with a connection's failure.
		_ = conn.Open(&p.wg) // nolint
	}
}

// Close closes the Pool.
func (p *Pool) Close() error {
	p.closeDoneChan()
	p.listenerMutex.Lock()
	if p.listener != nil {
		p.listener.Close()
	}
	p.listenerMutex.Unlock()

	p.mux.Lock()
	for _, conn := range p.conns {
		conn.Close()
	}
	p.mux.Unlock()
	p.wg.Wait()
	return nil
}

// Link returns a connection of a given remote static public key.
func (p *Pool) Link(id cipher.PubKey) (*Link, bool) {
	p.mux.RLock()
	conn, ok := p.conns[id]
	p.mux.RUnlock()
	return conn, ok
}

// All obtains all connections within pool.
func (p *Pool) All() []*Link {
	p.mux.RLock()
	out := make([]*Link, len(p.conns))
	i := 0
	for _, conn := range p.conns {
		out[i] = conn
		i++
	}
	p.mux.RUnlock()
	return out
}

// ConnAction performs an action on a connection.
// If ok is true, ConnAction is to be called on the next connection.
type ConnAction func(id cipher.PubKey, conn *Link) (ok bool)

// Range ranges over all connections within pool and exits on caller's instruction.
func (p *Pool) Range(action ConnAction) error {
	p.mux.RLock()
	for id, conn := range p.conns {
		if ok := action(id, conn); !ok {
			break
		}
	}
	p.mux.RUnlock()
	return nil
}

func (p *Pool) setConn(id cipher.PubKey, conn *Link) {
	p.mux.Lock()
	p.conns[id] = conn
	p.mux.Unlock()
}

func (p *Pool) delConn(id cipher.PubKey) {
	p.mux.Lock()
	delete(p.conns, id)
	p.mux.Unlock()
}

func (p *Pool) closeDoneChan() {
	select {
	case <-p.doneChan: // already closed
	default:
		close(p.doneChan)
	}
}

func (p *Pool) handshakeCompleteAction(action HandshakeCompleteAction) HandshakeCompleteAction {
	if action == nil {
		action = func(conn *Link) {}
	}
	return func(conn *Link) {
		p.setConn(conn.Remote(), conn)
		action(conn)
	}
}

func (p *Pool) messageAction(action FrameAction) FrameAction {
	if action == nil {
		action = func(conn *Link, dt FrameType, body []byte) error {
			return nil
		}
	}
	return func(conn *Link, dt FrameType, body []byte) error {
		return action(conn, dt, body)
	}
}

func (p *Pool) closeAction(action TCPCloseAction) TCPCloseAction {
	if action == nil {
		action = func(conn *Link, remote bool) {}
	}
	return func(conn *Link, remote bool) {
		p.delConn(conn.Remote())
		action(conn, remote)
	}
}
