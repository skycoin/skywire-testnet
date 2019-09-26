package dmsg

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/dmsg/cipher"
)

// ClientConn represents a connection between a dmsg.Client and dmsg.Server from a client's perspective.
type ClientConn struct {
	log *logging.Logger

	net.Conn               // conn to dmsg server
	lPK      cipher.PubKey // local client's pk
	srvPK    cipher.PubKey // dmsg server's public key

	// nextInitID keeps track of unused tp_ids to assign a future locally-initiated tp.
	// locally-initiated tps use an even tp_id between local and intermediary dms_server.
	nextInitID uint16

	// Transports: map of transports to remote dms_clients (key: tp_id, val: transport).
	tps map[uint16]*Transport
	mx  sync.RWMutex // to protect tps

	pm *PortManager

	done chan struct{}
	once sync.Once
	wg   sync.WaitGroup
}

// NewClientConn creates a new ClientConn.
func NewClientConn(log *logging.Logger, pm *PortManager, conn net.Conn, lPK, rPK cipher.PubKey) *ClientConn {
	cc := &ClientConn{
		log:        log,
		Conn:       conn,
		lPK:        lPK,
		srvPK:      rPK,
		nextInitID: randID(true),
		tps:        make(map[uint16]*Transport),
		pm:         pm,
		done:       make(chan struct{}),
	}
	cc.wg.Add(1)
	return cc
}

// RemotePK returns the remote Server's PK that the ClientConn is connected to.
func (c *ClientConn) RemotePK() cipher.PubKey { return c.srvPK }

func (c *ClientConn) getNextInitID(ctx context.Context) (uint16, error) {
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

func (c *ClientConn) addTp(ctx context.Context, rPK cipher.PubKey, lPort, rPort uint16, closeCB func()) (*Transport, error) {
	c.mx.Lock()
	defer c.mx.Unlock()

	id, err := c.getNextInitID(ctx)
	if err != nil {
		return nil, err
	}
	tp := NewTransport(c.Conn, c.log, Addr{c.lPK, lPort}, Addr{rPK, rPort}, id, func() {
		c.delTp(id)
		closeCB()
	})
	c.tps[id] = tp
	return tp, nil
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

func (c *ClientConn) setNextInitID(nextInitID uint16) {
	c.mx.Lock()
	c.nextInitID = nextInitID
	c.mx.Unlock()
}

func (c *ClientConn) readOK() error {
	_, df, err := readFrame(c.Conn)
	if err != nil {
		return errors.New("failed to get OK from server")
	}
	if df.Type != OkType {
		return fmt.Errorf("wrong frame from server: %v", df.Type)
	}
	return nil
}

// This handles 'REQUEST' frames which represent remotely-initiated tps. 'REQUEST' frames should:
// - have a HandshakePayload marshaled to JSON as payload.
// - have a resp_pk be of local client.
// - have an odd tp_id.
func (c *ClientConn) handleRequestFrame(log *logrus.Entry, id uint16, p []byte) (cipher.PubKey, error) {

	// The public key of the initiating client (or the client that sent the 'REQUEST' frame).
	var initPK cipher.PubKey

	// Attempts to close tp due to given error.
	// When we fail to close tp (a.k.a fail to send 'CLOSE' frame) or if the local client is closed,
	// the connection to server should be closed.
	// TODO(evanlinjin): derive close reason from error.
	closeTp := func(origErr error) (cipher.PubKey, error) {
		if err := writeCloseFrame(c.Conn, id, PlaceholderReason); err != nil {
			log.WithError(err).Warn("handleRequestFrame: failed to close transport: ending conn to server.")
			log.WithError(c.Close()).Warn("handleRequestFrame: closing connection to server.")
			return initPK, origErr
		}
		switch origErr {
		case ErrClientClosed:
			log.WithError(c.Close()).Warn("handleRequestFrame: closing connection to server.")
		}
		return initPK, origErr
	}

	pay, err := unmarshalHandshakePayload(p)
	if err != nil {
		return closeTp(ErrRequestCheckFailed) // TODO(nkryuchkov): reason = payload format is incorrect.
	}
	initPK = pay.InitAddr.PK

	if pay.RespAddr.PK != c.lPK || isInitiatorID(id) {
		return closeTp(ErrRequestCheckFailed) // TODO(nkryuchkov): reason = payload is malformed.
	}
	lis, ok := c.pm.Listener(pay.RespAddr.Port)
	if !ok {
		return closeTp(ErrPortNotListening) // TODO(nkryuchkov): reason = port is not listening.
	}
	if c.isClosed() {
		return closeTp(ErrClientClosed) // TODO(nkryuchkov): reason = client is closed.
	}

	tp := NewTransport(c.Conn, c.log, pay.RespAddr, pay.InitAddr, id, func() { c.delTp(id) })
	if err := lis.IntroduceTransport(tp); err != nil {
		return initPK, err
	}
	c.setTp(tp)
	return initPK, nil
}

// Serve handles incoming frames.
// Remote-initiated tps that are successfully created are pushing into 'accept' and exposed via 'Client.Accept()'.
func (c *ClientConn) Serve(ctx context.Context) (err error) {
	log := c.log.WithField("remoteServer", c.srvPK)
	log.WithField("connCount", incrementServeCount()).Infoln("ServingConn")
	defer func() {
		c.close()
		log.WithError(err).WithField("connCount", decrementServeCount()).Infoln("ConnectionClosed")
		c.wg.Done()
	}()

	for {
		f, df, err := readFrame(c.Conn)
		if err != nil {
			return fmt.Errorf("read failed: %s", err)
		}
		log = log.WithField("received", f)

		// If tp of tp_id exists, attempt to forward frame to tp.
		// Delete tp on any failure.
		if tp, ok := c.getTp(df.TpID); ok {
			if err := tp.HandleFrame(f); err != nil {
				log.WithError(err).Warnf("Rejected [%s]: Transport closed.", df.Type)
			}
			continue
		}
		c.delTp(df.TpID) // rm tp in case closed tp is not fully removed.

		// if tp does not exist, frame should be 'REQUEST'.
		// otherwise, handle any unexpected frames accordingly.
		switch df.Type {
		case RequestType:
			c.wg.Add(1)
			go func(log *logrus.Entry) {
				defer c.wg.Done()
				if initPK, err := c.handleRequestFrame(log, df.TpID, df.Pay); err != nil {
					log.WithField("remoteClient", initPK).WithError(err).Warn("Rejected [REQUEST]")
				} else {
					log.WithField("remoteClient", initPK).Info("Accepted [REQUEST]")
				}
			}(log)

		default:
			log.Debugf("Ignored [%s]: No transport of given ID.", df.Type)
			if df.Type != CloseType {
				if err := writeCloseFrame(c.Conn, df.TpID, PlaceholderReason); err != nil {
					return err
				}
			}
		}
	}
}

// DialTransport dials a transport to remote dms_client.
func (c *ClientConn) DialTransport(ctx context.Context, rPK cipher.PubKey, rPort uint16) (*Transport, error) {
	lPort, closeCB, err := c.pm.ReserveEphemeral(ctx)
	if err != nil {
		return nil, err
	}
	tp, err := c.addTp(ctx, rPK, lPort, rPort, closeCB) // TODO: Have proper local port.
	if err != nil {
		return nil, err
	}
	if err := tp.WriteRequest(); err != nil {
		return nil, err
	}
	if err := tp.ReadAccept(ctx); err != nil {
		return nil, err
	}
	go tp.Serve()
	return tp, nil
}

func (c *ClientConn) close() (closed bool) {
	if c == nil {
		return false
	}
	c.once.Do(func() {
		closed = true
		c.log.WithField("remoteServer", c.srvPK).Infoln("ClosingConnection")
		close(c.done)
		c.mx.Lock()
		for _, tp := range c.tps {
			tp := tp
			go func() {
				if err := tp.Close(); err != nil {
					log.WithError(err).Warn("Failed to close transport")
				}
			}()
		}
		if err := c.Conn.Close(); err != nil {
			log.WithError(err).Warn("Failed to close connection")
		}
		c.mx.Unlock()
	})
	return closed
}

// Close closes the connection to dms_server.
func (c *ClientConn) Close() error {
	if c.close() {
		c.wg.Wait()
	}
	return nil
}

func (c *ClientConn) isClosed() bool {
	select {
	case <-c.done:
		return true
	default:
		return false
	}
}
