package dmsg

import (
	"context"
	"encoding/json"
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

	net.Conn                // conn to dmsg server
	local     cipher.PubKey // local client's pk
	remoteSrv cipher.PubKey // dmsg server's public key

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
func NewClientConn(log *logging.Logger, conn net.Conn, local, remote cipher.PubKey, pm *PortManager) *ClientConn {
	cc := &ClientConn{
		log:        log,
		Conn:       conn,
		local:      local,
		remoteSrv:  remote,
		nextInitID: randID(true),
		tps:        make(map[uint16]*Transport),
		pm:         pm,
		done:       make(chan struct{}),
	}
	cc.wg.Add(1)
	return cc
}

// RemotePK returns the remote Server's PK that the ClientConn is connected to.
func (c *ClientConn) RemotePK() cipher.PubKey { return c.remoteSrv }

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

func (c *ClientConn) addTp(ctx context.Context, rPK cipher.PubKey, lPort, rPort uint16) (*Transport, error) {
	c.mx.Lock()
	defer c.mx.Unlock()

	id, err := c.getNextInitID(ctx)
	if err != nil {
		return nil, err
	}
	tp := NewTransport(c.Conn, c.log, Addr{c.local, lPort}, Addr{rPK, rPort}, id, c.delTp)
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
	fr, err := readFrame(c.Conn)
	if err != nil {
		return errors.New("failed to get OK from server")
	}

	ft, _, _ := fr.Disassemble()
	if ft != OkType {
		return fmt.Errorf("wrong frame from server: %v", ft)
	}

	return nil
}

func (c *ClientConn) handleRequestFrame(id uint16, p []byte) (cipher.PubKey, error) {
	// remotely-initiated tps should:
	// - have a payload structured as HandshakePayload marshaled to JSON.
	// - resp_pk should be of local client.
	// - use an odd tp_id with the intermediary dmsg_server.
	payload, err := unmarshalHandshakePayload(p)
	if err != nil {
		// TODO(nkryuchkov): When implementing reasons, send that payload format is incorrect.
		if err := writeCloseFrame(c.Conn, id, PlaceholderReason); err != nil {
			return cipher.PubKey{}, err
		}
		return cipher.PubKey{}, ErrRequestCheckFailed
	}

	if payload.RespPK != c.local || isInitiatorID(id) {
		// TODO(nkryuchkov): When implementing reasons, send that payload is malformed.
		if err := writeCloseFrame(c.Conn, id, PlaceholderReason); err != nil {
			return payload.InitPK, err
		}
		return payload.InitPK, ErrRequestCheckFailed
	}

	lis, ok := c.pm.Listener(payload.Port)
	if !ok {
		// TODO(nkryuchkov): When implementing reasons, send that port is not listening
		if err := writeCloseFrame(c.Conn, id, PlaceholderReason); err != nil {
			return payload.InitPK, err
		}
		return payload.InitPK, ErrPortNotListening
	}

	tp := NewTransport(c.Conn, c.log, Addr{c.local, payload.Port}, Addr{payload.InitPK, 0}, id, c.delTp) // TODO: Have proper remote port.

	select {
	case <-c.done:
		if err := tp.Close(); err != nil {
			log.WithError(err).Warn("Failed to close transport")
		}
		return payload.InitPK, ErrClientClosed

	default:
		err := lis.IntroduceTransport(tp)
		if err == nil || err == ErrClientAcceptMaxed {
			c.setTp(tp)
		}
		return payload.InitPK, err
	}
}

// Serve handles incoming frames.
// Remote-initiated tps that are successfully created are pushing into 'accept' and exposed via 'Client.Accept()'.
func (c *ClientConn) Serve(ctx context.Context) (err error) {
	log := c.log.WithField("remoteServer", c.remoteSrv)
	log.WithField("connCount", incrementServeCount()).Infoln("ServingConn")
	defer func() {
		c.close()
		log.WithError(err).WithField("connCount", decrementServeCount()).Infoln("ConnectionClosed")
		c.wg.Done()
	}()

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
			if err := tp.HandleFrame(f); err != nil {
				log.WithError(err).Warnf("Rejected [%s]: Transport closed.", ft)
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
				initPK, err := c.handleRequestFrame(id, p)
				if err != nil {
					log.WithField("remoteClient", initPK).WithError(err).Infoln("Rejected [REQUEST]")
					if isWriteError(err) || err == ErrClientClosed {
						err := c.Close()
						log.WithError(err).Warn("ClosingConnection")
					}
					return
				}
				log.WithField("remoteClient", initPK).Infoln("Accepted [REQUEST]")
			}(log)

		default:
			log.Debugf("Ignored [%s]: No transport of given ID.", ft)
			if ft != CloseType {
				if err := writeCloseFrame(c.Conn, id, PlaceholderReason); err != nil {
					return err
				}
			}
		}
	}
}

// DialTransport dials a transport to remote dms_client.
func (c *ClientConn) DialTransport(ctx context.Context, clientPK cipher.PubKey, port uint16) (*Transport, error) {
	tp, err := c.addTp(ctx, clientPK, 0, port) // TODO: Have proper local port.
	if err != nil {
		return nil, err
	}
	if err := tp.WriteRequest(port); err != nil {
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
		c.log.WithField("remoteServer", c.remoteSrv).Infoln("ClosingConnection")
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

func marshalHandshakePayload(p HandshakePayload) ([]byte, error) {
	return json.Marshal(p)
}

func unmarshalHandshakePayload(b []byte) (HandshakePayload, error) {
	var p HandshakePayload
	err := json.Unmarshal(b, &p)
	return p, err
}
