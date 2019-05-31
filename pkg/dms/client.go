package dms

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"sync"
	"time"

	"github.com/prometheus/common/log"
	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/internal/noise"
	"github.com/skycoin/skywire/pkg/cipher"
	"github.com/skycoin/skywire/pkg/messaging-discovery/client"
	"github.com/skycoin/skywire/pkg/transport"
)

var (
	// ErrNoSrv indicate that remote client does not have DelegatedServers in entry.
	ErrNoSrv = errors.New("remote has no DelegatedServers")
	// ErrRejected indicates that ChannelOpen frame was rejected by remote server.
	ErrRejected = errors.New("rejected")
	// ErrChannelClosed indicates that underlying channel is being closed and writes are prohibited.
	ErrChannelClosed = errors.New("channel closed")
	// ErrDeadlineExceeded indicates that read/write operation failed due to timeout.
	ErrDeadlineExceeded = errors.New("deadline exceeded")
	// ErrClientClosed indicates that client is closed and not accepting new connections.
	ErrClientClosed = errors.New("client closed")
)

// Conn represents a connection between a dms.Client and dms.Server from a client's perspective.
type Conn struct {
	log       *logging.Logger
	net.Conn                             // conn to dms server
	local     cipher.PubKey              // local client's pk
	remoteSrv cipher.PubKey              // dms server's public key
	nextID    uint16                     // next unused channel ID
	tps       [math.MaxUint16]*Transport // channels to dms clients
	mx        sync.RWMutex
	wg        sync.WaitGroup
}

func NewConn(log *logging.Logger, conn net.Conn, local, remote cipher.PubKey) *Conn {
	return &Conn{log: log, Conn: conn, local: local, remoteSrv: remote, nextID: 0}
}

func (c *Conn) delTp(id uint16) {
	c.mx.Lock()
	c.tps[id] = nil
	c.mx.Unlock()
}

func (c *Conn) setTp(ch *Transport) {
	c.mx.Lock()
	c.tps[ch.id] = ch
	c.mx.Unlock()
}

func (c *Conn) addTp(ctx context.Context, clientPK cipher.PubKey) (*Transport, error) {
	c.mx.Lock()
	defer c.mx.Unlock()

	for {
		if ch := c.tps[c.nextID]; ch == nil || ch.IsDone() {
			break
		}
		c.nextID += 2

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
	}

	id := c.nextID
	c.nextID = id + 2
	ch := NewTransport(c.Conn, c.local, clientPK, id)
	c.tps[id] = ch
	return ch, nil
}

func (c *Conn) getTp(id uint16) (*Transport, bool) {
	c.mx.RLock()
	tp := c.tps[id]
	c.mx.RUnlock()
	ok := tp != nil && !tp.IsDone()
	return tp, ok
}

// local:  local client pk  (also responding client).
// remote: remote client pk (also initiating client).
func checkRequest(local cipher.PubKey, id uint16, p []byte) (remote cipher.PubKey, ok bool) {
	// server-initiated channels should have odd channel ID
	if isEven(id) {
		return cipher.PubKey{}, false
	}

	// check expected request payload
	initPK, respPK, ok := splitPKs(p)
	if !ok || respPK != local {
		return cipher.PubKey{}, false
	}

	return initPK, true
}

func (c *Conn) Serve(ctx context.Context, accept chan<- *Transport) error {
	c.wg.Add(1)
	defer c.wg.Done()

	log := c.log.WithField("remoteSrv", c.remoteSrv)

	for {
		f, err := readFrame(c.Conn)
		if err != nil {
			return err
		}
		ft, id, p := f.Disassemble()
		tp, ok := c.getTp(id)
		log.Infof("readFrame: frameType(%v) channelID(%v) payloadLen(%v)", ft, id, f.PayLen())

		if !ok {
			c.delTp(id)
			switch ft {
			case RequestType:
				remote, ok := checkRequest(c.local, id, p)
				if !ok {
					_ = writeFrame(c.Conn, MakeFrame(CloseType, id, []byte{0}))
				} else {
					tp = NewTransport(c.Conn, c.local, remote, id)
					c.setTp(tp)
					if err := tp.Handshake(ctx); err != nil {
						return err
					}
					select {
					case accept <- tp:
						log.Infof("channelAccepted: remoteClient(%v) channelID(%v)", tp.remoteClient, tp.id)
					case <-ctx.Done():
						return ctx.Err()
					}
				}
			case CloseType:
			default:
				_ = writeFrame(c.Conn, MakeFrame(CloseType, id, []byte{0}))
			}
		} else if !tp.AwaitRead(f) {
			c.delTp(id)
		}
	}
}

func (c *Conn) DialTransport(ctx context.Context, clientPK cipher.PubKey) (*Transport, error) {
	tp, err := c.addTp(ctx, clientPK)
	if err != nil {
		return nil, err
	}
	return tp, tp.Handshake(ctx)
}

func (c *Conn) Close() error {
	c.log.Infof("closingLink: remoteSrv(%v)", c.remoteSrv)
	c.mx.Lock()
	for _, ch := range c.tps {
		if ch != nil {
			_ = ch.Close()
		}
	}
	err := c.Conn.Close()
	c.mx.Unlock()
	c.wg.Wait()
	return err
}

type Client struct {
	log *logging.Logger

	pk cipher.PubKey
	sk cipher.SecKey
	dc client.APIClient

	conns map[cipher.PubKey]*Conn // conns with messaging servers. Key: pk of server
	mx    sync.RWMutex

	accept chan *Transport
	once   sync.Once
}

func NewClient(pk cipher.PubKey, sk cipher.SecKey, dc client.APIClient) *Client {
	return &Client{
		log:    logging.MustGetLogger("dms_client"),
		pk:     pk,
		sk:     sk,
		dc:     dc,
		conns:  make(map[cipher.PubKey]*Conn),
		accept: make(chan *Transport),
	}
}

func (c *Client) SetLogger(log *logging.Logger) {
	c.log = log
}

func (c *Client) setConn(l *Conn) {
	c.mx.Lock()
	c.conns[l.remoteSrv] = l
	c.mx.Unlock()
}

func (c *Client) delConn(pk cipher.PubKey) {
	c.mx.Lock()
	delete(c.conns, pk)
	c.mx.Unlock()
}

func (c *Client) getConn(pk cipher.PubKey) (*Conn, bool) {
	c.mx.RLock()
	l, ok := c.conns[pk]
	c.mx.RUnlock()
	return l, ok
}

func (c *Client) newConn(ctx context.Context, srvPK cipher.PubKey, addr string) (*Conn, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	ns, err := noise.New(noise.HandshakeXK, noise.Config{
		LocalPK:   c.pk,
		LocalSK:   c.sk,
		RemotePK:  srvPK,
		Initiator: true,
	})
	if err != nil {
		return nil, err
	}
	nc, err := noise.WrapConn(conn, ns, hsTimeout)
	if err != nil {
		return nil, err
	}
	l := NewConn(c.log, nc, c.pk, srvPK)
	go func() {
		if err := l.Serve(ctx, c.accept); err != nil {
			l.log.WithError(err).WithField("srv_pk", l.remoteSrv).Warn("link with server closed")
			c.delConn(l.remoteSrv)
		}
	}()
	return l, nil
}

func (c *Client) InitiateServers(ctx context.Context, n int) error {
	if n == 0 {
		return nil
	}
	var entries []*client.Entry
	var err error
	for {
		if entries, err = c.dc.AvailableServers(ctx); err != nil || len(entries) == 0 {
			select {
			case <-ctx.Done():
				return fmt.Errorf("messaging servers are not available: %s", err)
			case <-time.Tick(time.Second):
				continue
			}
		}
		break
	}
	for _, entry := range entries {
		if len(c.conns) > n {
			break
		}
		conn, err := c.newConn(ctx, entry.Static, entry.Server.Address)
		if err != nil {
			log.Warnf("Failed to connect to server %s: %s", entry.Static, err)
			continue
		}
		c.conns[conn.remoteSrv] = conn
	}
	if len(c.conns) == 0 {
		return fmt.Errorf("servers are not available: all servers failed")
	}
	if err := c.updateDiscEntry(ctx); err != nil {
		return fmt.Errorf("updating client's discovery entry failed with: %s", err)
	}
	return nil
}

func (c *Client) findConn(ctx context.Context, srvPKs []cipher.PubKey) (*Conn, error) {
	for _, srvPK := range srvPKs {
		conn, ok := c.conns[srvPK]
		if !ok {
			continue
		}
		return conn, nil
	}
	for _, srvPK := range srvPKs {
		entry, err := c.dc.Entry(ctx, srvPK)
		if err != nil {
			return nil, fmt.Errorf("get server failure: %s", err)
		}
		conn, err := c.newConn(ctx, entry.Static, entry.Server.Address)
		if err != nil {
			log.Warnf("Failed to connect to server %s: %s", entry.Static, err)
			continue
		}
		c.conns[conn.remoteSrv] = conn
		return conn, nil
	}
	return nil, ErrNoSrv
}

func (c *Client) updateDiscEntry(ctx context.Context) error {
	log.Info("updatingEntry")
	var srvPKs []cipher.PubKey
	c.mx.RLock()
	for pk := range c.conns {
		srvPKs = append(srvPKs, pk)
	}
	c.mx.RUnlock()
	entry, err := c.dc.Entry(ctx, c.pk)
	if err != nil {
		entry = client.NewClientEntry(c.pk, 0, srvPKs)
		if err := entry.Sign(c.sk); err != nil {
			return err
		}
		return c.dc.SetEntry(ctx, entry)
	}
	entry.Client.DelegatedServers = srvPKs
	return c.dc.UpdateEntry(ctx, c.sk, entry)
}

func (c *Client) Accept(ctx context.Context) (transport.Transport, error) {
	select {
	case tp, ok := <-c.accept:
		if !ok {
			return nil, ErrClientClosed
		}
		return tp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *Client) Dial(ctx context.Context, remote cipher.PubKey) (transport.Transport, error) {
	c.mx.Lock()
	defer c.mx.Unlock()

	entry, err := c.dc.Entry(ctx, remote)
	if err != nil {
		return nil, fmt.Errorf("get entry failure: %s", err)
	}
	if len(entry.Client.DelegatedServers) == 0 {
		return nil, ErrNoSrv
	}
	conn, err := c.findConn(ctx, entry.Client.DelegatedServers)
	if err != nil {
		return nil, err
	}
	return conn.DialTransport(ctx, remote)
}

func (c *Client) Local() cipher.PubKey {
	return c.pk
}

func (c *Client) Type() string {
	return Type
}

// TODO(evaninjin): proper error handling.
func (c *Client) Close() error {
	c.mx.Lock()
	defer c.mx.Unlock()

	for _, link := range c.conns {
		_ = link.Close()
	}
	c.conns = make(map[cipher.PubKey]*Conn)
	c.once.Do(func() {
		close(c.accept)
	})
	return nil
}
