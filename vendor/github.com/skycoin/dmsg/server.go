package dmsg

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/dmsg/disc"
	"github.com/skycoin/dmsg/noise"
)

// ErrListenerAlreadyWrappedToNoise occurs when the provided net.Listener is already wrapped with noise.Listener
var ErrListenerAlreadyWrappedToNoise = errors.New("listener is already wrapped to *noise.Listener")

// Server represents a dms_server.
type Server struct {
	log *logging.Logger

	pk cipher.PubKey
	sk cipher.SecKey
	dc disc.APIClient

	addr  string
	lis   net.Listener
	conns map[cipher.PubKey]*ServerConn
	mx    sync.RWMutex

	wg sync.WaitGroup

	lisDone  int32
	doneOnce sync.Once
}

// NewServer creates a new dms_server.
func NewServer(pk cipher.PubKey, sk cipher.SecKey, addr string, l net.Listener, dc disc.APIClient) (*Server, error) {
	if addr == "" {
		addr = l.Addr().String()
	}

	if _, ok := l.(*noise.Listener); ok {
		return nil, ErrListenerAlreadyWrappedToNoise
	}

	return &Server{
		log:   logging.MustGetLogger("dms_server"),
		pk:    pk,
		sk:    sk,
		addr:  addr,
		lis:   noise.WrapListener(l, pk, sk, false, noise.HandshakeXK),
		dc:    dc,
		conns: make(map[cipher.PubKey]*ServerConn),
	}, nil
}

// SetLogger set's the logger.
func (s *Server) SetLogger(log *logging.Logger) {
	s.log = log
}

// Addr returns the server's listening address.
func (s *Server) Addr() string {
	return s.addr
}

func (s *Server) setConn(l *ServerConn) {
	s.mx.Lock()
	s.conns[l.remoteClient] = l
	s.mx.Unlock()
}

func (s *Server) delConn(pk cipher.PubKey) {
	s.mx.Lock()
	delete(s.conns, pk)
	s.mx.Unlock()
}

func (s *Server) getConn(pk cipher.PubKey) (*ServerConn, bool) {
	s.mx.RLock()
	l, ok := s.conns[pk]
	s.mx.RUnlock()
	return l, ok
}

func (s *Server) connCount() int {
	s.mx.RLock()
	n := len(s.conns)
	s.mx.RUnlock()
	return n
}

func (s *Server) close() (closed bool, err error) {
	s.doneOnce.Do(func() {
		closed = true
		atomic.StoreInt32(&s.lisDone, 1)

		if err = s.lis.Close(); err != nil {
			return
		}

		s.mx.Lock()
		s.conns = make(map[cipher.PubKey]*ServerConn)
		s.mx.Unlock()
	})

	return closed, err
}

// Close closes the dms_server.
func (s *Server) Close() error {
	closed, err := s.close()
	if !closed {
		return errors.New("server is already closed")
	}
	if err != nil {
		return err
	}

	s.wg.Wait()
	return nil
}

func (s *Server) isLisClosed() bool {
	return atomic.LoadInt32(&s.lisDone) == 1
}

// Serve serves the dmsg_server.
func (s *Server) Serve() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := s.retryUpdateEntry(ctx, TransportHandshakeTimeout); err != nil {
		return fmt.Errorf("updating server's client entry failed with: %s", err)
	}

	s.log.Infof("serving: pk(%s) addr(%s)", s.pk, s.addr)

	for {
		rawConn, err := s.lis.Accept()
		if err != nil {
			// if the listener is closed, it means that this error is not interesting
			// for the outer client
			if s.isLisClosed() {
				return nil
			}
			return err
		}
		s.log.Infof("newConn: %v", rawConn.RemoteAddr())
		conn := NewServerConn(s.log, rawConn, rawConn.RemoteAddr().(*noise.Addr).PK)
		s.setConn(conn)

		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			err := conn.Serve(ctx, s.getConn)
			s.log.Infof("connection with client %s closed: error(%v)", conn.PK(), err)
			s.delConn(conn.PK())
		}()
	}
}

func (s *Server) updateDiscEntry(ctx context.Context) error {
	entry, err := s.dc.Entry(ctx, s.pk)
	if err != nil {
		entry = disc.NewServerEntry(s.pk, 0, s.addr, 10)
		if err := entry.Sign(s.sk); err != nil {
			return err
		}
		return s.dc.SetEntry(ctx, entry)
	}

	entry.Server.Address = s.Addr()
	s.log.Infoln("updatingEntry:", entry)

	return s.dc.UpdateEntry(ctx, s.sk, entry)
}

func (s *Server) retryUpdateEntry(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		if err := s.updateDiscEntry(ctx); err != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				retry := time.Second
				s.log.WithError(err).Warnf("updateEntry failed: trying again in %d second...", retry)
				time.Sleep(retry)
				continue
			}
		}
		return nil
	}
}
