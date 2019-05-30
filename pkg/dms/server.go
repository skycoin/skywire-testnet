package dms

import (
	"context"
	"fmt"
	"github.com/prometheus/common/log"
	"github.com/skycoin/skycoin/src/util/logging"
	"io"
	"math"
	"net"
	"sync"

	"github.com/skycoin/skywire/internal/noise"
	"github.com/skycoin/skywire/pkg/cipher"
	"github.com/skycoin/skywire/pkg/messaging-discovery/client"
)

type Dispatch struct {
	Frame
	Src *ServerLink
}

// Relation relates two channels together
// Key: srcPK, srcID
type Relation struct {
	Link *ServerLink
	ID   uint16
}

// ServerLink from a server's perspective.
type ServerLink struct {
	log *logging.Logger

	net.Conn
	remoteClient cipher.PubKey

	nextID    uint16
	relations [math.MaxUint16]*Relation
	mx        sync.RWMutex
}

func NewServerLink(log *logging.Logger, conn net.Conn, remoteClient cipher.PubKey) *ServerLink {
	return &ServerLink{log: log, Conn: conn, remoteClient: remoteClient, nextID: 1}
}

func (l *ServerLink) delRelation(id uint16) {
	l.mx.Lock()
	l.relations[id] = nil
	l.mx.Unlock()
}

func (l *ServerLink) setRelation(id uint16, r *Relation) {
	l.mx.Lock()
	l.relations[id] = r
	l.mx.Unlock()
}

func (l *ServerLink) getRelation(id uint16) (*Relation, bool) {
	l.mx.RLock()
	r := l.relations[id]
	l.mx.RUnlock()
	return r, r != nil
}

func (l *ServerLink) addRelation(ctx context.Context, r *Relation) (uint16, error) {
	l.mx.Lock()
	defer l.mx.Unlock()

	for {
		if r := l.relations[l.nextID]; r == nil {
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
	l.relations[id] = r
	return id, nil
}

func (l *ServerLink) PK() cipher.PubKey {
	return l.remoteClient
}

func (l *ServerLink) Serve(ctx context.Context, in chan<- Dispatch) error {
	log := l.log.WithField("remoteClient", l.remoteClient)
	for {
		f, err := readFrame(l.Conn)
		if err != nil {
			log.WithError(err).Errorf("readFrame failed")
			return err
		}
		select {
		case in <- Dispatch{Frame: f, Src: l}:
		case <-ctx.Done():
			log.Info("context done")
			return ctx.Err()
		}
	}
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

	in chan Dispatch
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
		in:    make(chan Dispatch),
	}
}

func (s *Server) SetLogger(log *logging.Logger) {
	s.log = log
}

func (s *Server) setLink(l *ServerLink) {
	s.mx.Lock()
	if l, ok := s.links[l.remoteClient]; ok {
		l.Close()
	}
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

func (s *Server) Close() error {
	if err := s.lis.Close(); err != nil {
		return err
	}

	s.mx.Lock()
	defer s.mx.Unlock()

	for _, l := range s.links {
		_ = l.Close()
	}
	s.links = make(map[cipher.PubKey]*ServerLink)
	s.wg.Wait()
	close(s.in)
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

	go s.serve()

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
			_ = link.Serve(ctx, s.in) // TODO: log error.
		}()
	}
}

func (s *Server) serve() {

	for d := range s.in {
		// request.
		ft, srcID, p := d.Frame.Disassemble()
		s.log.Infof("readFrame: frameType(%v) srcID(%v) pLen(%v)", ft, srcID, len(p))

		// response.
		why, ok := byte(0), false

		switch ft {
		case RequestType:
			ctx, cancel := context.WithTimeout(context.Background(), hsTimeout)
			why, ok = s.handleRequest(ctx, d.Src, srcID, p)
			cancel()
		case AcceptType:
			why, ok = s.handleAccept(d.Src, srcID, p)
		case CloseType:
			why, ok = s.handleClose(d.Src, srcID, p)
		case FwdType:
			why, ok = s.handleFwd(d.Src, srcID, p)
		default:
			why, ok = 0, false
		}
		if !ok {
			f := MakeFrame(CloseType, srcID, []byte{why})
			_ = writeFrame(d.Src.Conn, f)
			d.Src.delRelation(srcID)
		}
	}
}

func (s *Server) handleRequest(ctx context.Context, src *ServerLink, srcID uint16, p []byte) (byte, bool) {
	initPK, respPK, ok := splitPKs(p)
	fmt.Println(initPK, respPK, ok)
	if !ok || initPK != src.PK() {
		return 0, false
	}
	dst, ok := s.getLink(respPK)
	if !ok {
		return 0, false
	}

	// Set relations.
	dstID, err := dst.addRelation(ctx, &Relation{Link: src, ID: srcID})
	if err != nil {
		return 0, false
	}
	src.setRelation(srcID, &Relation{Link: dst, ID: dstID})

	// send to dst
	f := MakeFrame(RequestType, dstID, p)
	if err := writeFrame(dst, f); err != nil {
		_ = dst.Close()
		s.delLink(dst.PK())
		return 0, false
	}

	return 0, true
}

func (s *Server) handleAccept(src *ServerLink, srcID uint16, p []byte) (byte, bool) {
	r, ok := src.getRelation(srcID)
	if !ok {
		return 0, false
	}
	dst, dstID := r.Link, r.ID
	if err := writeFrame(dst.Conn, MakeFrame(AcceptType, dstID, p)); err != nil {
		_ = dst.Close()
		s.delLink(dst.PK())
		return 0, false
	}
	return 0, true
}

func (s *Server) handleClose(src *ServerLink, srcID uint16, p []byte) (byte, bool) {
	r, ok := src.getRelation(srcID)
	if !ok {
		return 0, false
	}
	dst, dstID := r.Link, r.ID
	if err := writeFrame(dst.Conn, MakeFrame(CloseType, dstID, p)); err != nil {
		_ = dst.Close()
		s.delLink(dst.PK())
		return 0, false
	}
	dst.delRelation(dstID)
	src.delRelation(srcID)
	return 0, true
}

func (s *Server) handleFwd(src *ServerLink, srcID uint16, p []byte) (byte, bool) {
	r, ok := src.getRelation(srcID)
	if !ok {
		return 0, false
	}
	dst, dstID := r.Link, r.ID
	if err := writeFrame(dst.Conn, MakeFrame(FwdType, dstID, p)); err != nil {
		_ = dst.Close()
		s.delLink(dst.PK())
		return 0, false
	}
	return 0, true
}

func (s *Server) updateDiscEntry(ctx context.Context) error {
	log.Info("updating server discovery entry...")
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
