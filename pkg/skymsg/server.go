package skymsg

import (
	"context"
	"fmt"
	"github.com/skycoin/skywire/internal/noise"
	"github.com/skycoin/skywire/pkg/cipher"
	"github.com/skycoin/skywire/pkg/messaging-discovery/client"
	"math"
	"net"
	"sync"
)


// Relation relates two channels together
// Key: srcPK, srcID
type Relation struct {
	NextPK cipher.PubKey
	NextID uint16
}

// ServerLink from a server's perspective.
type ServerLink struct {
	net.Conn
	local        cipher.PubKey
	remoteClient cipher.PubKey
	nextID       uint16
	relations    [math.MaxUint16]*Relation // Key: src_id
	mx           sync.RWMutex
}

func NewServerLink(conn net.Conn, local, remoteClient cipher.PubKey) *ServerLink {
	return &ServerLink{Conn: conn, local: local, remoteClient: remoteClient, nextID: 1}
}

type Server struct {
	pk   cipher.PubKey
	sk   cipher.SecKey
	addr string
	dc   client.APIClient
	links map[cipher.PubKey]*ServerLink
	mx    sync.RWMutex
}

func NewServer(pk cipher.PubKey, sk cipher.SecKey, addr string, dc client.APIClient) *Server {
	return &Server{
		pk:    pk,
		sk:    sk,
		addr:  addr,
		dc:    dc,
		links: make(map[cipher.PubKey]*ServerLink),
	}
}

func (s *Server) ListenAndServe(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	if err := s.updateDiscEntry(context.Background()); err != nil {
		return fmt.Errorf("updating server's discovery entry failed with: %s", err)
	}

	log.Infof("serving with: pk(%s) addr(%s)", s.pk, lis.Addr())
	lis = noise.WrapListener(lis, s.pk, s.sk, false, noise.HandshakeXK)
	for {
		conn, err := lis.Accept()
		if err != nil {
			return err
		}

	}
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