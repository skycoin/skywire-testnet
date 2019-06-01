package dms

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"sync"

	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/internal/noise"
	"github.com/skycoin/skywire/pkg/cipher"
	"github.com/skycoin/skywire/pkg/messaging-discovery/client"
)

// NextConn provides information on the next connection.
type NextConn struct {
	l  *ServerConn
	id uint16
}

func (r *NextConn) writeFrame(ft FrameType, p []byte) error {
	if err := writeFrame(r.l.Conn, MakeFrame(ft, r.id, p)); err != nil {
		go r.l.Close()
		return err
	}
	return nil
}

// ServerConn is a connection between a dms.Server and a dms.Client from a server's perspective.
type ServerConn struct {
	log *logging.Logger

	net.Conn
	remoteClient cipher.PubKey

	nextRespID uint16
	nextLinks  [math.MaxUint16]*NextConn
	mx         sync.RWMutex
}

// NewServerConn creates a new connection from the perspective of a dms_server.
func NewServerConn(log *logging.Logger, conn net.Conn, remoteClient cipher.PubKey) *ServerConn {
	return &ServerConn{log: log, Conn: conn, remoteClient: remoteClient, nextRespID: randID(false)}
}

func (c *ServerConn) delNext(id uint16) {
	c.mx.Lock()
	c.nextLinks[id] = nil
	c.mx.Unlock()
}

func (c *ServerConn) setNext(id uint16, r *NextConn) {
	c.mx.Lock()
	c.nextLinks[id] = r
	c.mx.Unlock()
}

func (c *ServerConn) getNext(id uint16) (*NextConn, bool) {
	c.mx.RLock()
	r := c.nextLinks[id]
	c.mx.RUnlock()
	return r, r != nil
}

func (c *ServerConn) addNext(ctx context.Context, r *NextConn) (uint16, error) {
	c.mx.Lock()
	defer c.mx.Unlock()

	for {
		if r := c.nextLinks[c.nextRespID]; r == nil {
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
	c.nextLinks[id] = r
	return id, nil
}

// PK returns the remote dms_client's public key.
func (c *ServerConn) PK() cipher.PubKey {
	return c.remoteClient
}

type getConnFunc func(pk cipher.PubKey) (*ServerConn, bool)

// Serve handles (and forwards when necessary) incoming frames.
func (c *ServerConn) Serve(ctx context.Context, getConn getConnFunc) error {
	log := c.log.WithField("client", c.remoteClient)

	go func() {
		<-ctx.Done()
		c.Conn.Close()
	}()

	for {
		f, err := readFrame(c.Conn)
		if err != nil {
			return fmt.Errorf("failed to read frame: %s", err)
		}

		ft, id, p := f.Disassemble()
		log.Infof("readFrame: frameType(%v) srcID(%v) pLen(%v)", ft, id, len(p))

		switch ft {
		case RequestType:
			go func() {
				ctx, cancel := context.WithTimeout(ctx, hsTimeout)
				defer cancel()

				_, why, ok := c.handleRequest(ctx, getConn, id, p)
				if !ok {
					if err := c.delChan(id, why); err != nil {
						c.Conn.Close()
					}
				}
			}()

		case AcceptType, FwdType, AckType, CloseType:
			next, why, ok := c.forwardFrame(ft, id, p)
			if !ok {
				// Delete channel (and associations) on failure.
				if err := c.delChan(id, why); err != nil {
					return err
				}
				continue
			}

			// On success, if Close frame, delete the associations.
			if ft == CloseType {
				c.delNext(id)
				next.l.delNext(next.id)
			}

		default:
			// Unknown frame type.
			if err := c.delChan(id, 0); err != nil {
				return err
			}
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
	respID, err := respL.addNext(ctx, &NextConn{l: c, id: id})
	if err != nil {
		return nil, 0, false
	}
	next := &NextConn{l: respL, id: respID}
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

	pk   cipher.PubKey
	sk   cipher.SecKey
	addr string
	dc   client.APIClient

	lis   net.Listener
	conns map[cipher.PubKey]*ServerConn
	mx    sync.RWMutex

	wg sync.WaitGroup
}

// NewServer creates a new dms_server.
func NewServer(pk cipher.PubKey, sk cipher.SecKey, addr string, dc client.APIClient) *Server {
	return &Server{
		log:   logging.MustGetLogger("dms_server"),
		pk:    pk,
		sk:    sk,
		addr:  addr,
		dc:    dc,
		conns: make(map[cipher.PubKey]*ServerConn),
	}
}

// SetLogger set's the logger.
func (s *Server) SetLogger(log *logging.Logger) {
	s.log = log
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

// Close closes the dms_server.
func (s *Server) Close() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New("listener has not started")
		}
	}()
	if err = s.lis.Close(); err != nil {
		return err
	}

	s.mx.Lock()
	s.conns = make(map[cipher.PubKey]*ServerConn)
	s.mx.Unlock()

	s.wg.Wait()
	return nil
}

// ListenAndServe serves the dms_server.
func (s *Server) ListenAndServe(addr string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	if err := s.updateDiscEntry(ctx); err != nil {
		return fmt.Errorf("updating server's discovery entry failed with: %s", err)
	}

	s.log.Infof("serving: pk(%s) addr(%s)", s.pk, lis.Addr())
	lis = noise.WrapListener(lis, s.pk, s.sk, false, noise.HandshakeXK)
	s.lis = lis
	for {
		rawConn, err := lis.Accept()
		if err != nil {
			if err == io.ErrUnexpectedEOF {
				continue
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
	s.log.Info("updating server discovery entry...")
	entry, err := s.dc.Entry(ctx, s.pk)
	if err != nil {
		entry = client.NewServerEntry(s.pk, 0, s.addr, 10)
		if err := entry.Sign(s.sk); err != nil {
			return err
		}
		return s.dc.SetEntry(ctx, entry)
	}
	entry.Server.Address = s.addr
	return s.dc.UpdateEntry(ctx, s.sk, entry)
}
