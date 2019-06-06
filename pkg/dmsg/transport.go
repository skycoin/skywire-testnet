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
		log.Fatalln("failed to set ack_waiter seq:", err)
	}
	return tp
}

func (tp *Transport) close() (closed bool) {
	tp.doneOnce.Do(func() {
		closed = true
		close(tp.doneCh)

		// Kill all goroutines pushing to `tp.readCh` before closing it.
		// No more goroutines pushing to `tp.readCh` should be created once `tp.doneCh` is closed.
		for {
			select {
			case <-tp.readCh:
			default:
				close(tp.readCh)
				return
			}
		}
	})
	return closed
}

func (tp *Transport) awaitResponse(ctx context.Context) error {
	select {
	case <-tp.doneCh:
		return ErrRequestRejected
	case <-ctx.Done():
		return ctx.Err()
	case f, ok := <-tp.readCh:
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
func (tp *Transport) Handshake(ctx context.Context) error {
	// if channel ID is even, client is initiator.
	if isInitiatorID(tp.id) {
		pks := combinePKs(tp.local, tp.remote)
		f := MakeFrame(RequestType, tp.id, pks)
		if err := writeFrame(tp.Conn, f); err != nil {
			tp.close()
			return err
		}
		if err := tp.awaitResponse(ctx); err != nil {
			tp.close()
			return err
		}
	} else {
		f := MakeFrame(AcceptType, tp.id, combinePKs(tp.remote, tp.local))
		if err := writeFrame(tp.Conn, f); err != nil {
			tp.log.WithError(err).Error("HandshakeFailed")
			tp.close()
			return err
		}
		tp.log.WithField("sent", f).Infoln("HandshakeCompleted")
	}
	return nil
}

// IsDone returns whether dms_tp is closed.
func (tp *Transport) IsDone() bool {
	select {
	case <-tp.doneCh:
		return true
	default:
		return false
	}
}

// InjectRead blocks until frame is read.
// Returns false when read fails (e.g. when tp is closed).
func (tp *Transport) InjectRead(f Frame) bool {
	ok := tp.injectRead(f)
	if !ok {
		tp.close()
	}
	return ok
}

func (tp *Transport) injectRead(f Frame) bool {
	push := func(f Frame) bool {
		select {
		case <-tp.doneCh:
			return false
		case tp.readCh <- f:
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
		tp.ackWaiter.Done(ioutil.DecodeUint16Seq(p))
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
			if err := writeFrame(tp.Conn, MakeFrame(AckType, tp.id, p[:2])); err != nil {
				tp.close()
			}
		}()
		return true

	default:
		return push(f)
	}
}

// Read implements io.Reader
func (tp *Transport) Read(p []byte) (n int, err error) {
	tp.readMx.Lock()
	defer tp.readMx.Unlock()

	if tp.readBuf.Len() != 0 {
		return tp.readBuf.Read(p)
	}

	select {
	case <-tp.doneCh:
		return 0, io.ErrClosedPipe
	case f, ok := <-tp.readCh:
		if !ok {
			return 0, io.ErrClosedPipe
		}
		if f.Type() == FwdType {
			return ioutil.BufRead(&tp.readBuf, f.Pay()[2:], p)
		}
		return 0, errors.New("unexpected frame")
	}
}

// Write implements io.Writer
func (tp *Transport) Write(p []byte) (int, error) {
	select {
	case <-tp.doneCh:
		return 0, io.ErrClosedPipe
	default:
		ctx, cancel := context.WithTimeout(context.Background(), readTimeout)
		go func() {
			select {
			case <-ctx.Done():
			case <-tp.doneCh:
				cancel()
			}
		}()
		err := tp.ackWaiter.Wait(ctx, func(seq ioutil.Uint16Seq) error {
			if err := writeFwdFrame(tp.Conn, tp.id, seq, p); err != nil {
				tp.close()
				return err
			}
			return nil
		})
		if err != nil {
			cancel()
			return 0, err
		}
		return len(p), nil
	}
}

// Close closes the dms_tp.
func (tp *Transport) Close() error {
	if tp.close() {
		_ = writeFrame(tp.Conn, MakeFrame(CloseType, tp.id, []byte{0})) //nolint:errcheck
		return nil
	}
	return io.ErrClosedPipe
}

// Edges returns the local/remote edges of the transport (dms_client to dms_client).
func (tp *Transport) Edges() [2]cipher.PubKey {
	return transport.SortPubKeys(tp.local, tp.remote)
}

// Type returns the transport type.
func (tp *Transport) Type() string {
	return Type
}
