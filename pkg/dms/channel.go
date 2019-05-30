package dms

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"sync"

	"github.com/skycoin/skywire/pkg/cipher"
	"github.com/skycoin/skywire/pkg/transport"
)

// Channel is a channel from a client's perspective.
type Channel struct {
	net.Conn     // link with server.
	id           uint16
	local        cipher.PubKey
	remoteClient cipher.PubKey // remote PK
	readBuf      bytes.Buffer
	readCh       chan Frame    // TODO(evanlinjin): find proper way of closing readCh.
	doneCh       chan struct{} // stop writing
	doneOnce     sync.Once
}

func NewChannel(conn net.Conn, local, remote cipher.PubKey, id uint16) *Channel {
	return &Channel{
		Conn:         conn,
		id:           id,
		local:        local,
		remoteClient: remote,
		readCh:       make(chan Frame, readBufLen),
		doneCh:       make(chan struct{}),
	}
}

func (c *Channel) awaitResponse(ctx context.Context) error {
	select {
	case f := <-c.readCh:
		switch f.Type() {
		case AcceptType:
			return nil
		case CloseType:
			return errors.New("rejected")
		default:
			return errors.New("invalid remote response")
		}
	case <-c.doneCh:
		return errors.New("closed")
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Channel) close() bool {
	closed := false
	c.doneOnce.Do(func() {
		close(c.doneCh)
		closed = true
	})
	return closed
}

func (c *Channel) Handshake(ctx context.Context) error {
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

func (c *Channel) IsDone() bool {
	select {
	case <-c.doneCh:
		return true
	default:
		return false
	}
}

func (c *Channel) AwaitRead(f Frame) bool {
	select {
	case c.readCh <- f:
		return true
	case <-c.doneCh:
		return false
	}
}

func (c *Channel) Edges() [2]cipher.PubKey {
	return transport.SortPubKeys(c.local, c.remoteClient)
}

func (c *Channel) Type() string {
	return TpType
}

func (c *Channel) Read(p []byte) (n int, err error) {
	if c.readBuf.Len() != 0 {
		return c.readBuf.Read(p)
	}

	select {
	case <-c.doneCh:
		return 0, io.ErrClosedPipe
	case f := <-c.readCh:
		switch f.Type() {
		case FwdType:
			return c.bufRead(f.Pay(), p)
		case CloseType:
			c.close()
			return 0, io.ErrClosedPipe
		default:
			return 0, errors.New("unexpected frame")
		}
	}
}

func (c *Channel) bufRead(data, p []byte) (int, error) {
	if len(data) > len(p) {
		if _, err := c.readBuf.Write(data[len(p):]); err != nil {
			return 0, io.ErrShortBuffer
		}
		copy(p, data[:len(p)])
	}
	return copy(p, data), nil
}

func (c *Channel) Write(p []byte) (int, error) {
	select {
	case <-c.doneCh:
		return 0, io.ErrClosedPipe
	default:
		f := MakeFrame(FwdType, c.id, p)
		if err := writeFrame(c.Conn, f); err != nil {
			c.close()
			return 0, err
		}
		return f.PayLen(), nil
	}
}

func (c *Channel) Close() error {
	if c.close() {
		_ = writeFrame(c.Conn, MakeFrame(CloseType, c.id, []byte{0}))
	}
	return nil
}
