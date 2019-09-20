package dmsg

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/dmsg/cipher"
)

// NextConn provides information on the next connection.
type NextConn struct {
	conn *ServerConn
	id   uint16
}

func (r *NextConn) writeFrame(ft FrameType, p []byte) error {
	if err := writeFrame(r.conn.Conn, MakeFrame(ft, r.id, p)); err != nil {
		go func() {
			if err := r.conn.Close(); err != nil {
				log.WithError(err).Warn("Failed to close connection")
			}
		}()
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
	delete(c.nextConns, id)
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
	log := c.log.WithField("srcClient", c.remoteClient)

	// Only manually close the underlying net.Conn when the done signal is context-initiated.
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-done:
		case <-ctx.Done():
			if err := c.Conn.Close(); err != nil {
				log.WithError(err).Warn("failed to close underlying connection")
			}
		}
	}()

	defer func() {
		// Send CLOSE frames to all transports which are established with this dmsg.Client
		// This ensures that all parties are informed about the transport closing.
		c.mx.Lock()
		for _, conn := range c.nextConns {
			why := byte(0)
			if err := conn.writeFrame(CloseType, []byte{why}); err != nil {
				log.WithError(err).Warnf("failed to write frame: %s", err)
			}
		}
		c.mx.Unlock()

		log.WithError(err).WithField("connCount", decrementServeCount()).Infoln("ClosingConn")
		if err := c.Conn.Close(); err != nil {
			log.WithError(err).Warn("Failed to close connection")
		}
	}()

	log.WithField("connCount", incrementServeCount()).Infoln("ServingConn")

	err = c.writeOK()
	if err != nil {
		return fmt.Errorf("sending OK failed: %s", err)
	}

	for {
		f, df, err := readFrame(c.Conn)
		if err != nil {
			return fmt.Errorf("read failed: %s", err)
		}
		log := log.WithField("received", f)

		switch df.Type {
		case RequestType:
			ctx, cancel := context.WithTimeout(ctx, TransportHandshakeTimeout)
			_, why, ok := c.handleRequest(ctx, getConn, df.TpID, df.Pay)
			cancel()
			if !ok {
				log.Debugln("FrameRejected: Erroneous request or unresponsive dstClient.")
				if err := c.delChan(df.TpID, why); err != nil {
					return err
				}
			}
			log.Debugln("FrameForwarded")

		case AcceptType, FwdType, AckType, CloseType:
			next, why, ok := c.forwardFrame(df.Type, df.TpID, df.Pay)
			if !ok {
				log.Debugln("FrameRejected: Failed to forward to dstClient.")
				// Delete channel (and associations) on failure.
				if err := c.delChan(df.TpID, why); err != nil {
					return err
				}
				continue
			}
			log.Debugln("FrameForwarded")

			// On success, if Close frame, delete the associations.
			if df.Type == CloseType {
				c.delNext(df.TpID)
				next.conn.delNext(next.id)
			}

		default:
			log.Debugln("FrameRejected: Unknown frame type.")
			return errors.New("unknown frame of type received")
		}
	}
}

func (c *ServerConn) delChan(id uint16, why byte) error {
	c.delNext(id)
	if err := writeCloseFrame(c.Conn, id, why); err != nil {
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

// nolint:unparam
func (c *ServerConn) forwardFrame(ft FrameType, id uint16, p []byte) (*NextConn, byte, bool) {
	next, ok := c.getNext(id)
	if !ok {
		return next, 0, false
	}
	if err := next.writeFrame(ft, p); err != nil {
		return next, 0, false
	}
	return next, 0, true
}

// nolint:unparam
func (c *ServerConn) handleRequest(ctx context.Context, getLink getConnFunc, id uint16, p []byte) (*NextConn, byte, bool) {
	payload, err := unmarshalHandshakePayload(p)
	if err != nil || payload.InitAddr.PK != c.PK() {
		return nil, 0, false
	}
	respL, ok := getLink(payload.RespAddr.PK)
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
