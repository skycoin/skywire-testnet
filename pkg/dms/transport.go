package dms

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"sync"

	"github.com/skycoin/skywire/internal/ioutil"

	"github.com/skycoin/skywire/pkg/cipher"
	"github.com/skycoin/skywire/pkg/transport"
)

// Errors related to REQUESTs.
var (
	ErrRequestRejected    = errors.New("request rejected")
	ErrRequestCheckFailed = errors.New("request check failed")
)

// Transport represents a connection from dms.Client to remote dms.Client (via dms.Server intermediary).
// It implements transport.Transport
type Transport struct {
	net.Conn     // link with server.
	id           uint16
	local        cipher.PubKey
	remoteClient cipher.PubKey // remote PK
	readBuf      bytes.Buffer
	readMx       sync.Mutex
	readCh       chan Frame    // TODO(evanlinjin): find proper way of closing readCh.
	doneCh       chan struct{} // stop writing
	doneOnce     sync.Once
}

// NewTransport creates a new dms_tp.
func NewTransport(conn net.Conn, local, remote cipher.PubKey, id uint16) *Transport {
	return &Transport{
		Conn:         conn,
		id:           id,
		local:        local,
		remoteClient: remote,
		readCh:       make(chan Frame, readBufLen),
		doneCh:       make(chan struct{}),
	}
}

func (c *Transport) awaitResponse(ctx context.Context) error {
	select {
	case f := <-c.readCh:
		switch f.Type() {
		case AcceptType:
			return nil
		case CloseType:
			return ErrRequestRejected
		default:
			return errors.New("invalid remote response")
		}
	case <-c.doneCh:
		return errors.New("closed")
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Transport) close() (closed bool) {
	c.doneOnce.Do(func() {
		close(c.doneCh)
		closed = true
	})
	return closed
}

// Handshake performs a tp handshake (before tp is considered valid).
func (c *Transport) Handshake(ctx context.Context) error {
	// if channel ID is even, client is initiator.
	if init := isEven(c.id); init {

		pks := combinePKs(c.local, c.remoteClient)
		f := MakeFrame(RequestType, c.id, pks)
		if err := writeFrame(c.Conn, f); err != nil {
			c.close()
			return err
		}
		if err := c.awaitResponse(ctx); err != nil {
			c.close()
			return err
		}
	} else {
		f := MakeFrame(AcceptType, c.id, combinePKs(c.remoteClient, c.local))
		if err := writeFrame(c.Conn, f); err != nil {
			c.close()
			return err
		}
	}
	return nil
}

// IsDone returns whether dms_tp is closed.
func (c *Transport) IsDone() bool {
	select {
	case <-c.doneCh:
		return true
	default:
		return false
	}
}

// AwaitRead blocks until frame is read.
// Returns false when read fails (when tp is closed).
func (c *Transport) AwaitRead(f Frame) bool {
	select {
	case c.readCh <- f:
		return true
	case <-c.doneCh:
		return false
	}
}

// Edges returns the local/remote edges of the transport (dms_client to dms_client).
func (c *Transport) Edges() [2]cipher.PubKey {
	return transport.SortPubKeys(c.local, c.remoteClient)
}

// Type returns the transport type.
func (c *Transport) Type() string {
	return Type
}

// Read implements io.Reader
func (c *Transport) Read(p []byte) (n int, err error) {
	c.readMx.Lock()
	defer c.readMx.Unlock()

	if c.readBuf.Len() != 0 {
		return c.readBuf.Read(p)
	}

	select {
	case <-c.doneCh:
		return 0, io.ErrClosedPipe
	case f := <-c.readCh:
		switch f.Type() {
		case SendType:
			return ioutil.BufRead(&c.readBuf, f.Pay(), p)
		case CloseType:
			c.close()
			return 0, io.ErrClosedPipe
		default:
			return 0, errors.New("unexpected frame")
		}
	}
}

// Write implements io.Writer
func (c *Transport) Write(p []byte) (int, error) {
	select {
	case <-c.doneCh:
		return 0, io.ErrClosedPipe
	default:
		f := MakeFrame(SendType, c.id, p)
		if err := writeFrame(c.Conn, f); err != nil {
			c.close()
			return 0, err
		}
		return f.PayLen(), nil
	}
}

// Close closes the dms_tp.
func (c *Transport) Close() error {
	if c.close() {
		_ = writeFrame(c.Conn, MakeFrame(CloseType, c.id, []byte{0})) //nolint:errcheck
		return nil
	}
	return io.ErrClosedPipe
}
