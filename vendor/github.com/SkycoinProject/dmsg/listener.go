package dmsg

import (
	"net"
	"sync"

	"github.com/SkycoinProject/dmsg/cipher"
)

// Listener listens for remote-initiated transports.
type Listener struct {
	pk     cipher.PubKey
	port   uint16
	mx     sync.Mutex // protects 'accept'
	accept chan *Transport
	done   chan struct{}
	once   sync.Once
}

func newListener(pk cipher.PubKey, port uint16) *Listener {
	return &Listener{
		pk:     pk,
		port:   port,
		accept: make(chan *Transport, AcceptBufferSize),
		done:   make(chan struct{}),
	}
}

// Accept accepts a connection.
func (l *Listener) Accept() (net.Conn, error) {
	return l.AcceptTransport()
}

// Close closes the listener.
func (l *Listener) Close() error {
	if l.close() {
		return nil
	}
	return ErrClientClosed
}

func (l *Listener) close() (closed bool) {
	l.once.Do(func() {
		closed = true

		l.mx.Lock()
		defer l.mx.Unlock()

		close(l.done)
		for {
			select {
			case <-l.accept:
			default:
				close(l.accept)
				return
			}
		}
	})
	return closed
}

func (l *Listener) isClosed() bool {
	select {
	case <-l.done:
		return true
	default:
		return false
	}
}

// Addr returns the listener's address.
func (l *Listener) Addr() net.Addr {
	return Addr{
		PK:   l.pk,
		Port: l.port,
	}
}

// AcceptTransport accepts a transport connection.
func (l *Listener) AcceptTransport() (*Transport, error) {
	select {
	case <-l.done:
		return nil, ErrClientClosed
	case tp, ok := <-l.accept:
		if !ok {
			return nil, ErrClientClosed
		}
		return tp, nil
	}
}

// Type returns the transport type.
func (l *Listener) Type() string {
	return Type
}

// IntroduceTransport handles a transport after receiving a REQUEST frame.
func (l *Listener) IntroduceTransport(tp *Transport) error {
	l.mx.Lock()
	defer l.mx.Unlock()

	if l.isClosed() {
		return ErrClientClosed
	}

	select {
	case <-l.done:
		return ErrClientClosed

	case l.accept <- tp:
		if err := tp.WriteAccept(); err != nil {
			return err
		}
		go tp.Serve()
		return nil

	default:
		if err := tp.Close(); err != nil {
			log.WithError(err).Warn("Failed to close transport")
		}
		return ErrClientAcceptMaxed
	}
}
