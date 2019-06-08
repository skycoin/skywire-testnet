package dmsg

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"sync"
	"time"

	"github.com/skycoin/skywire/internal/ioutil"

	"github.com/sirupsen/logrus"

	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/internal/noise"
	"github.com/skycoin/skywire/pkg/cipher"
	"github.com/skycoin/skywire/pkg/messaging-discovery/client"
	"github.com/skycoin/skywire/pkg/transport"
)

var (
	// ErrNoSrv indicate that remote client does not have DelegatedServers in entry.
	ErrNoSrv = errors.New("remote has no DelegatedServers")
	// ErrClientClosed indicates that client is closed and not accepting new connections.
	ErrClientClosed = errors.New("client closed")
)

type dialWaiter struct {
	tp     *Transport
	accept chan bool
}

func makeDialWaiter(tp *Transport) *dialWaiter {
	return &dialWaiter{tp: tp, accept: make(chan bool)}
}

// ClientConn represents a connection between a dmsg.Client and dmsg.Server from a client's perspective.
type ClientConn struct {
	log *logging.Logger

	net.Conn                // conn to dmsg server
	local     cipher.PubKey // local client's pk
	remoteSrv cipher.PubKey // dmsg server's public key

	// nextInitID keeps track of unused tp_ids to assign a future locally-initiated tp.
	// locally-initiated tps use an even tp_id between local and intermediary dms_server.
	nextInitID uint16

	// Transports: map of transports to remote dms_clients (key: tp_id, val: transport).
	tps [math.MaxUint16 + 1]*Transport

	// Dial Waiters: locally-initiated transports awaiting remote confirmation.
	dws map[uint16]*dialWaiter

	mx sync.RWMutex // to protect tps and dws

	done chan struct{}
	once sync.Once
	wg   sync.WaitGroup
}

// NewClientConn creates a new ClientConn.
func NewClientConn(log *logging.Logger, conn net.Conn, local, remote cipher.PubKey) *ClientConn {
	cc := &ClientConn{
		log:        log,
		Conn:       conn,
		local:      local,
		remoteSrv:  remote,
		nextInitID: randID(true),
		dws:        make(map[uint16]*dialWaiter),
		done:       make(chan struct{}),
	}
	cc.wg.Add(1)
	return cc
}

func (c *ClientConn) nextID(ctx context.Context) (uint16, error) {
	for {
		select {
		case <-c.done:
			return 0, ErrClientClosed
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
			if ch := c.tps[c.nextInitID]; ch != nil && !ch.IsClosed() {
				c.nextInitID += 2
				continue
			}
			c.tps[c.nextInitID] = nil
			id := c.nextInitID
			c.nextInitID = id + 2
			return id, nil
		}
	}
}

// add dial waiter
func (c *ClientConn) DialTransport(ctx context.Context, clientPK cipher.PubKey) (*Transport, error) {
	var (
		id uint16
		tp *Transport
		dw *dialWaiter
	)

	prepareWaiter := func() (err error) {
		c.mx.Lock()
		defer c.mx.Unlock()

		if id, err = c.nextID(ctx); err != nil {
			return err
		}
		tp = NewTransport(c.Conn, c.log, c.local, clientPK, id)
		dw = makeDialWaiter(tp)
		c.dws[id] = dw
		return nil
	}

	startWaiting := func() error {
		if err := tp.WriteRequest(); err != nil {
			return err
		}
		select {
		case <-c.done:
			tp.close()
			return io.ErrClosedPipe
		case <-ctx.Done():
			tp.close()
			return ctx.Err()
		case ok := <-dw.accept:
			if !ok {
				tp.close()
				return ErrRequestRejected
			}
			return nil
		}
	}

	stopWaiting := func() {
		c.mx.Lock()
		defer c.mx.Unlock()

		for {
			select {
			case <-dw.accept:
				continue
			default:
				close(dw.accept)
				delete(c.dws, id)
				return
			}
		}
	}

	defer stopWaiting()

	if err := prepareWaiter(); err != nil {
		return nil, err
	}
	if err := startWaiting(); err != nil {
		return nil, err
	}
	return tp, nil
}

func (c *ClientConn) completeDial(id uint16, accept bool) error {
	c.mx.RLock()
	defer c.mx.RUnlock()

	dw, ok := c.dws[id]
	if !ok {
		return errors.New("failed to complete dial: tp_id not found")
	}

	select {
	case dw.accept <- accept:
		if accept {
			c.tps[dw.tp.id] = dw.tp
		}
		return nil
	default:
		return errors.New("failed to complete dial: dial canceled locally")
	}
}

func (c *ClientConn) setTp(tp *Transport) {
	c.mx.Lock()
	c.tps[tp.id] = tp
	c.mx.Unlock()
}

func (c *ClientConn) delTp(id uint16) {
	c.mx.Lock()
	c.tps[id] = nil
	c.mx.Unlock()
}

func (c *ClientConn) getTp(id uint16) (*Transport, bool) {
	c.mx.RLock()
	tp := c.tps[id]
	c.mx.RUnlock()
	ok := tp != nil && !tp.IsClosed()
	return tp, ok
}

func (c *ClientConn) handleRequestFrame(ctx context.Context, accept chan<- *Transport, id uint16, p []byte) (cipher.PubKey, error) {
	// remotely-initiated tps should:
	// - have a payload structured as 'init_pk:resp_pk'.
	// - resp_pk should be of local client.
	// - use an odd tp_id with the intermediary dms_server.
	initPK, respPK, ok := splitPKs(p)
	if !ok || respPK != c.local || isInitiatorID(id) {
		if err := writeCloseFrame(c.Conn, id, 0); err != nil {
			return initPK, err
		}
		return initPK, ErrRequestCheckFailed
	}

	tp := NewTransport(c.Conn, c.log, c.local, initPK, id)
	if err := tp.WriteAccept(); err != nil {
		return initPK, err
	}
	c.setTp(tp)

	select {
	case <-c.done:
		if err := writeCloseFrame(c.Conn, id, 0); err != nil {
			return initPK, err
		}
		return initPK, ErrClientClosed
	case <-ctx.Done():
		if err := writeCloseFrame(c.Conn, id, 0); err != nil {
			return initPK, err
		}
		return initPK, ctx.Err()
	case accept <- tp:
	}
	return initPK, nil
}

func (c *ClientConn) handleAcceptFrame(id uint16, p []byte) (cipher.PubKey, error) {
	// locally-initiated tps should:
	initPK, respPK, ok := splitPKs(p)
	if !ok || initPK != c.local || !isInitiatorID(id) {
		_ = c.completeDial(id, false) //nolint:errcheck
		if err := writeCloseFrame(c.Conn, id, 0); err != nil {
			return respPK, err
		}
		return respPK, ErrRequestRejected
	}
	return respPK, c.completeDial(id, true)
}

// Serve handles incoming frames.
// Remote-initiated tps that are successfully created are pushing into 'accept' and exposed via 'Client.Accept()'.
func (c *ClientConn) Serve(ctx context.Context, accept chan<- *Transport) (err error) {
	defer c.wg.Done()

	log := c.log.WithField("remoteServer", c.remoteSrv)

	log.WithField("connCount", incrementServeCount()).
		Infoln("ServingConn")
	defer func() {
		log.WithError(err).
			WithField("connCount", decrementServeCount()).
			Infoln("ClosingConn")
	}()

	closeConn := func(log *logrus.Entry) {
		log.WithError(c.Close()).Warn("ClosingConnection")
	}

	for {
		f, err := readFrame(c.Conn)
		if err != nil {
			return fmt.Errorf("read failed: %s", err)
		}
		log = log.WithField("received", f)

		ft, id, p := f.Disassemble()

		// If tp of tp_id exists, attempt to forward frame to tp.
		// delete tp on any failure.
		if tp, ok := c.getTp(id); ok {
			log = log.WithField("remoteClient", tp.remote)

			switch ft {
			case RequestType:
				log.Infoln("TransportRejected: ID already occupied, malicious server.")
				closeConn(log)

			case CloseType:
				log.Infoln("CloseTransport")
				tp.close()
				c.delTp(tp.id)

			case AckType:
				if len(p) != 2 {
					log.Warnln("AckRejected: Invalid sequence.")
					tp.close()
					c.delTp(tp.id)
					if err := writeCloseFrame(c.Conn, id, 0); err != nil {
						return err
					}
					continue
				}
				if err := tp.InjectAck(ioutil.DecodeUint16Seq(p)); err != nil {
					log.WithError(err).Warnln("AckRejected")
					continue
				}
				log.Infoln("AckInjected")

			case FwdType:
				if len(p) < 2 {
					log.Warnln("FwdRejected: Invalid frame.")
					if err := writeCloseFrame(c.Conn, id, 0); err != nil {
						return err
					}
					continue
				}
				if err := tp.InjectFwd(p[2:]); err != nil {
					log.WithError(err).Warnln("FwdRejected")
					continue
				}
				if err := writeFrame(c.Conn, MakeFrame(AckType, tp.id, p[:2])); err != nil {
					return err
				}
				log.Infoln("FwdInjected")

			default:
				log.Infoln("Unexpected")
				if err := writeCloseFrame(c.Conn, id, 0); err != nil {
					return err
				}
			}

			continue
		}

		// if tp does not exist, frame should be 'REQUEST'.
		// otherwise, handle any unexpected frames accordingly.
		c.delTp(id) // rm tp in case closed tp is not fully removed.

		switch ft {
		case RequestType:
			c.wg.Add(1)
			go func(log *logrus.Entry) {
				defer c.wg.Done()

				ctx, cancel := context.WithTimeout(ctx, acceptTimeout)
				defer cancel()

				initPK, err := c.handleRequestFrame(ctx, accept, id, p)
				if err != nil {
					log.WithField("remoteClient", initPK).WithError(err).Infoln("Rejected:REQUEST")
					if isWriteError(err) || err == ErrClientClosed {
						closeConn(log)
					}
					return
				}
				log.WithField("remoteClient", initPK).Infoln("Accepted:REQUEST")
			}(log)

		case AcceptType:
			respPK, err := c.handleAcceptFrame(id, p)
			if err != nil {
				log.WithField("remoteClient", respPK).WithError(err).Infoln("Rejected:ACCEPT")
				if isWriteError(err) || err == ErrClientClosed {
					closeConn(log)
				}
				continue
			}
			log.WithField("remoteClient", respPK).Infoln("Accepted:ACCEPT")

		case CloseType:
			log.Infoln("CloseIgnored: Transport already ")

		default:
			log.Infoln("Unexpected")
			if err := writeCloseFrame(c.Conn, id, 0); err != nil {
				return err
			}
		}
	}
}

// DialTransport dials a transport to remote dms_client.
//func (c *ClientConn) DialTransport(ctx context.Context, clientPK cipher.PubKey) (*Transport, error) {
//	select {
//	case <-c.done:
//		return nil, ErrClientClosed
//	case <-ctx.Done():
//		return nil, ctx.Err()
//	default:
//		return c.addDialWaiter(ctx, clientPK)()
//	}
//}

// Close closes the connection to dms_server.
func (c *ClientConn) Close() error {
	c.log.WithField("remoteServer", c.remoteSrv).Infoln("ClosingConnection")
	c.once.Do(func() { close(c.done) })
	c.mx.Lock()
	for _, tp := range c.tps {
		if tp != nil {
			_ = tp.Close()
		}
	}
	err := c.Conn.Close()
	c.mx.Unlock()
	c.wg.Wait()
	return err
}

// Client implements transport.Factory
type Client struct {
	log *logging.Logger

	pk cipher.PubKey
	sk cipher.SecKey
	dc client.APIClient

	conns map[cipher.PubKey]*ClientConn // conns with messaging servers. Key: pk of server
	mx    sync.RWMutex

	accept chan *Transport
	done   chan struct{}
	once   sync.Once
}

// NewClient creates a new Client.
func NewClient(pk cipher.PubKey, sk cipher.SecKey, dc client.APIClient) *Client {
	return &Client{
		log:    logging.MustGetLogger("dmsg_client"),
		pk:     pk,
		sk:     sk,
		dc:     dc,
		conns:  make(map[cipher.PubKey]*ClientConn),
		accept: make(chan *Transport),
		done:   make(chan struct{}),
	}
}

// SetLogger sets the dms_client's logger.
func (c *Client) SetLogger(log *logging.Logger) {
	c.log = log
}

func (c *Client) updateDiscEntry(ctx context.Context) error {
	var srvPKs []cipher.PubKey
	for pk := range c.conns {
		srvPKs = append(srvPKs, pk)
	}
	entry, err := c.dc.Entry(ctx, c.pk)
	if err != nil {
		entry = client.NewClientEntry(c.pk, 0, srvPKs)
		if err := entry.Sign(c.sk); err != nil {
			return err
		}
		return c.dc.SetEntry(ctx, entry)
	}
	entry.Client.DelegatedServers = srvPKs
	c.log.Infoln("updatingEntry:", entry)
	return c.dc.UpdateEntry(ctx, c.sk, entry)
}

func (c *Client) setConn(ctx context.Context, l *ClientConn) {
	c.mx.Lock()
	c.conns[l.remoteSrv] = l
	if err := c.updateDiscEntry(ctx); err != nil {
		c.log.WithError(err).Warn("updateEntry: failed")
	}
	c.mx.Unlock()
}

func (c *Client) delConn(ctx context.Context, pk cipher.PubKey) {
	c.mx.Lock()
	delete(c.conns, pk)
	if err := c.updateDiscEntry(ctx); err != nil {
		c.log.WithError(err).Warn("updateEntry: failed")
	}
	c.mx.Unlock()
}

func (c *Client) getConn(pk cipher.PubKey) (*ClientConn, bool) {
	c.mx.RLock()
	l, ok := c.conns[pk]
	c.mx.RUnlock()
	return l, ok
}

func (c *Client) connCount() int {
	c.mx.RLock()
	n := len(c.conns)
	c.mx.RUnlock()
	return n
}

// InitiateServerConnections initiates connections with dms_servers.
func (c *Client) InitiateServerConnections(ctx context.Context, min int) error {
	if min == 0 {
		return nil
	}
	entries, err := c.findServerEntries(ctx)
	if err != nil {
		return err
	}
	c.log.Info("found dms_server entries:", entries)
	if err := c.findOrConnectToServers(ctx, entries, min); err != nil {
		return err
	}
	return nil
}

func (c *Client) findServerEntries(ctx context.Context) ([]*client.Entry, error) {
	for {
		entries, err := c.dc.AvailableServers(ctx)
		if err != nil || len(entries) == 0 {
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("dms_servers are not available: %s", err)
			default:
				retry := time.Second
				c.log.WithError(err).Warnf("no dms_servers found: trying again in %d second...", retry)
				time.Sleep(retry)
				continue
			}
		}
		return entries, nil
	}
}

func (c *Client) findOrConnectToServers(ctx context.Context, entries []*client.Entry, min int) error {
	for _, entry := range entries {
		_, err := c.findOrConnectToServer(ctx, entry.Static)
		if err != nil {
			c.log.Warnf("findOrConnectToServers: failed to find/connect to server %s: %s", entry.Static, err)
			continue
		}
		c.log.Infof("findOrConnectToServers: found/connected to server %s", entry.Static)
		if c.connCount() >= min {
			return nil
		}
	}
	return fmt.Errorf("findOrConnectToServers: all servers failed")
}

func (c *Client) findOrConnectToServer(ctx context.Context, srvPK cipher.PubKey) (*ClientConn, error) {
	if conn, ok := c.getConn(srvPK); ok {
		return conn, nil
	}

	entry, err := c.dc.Entry(ctx, srvPK)
	if err != nil {
		return nil, err
	}
	if entry.Server == nil {
		return nil, errors.New("entry is of client instead of server")
	}

	tcpConn, err := net.Dial("tcp", entry.Server.Address)
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
	nc, err := noise.WrapConn(tcpConn, ns, hsTimeout)
	if err != nil {
		return nil, err
	}

	conn := NewClientConn(c.log, nc, c.pk, srvPK)
	c.setConn(ctx, conn)
	go func() {
		if err := conn.Serve(ctx, c.accept); err != nil {
			conn.log.WithError(err).WithField("remoteServer", srvPK).Warn("connected with server closed")
			c.delConn(ctx, srvPK)

			// reconnect logic.
			t := time.NewTimer(time.Second * 2)
			select {
			case <-c.done:
			case <-ctx.Done():
			case <-t.C:
				conn.log.WithField("remoteServer", srvPK).Warn("Reconnecting")
				_, _ = c.findOrConnectToServer(ctx, srvPK) //nolint:errcheck
			}
			return
		}
	}()
	return conn, nil
}

// Accept accepts remotely-initiated tps.
func (c *Client) Accept(ctx context.Context) (transport.Transport, error) {
	select {
	case tp, ok := <-c.accept:
		if !ok {
			return nil, ErrClientClosed
		}
		return tp, nil
	case <-c.done:
		return nil, ErrClientClosed
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Dial dials a transport to remote dms_client.
func (c *Client) Dial(ctx context.Context, remote cipher.PubKey) (transport.Transport, error) {
	entry, err := c.dc.Entry(ctx, remote)
	if err != nil {
		return nil, fmt.Errorf("get entry failure: %s", err)
	}
	if entry.Client == nil {
		return nil, errors.New("entry is of server instead of client")
	}
	if len(entry.Client.DelegatedServers) == 0 {
		return nil, ErrNoSrv
	}
	for _, srvPK := range entry.Client.DelegatedServers {
		conn, err := c.findOrConnectToServer(ctx, srvPK)
		if err != nil {
			c.log.WithError(err).Warn("failed to connect to server")
			continue
		}
		return conn.DialTransport(ctx, remote)
	}
	return nil, errors.New("failed to find dms_servers for given client pk")
}

// Local returns the local dms_client's public key.
func (c *Client) Local() cipher.PubKey {
	return c.pk
}

// Type returns the transport type.
func (c *Client) Type() string {
	return Type
}

// Close closes the dms_client and associated connections.
// TODO(evaninjin): proper error handling.
func (c *Client) Close() error {
	c.once.Do(func() {
		close(c.done)
		for {
			select {
			case <-c.accept:
			default:
				close(c.accept)
				return
			}
		}
	})

	c.mx.Lock()
	for _, conn := range c.conns {
		_ = conn.Close()
	}
	c.conns = make(map[cipher.PubKey]*ClientConn)
	c.mx.Unlock()
	return nil
}
