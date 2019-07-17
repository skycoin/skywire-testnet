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

// NextConn provides information on the next connection.
type NextConn struct {
	conn *ServerConn
	id   uint16
}

func (r *NextConn) writeFrame(ft FrameType, p []byte) error {
	if err := writeFrame(r.conn.Conn, MakeFrame(ft, r.id, p)); err != nil {
		go r.conn.Close()
		return err
	}
	return nil
}

// ServerConn is a connection between a dmsg.Server and a dmsg.Client from a server's perspective.
type ServerConn struct {
	log *logging.Logger

	net.Conn
	remoteClient cipher.PubKey

	nextRespID uint16
	nextConns  map[uint16]*NextConn
	mx         sync.RWMutex
}

// NewServerConn creates a new connection from the perspective of a dms_server.
func NewServerConn(log *logging.Logger, conn net.Conn, remoteClient cipher.PubKey) *ServerConn {
	return &ServerConn{
		log:          log,
		Conn:         conn,
		remoteClient: remoteClient,
		nextRespID:   randID(false),
		nextConns:    make(map[uint16]*NextConn),
	}
}

func (c *ServerConn) delNext(id uint16) {
	c.mx.Lock()
	c.nextConns[id] = nil
	c.mx.Unlock()
}

func (c *ServerConn) setNext(id uint16, r *NextConn) {
	c.mx.Lock()
	c.nextConns[id] = r
	c.mx.Unlock()
}

func (c *ServerConn) getNext(id uint16) (*NextConn, bool) {
	c.mx.RLock()
	r := c.nextConns[id]
	c.mx.RUnlock()
	return r, r != nil
}

func (c *ServerConn) addNext(ctx context.Context, r *NextConn) (uint16, error) {
	c.mx.Lock()
	defer c.mx.Unlock()

	for {
		if r := c.nextConns[c.nextRespID]; r == nil {
			break
		}
		c.nextRespID += 2

		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}
	}

	id := c.nextRespID
	c.nextRespID = id + 2
	c.nextConns[id] = r
	return id, nil
}

// PK returns the remote dms_client's public key.
func (c *ServerConn) PK() cipher.PubKey {
	return c.remoteClient
}

type getConnFunc func(pk cipher.PubKey) (*ServerConn, bool)

// Serve handles (and forwards when necessary) incoming frames.
func (c *ServerConn) Serve(ctx context.Context, getConn getConnFunc) (err error) {
	go func() {
		<-ctx.Done()
		c.Conn.Close()
	}()

	log := c.log.WithField("srcClient", c.remoteClient)
	defer func() {
		log.WithError(err).WithField("connCount", decrementServeCount()).Infoln("ClosingConn")
		c.Conn.Close()
	}()
	log.WithField("connCount", incrementServeCount()).Infoln("ServingConn")

	err = c.writeOK()
	if err != nil {
		return fmt.Errorf("sending OK failed: %s", err)
	}

	for {
		f, err := readFrame(c.Conn)
		if err != nil {
			return fmt.Errorf("read failed: %s", err)
		}
		log = log.WithField("received", f)

		ft, id, p := f.Disassemble()

		switch ft {
		case RequestType:
			ctx, cancel := context.WithTimeout(ctx, TransportHandshakeTimeout)
			_, why, ok := c.handleRequest(ctx, getConn, id, p)
			cancel()
			if !ok {
				log.Debugln("FrameRejected: Erroneous request or unresponsive dstClient.")
				if err := c.delChan(id, why); err != nil {
					return err
				}
			}
			log.Debugln("FrameForwarded")

		case AcceptType, FwdType, AckType, CloseType:
			next, why, ok := c.forwardFrame(ft, id, p)
			if !ok {
				log.Debugln("FrameRejected: Failed to forward to dstClient.")
				// Delete channel (and associations) on failure.
				if err := c.delChan(id, why); err != nil {
					return err
				}
				continue
			}
			log.Debugln("FrameForwarded")

			// On success, if Close frame, delete the associations.
			if ft == CloseType {
				c.delNext(id)
				next.conn.delNext(next.id)
			}

		default:
			log.Debugln("FrameRejected: Unknown frame type.")
			// Unknown frame type.
			return errors.New("unknown frame of type received")
		}
	}
}

func (c *ServerConn) delChan(id uint16, why byte) error {
	c.delNext(id)
	if err := writeFrame(c.Conn, MakeFrame(CloseType, id, []byte{why})); err != nil {
		return fmt.Errorf("failed to write frame: %s", err)
	}
	return nil
}

func (c *ServerConn) writeOK() error {
	if err := writeFrame(c.Conn, MakeFrame(OkType, 0, nil)); err != nil {
		return err
	}
	return nil
}

func (c *ServerConn) forwardFrame(ft FrameType, id uint16, p []byte) (*NextConn, byte, bool) { //nolint:unparam
	next, ok := c.getNext(id)
	if !ok {
		return next, 0, false
	}
	if err := next.writeFrame(ft, p); err != nil {
		return next, 0, false
	}
	return next, 0, true
}

func (c *ServerConn) handleRequest(ctx context.Context, getLink getConnFunc, id uint16, p []byte) (*NextConn, byte, bool) { //nolint:unparam
	initPK, respPK, ok := splitPKs(p)
	if !ok || initPK != c.PK() {
		return nil, 0, false
	}
	respL, ok := getLink(respPK)
	if !ok {
		return nil, 0, false
	}

	// set next relations.
	respID, err := respL.addNext(ctx, &NextConn{conn: c, id: id})
	if err != nil {
		return nil, 0, false
	}
	next := &NextConn{conn: respL, id: respID}
	c.setNext(id, next)

	// forward to responding client.
	if err := next.writeFrame(RequestType, p); err != nil {
		return next, 0, false
	}
	return next, 0, true
}

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
