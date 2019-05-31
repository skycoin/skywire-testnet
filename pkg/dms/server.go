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

type Dispatch struct {
	Frame
	Src *ServerLink
}

// NextLink provides information on the next link.
type NextLink struct {
	l  *ServerLink
	id uint16
}

func (r *NextLink) writeFrame(ft FrameType, p []byte) error {
	if err := writeFrame(r.l.Conn, MakeFrame(ft, r.id, p)); err != nil {
		go r.l.Close()
		return err
	}
	return nil
}

// ServerLink from a server's perspective.
type ServerLink struct {
	log *logging.Logger

	net.Conn
	remoteClient cipher.PubKey

	nextID    uint16
	nextLinks [math.MaxUint16]*NextLink
	mx        sync.RWMutex
}

func NewServerLink(log *logging.Logger, conn net.Conn, remoteClient cipher.PubKey) *ServerLink {
	return &ServerLink{log: log, Conn: conn, remoteClient: remoteClient, nextID: 1}
}

func (l *ServerLink) delNext(id uint16) {
	l.mx.Lock()
	l.nextLinks[id] = nil
	l.mx.Unlock()
}

func (l *ServerLink) setNext(id uint16, r *NextLink) {
	l.mx.Lock()
	l.nextLinks[id] = r
	l.mx.Unlock()
}

func (l *ServerLink) getNext(id uint16) (*NextLink, bool) {
	l.mx.RLock()
	r := l.nextLinks[id]
	l.mx.RUnlock()
	return r, r != nil
}

func (l *ServerLink) addNext(ctx context.Context, r *NextLink) (uint16, error) {
	l.mx.Lock()
	defer l.mx.Unlock()

	for {
		if r := l.nextLinks[l.nextID]; r == nil {
			break
		}
		l.nextID += 2

		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}
	}

	id := l.nextID
	l.nextID = id + 2
	l.nextLinks[id] = r
	return id, nil
}

func (l *ServerLink) PK() cipher.PubKey {
	return l.remoteClient
}

type getLinkFunc func(pk cipher.PubKey) (*ServerLink, bool)

func (l *ServerLink) Serve(ctx context.Context, getLink getLinkFunc) error {
	go func() {
		select {
		case <-ctx.Done():
			l.Conn.Close()
		}
	}()

	log := l.log.WithField("client", l.remoteClient)
	for {
		f, err := readFrame(l.Conn)
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

				_, why, ok := l.handleRequest(ctx, getLink, id, p)
				if !ok {
					if err := l.delChan(id, why); err != nil {
						l.Conn.Close()
					}
				}
			}()

		case AcceptType, SendType, CloseType:
			next, why, ok := l.forwardFrame(ft, id, p)
			if !ok {
				// Delete channel (and associations) on failure.
				if err := l.delChan(id, why); err != nil {
					return err
				}
				continue
			}

			// On success, if Close frame, delete the associations.
			if ft == CloseType {
				l.delNext(id)
				next.l.delNext(next.id)
			}

		default:
			// Unknown frame type.
			if err := l.delChan(id, 0); err != nil {
				return err
			}
		}
	}
}

func (l *ServerLink) delChan(id uint16, why byte) error {
	l.delNext(id)
	if err := writeFrame(l.Conn, MakeFrame(CloseType, id, []byte{why})); err != nil {
		return fmt.Errorf("failed to write frame: %s", err)
	}
	return nil
}

func (l *ServerLink) forwardFrame(ft FrameType, id uint16, p []byte) (*NextLink, byte, bool) {
	next, ok := l.getNext(id)
	if !ok {
		return next, 0, false
	}
	if err := next.writeFrame(ft, p); err != nil {
		return next, 0, false
	}
	return next, 0, true
}

func (l *ServerLink) handleRequest(ctx context.Context, getLink getLinkFunc, id uint16, p []byte) (*NextLink, byte, bool) {
	initPK, respPK, ok := splitPKs(p)
	if !ok || initPK != l.PK() {
		return nil, 0, false
	}
	respL, ok := getLink(respPK)
	if !ok {
		return nil, 0, false
	}

	// set next relations.
	respID, err := respL.addNext(ctx, &NextLink{l: l, id: id})
	if err != nil {
		return nil, 0, false
	}
	next := &NextLink{l: respL, id: respID}
	l.setNext(id, next)

	// forward to responding client.
	if err := next.writeFrame(RequestType, p); err != nil {
		return next, 0, false
	}
	return next, 0, true
}

type Server struct {
	log *logging.Logger

	pk   cipher.PubKey
	sk   cipher.SecKey
	addr string
	dc   client.APIClient

	lis   net.Listener
	links map[cipher.PubKey]*ServerLink
	mx    sync.RWMutex

	wg sync.WaitGroup
}

func NewServer(pk cipher.PubKey, sk cipher.SecKey, addr string, dc client.APIClient) *Server {
	return &Server{
		log:   logging.MustGetLogger("dms_server"),
		pk:    pk,
		sk:    sk,
		addr:  addr,
		dc:    dc,
		links: make(map[cipher.PubKey]*ServerLink),
	}
}

func (s *Server) SetLogger(log *logging.Logger) {
	s.log = log
}

func (s *Server) setLink(l *ServerLink) {
	s.mx.Lock()
	s.links[l.remoteClient] = l
	s.mx.Unlock()
}

func (s *Server) delLink(pk cipher.PubKey) {
	s.mx.Lock()
	delete(s.links, pk)
	s.mx.Unlock()
}

func (s *Server) getLink(pk cipher.PubKey) (*ServerLink, bool) {
	s.mx.RLock()
	l, ok := s.links[pk]
	s.mx.RUnlock()
	return l, ok
}

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
	s.links = make(map[cipher.PubKey]*ServerLink)
	s.mx.Unlock()

	s.wg.Wait()
	return nil
}

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
		conn, err := lis.Accept()
		if err != nil {
			if err == io.ErrUnexpectedEOF {
				continue
			}
			return err
		}
		s.log.Infof("newLink: %v", conn.RemoteAddr())
		link := NewServerLink(s.log, conn, conn.RemoteAddr().(*noise.Addr).PK)
		s.setLink(link)

		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			_ = link.Serve(ctx, s.getLink) // TODO: log error.
			s.delLink(link.PK())
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
