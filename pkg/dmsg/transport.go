package dmsg

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"sync"

	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/internal/ioutil"
	"github.com/skycoin/skywire/pkg/cipher"
	"github.com/skycoin/skywire/pkg/transport"
)

// Errors related to REQUESTs.
var (
	ErrRequestRejected    = errors.New("request rejected")
	ErrRequestCheckFailed = errors.New("request check failed")
)

// Transport represents a connection from dmsg.Client to remote dmsg.Client (via dmsg.Server intermediary).
// It implements transport.Transport
type Transport struct {
	net.Conn // link with server.
	log      *logging.Logger

	id     uint16
	local  cipher.PubKey
	remote cipher.PubKey // remote PK

	ackWaiter ioutil.Uint16AckWaiter
	readBuf   bytes.Buffer
	readMx    sync.Mutex // This is for protecting 'readBuf'.
	readCh    chan Frame
	doneCh    chan struct{} // stop writing
	doneOnce  sync.Once
}

// NewTransport creates a new dms_tp.
func NewTransport(conn net.Conn, log *logging.Logger, local, remote cipher.PubKey, id uint16) *Transport {
	tp := &Transport{
		Conn:   conn,
		log:    log,
		id:     id,
		local:  local,
		remote: remote,
		readCh: make(chan Frame, readChSize),
		doneCh: make(chan struct{}),
	}
	if err := tp.ackWaiter.RandSeq(); err != nil {
		log.Fatalln("failed to set ack_water seq:", err)
	}
	return tp
}

func (c *Transport) close() (closed bool) {
	c.doneOnce.Do(func() {
		closed = true
		close(c.doneCh)

		// Kill all goroutines pushing to `c.readCh` before closing it.
		// No more goroutines pushing to `c.readCh` should be created once `c.doneCh` is closed.
		for {
			select {
			case <-c.readCh:
			default:
				close(c.readCh)
				return
			}
		}
	})
	return closed
}

func (c *Transport) awaitResponse(ctx context.Context) error {
	select {
	case <-c.doneCh:
		return ErrRequestRejected
	case <-ctx.Done():
		return ctx.Err()
	case f, ok := <-c.readCh:
		if !ok {
			return io.ErrClosedPipe
		}
		if f.Type() == AcceptType {
			return nil
		}
		return errors.New("invalid remote response")
	}
}

// Handshake performs a tp handshake (before tp is considered valid).
func (c *Transport) Handshake(ctx context.Context) error {
	// if channel ID is even, client is initiator.
	if isInitiatorID(c.id) {
		pks := combinePKs(c.local, c.remote)
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
		c.log.Infof("tp_hs responding...")
		f := MakeFrame(AcceptType, c.id, combinePKs(c.remote, c.local))
		if err := writeFrame(c.Conn, f); err != nil {
			c.log.WithError(err).Error("tp_hs responded with error.")
			c.close()
			return err
		}
		c.log.Infoln("tp_hs responded:", f)
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

// InjectRead blocks until frame is read.
// Returns false when read fails (e.g. when tp is closed).
func (c *Transport) InjectRead(f Frame) bool {
	ok := c.injectRead(f)
	if !ok {
		c.close()
	}
	return ok
}

func (c *Transport) injectRead(f Frame) bool {
	push := func(f Frame) bool {
		select {
		case <-c.doneCh:
			return false
		case c.readCh <- f:
			return true
		default:
			return false
		}
	}

	switch f.Type() {
	case CloseType:
		return false

	case AckType:
		p := f.Pay()
		if len(p) != 2 {
			return false
		}
		c.ackWaiter.Done(ioutil.DecodeUint16Seq(p))
		return true

	case FwdType:
		p := f.Pay()
		if len(p) < 2 {
			return false
		}
		if ok := push(f); !ok {
			return false
		}
		go func() {
			if err := writeFrame(c.Conn, MakeFrame(AckType, c.id, p[:2])); err != nil {
				c.close()
			}
		}()
		return true

	default:
		return push(f)
	}
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
	case f, ok := <-c.readCh:
		if !ok {
			return 0, io.ErrClosedPipe
		}
		if f.Type() == FwdType {
			return ioutil.BufRead(&c.readBuf, f.Pay()[2:], p)
		}
		return 0, errors.New("unexpected frame")
	}
}

// Write implements io.Writer
func (c *Transport) Write(p []byte) (int, error) {
	select {
	case <-c.doneCh:
		return 0, io.ErrClosedPipe
	default:
		ctx, cancel := context.WithTimeout(context.Background(), readTimeout)
		defer cancel()

		err := c.ackWaiter.Wait(ctx, c.doneCh, func(seq ioutil.Uint16Seq) error {
			if err := writeFwdFrame(c.Conn, c.id, seq, p); err != nil {
				c.close()
				return err
			}
			return nil
		})
		if err != nil {
			return 0, err
		}
		return len(p), nil
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

// Edges returns the local/remote edges of the transport (dms_client to dms_client).
func (c *Transport) Edges() [2]cipher.PubKey {
	return transport.SortPubKeys(c.local, c.remote)
}

// Type returns the transport type.
func (c *Transport) Type() string {
	return Type
}
