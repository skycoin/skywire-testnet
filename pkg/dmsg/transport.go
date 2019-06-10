package dmsg

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/internal/ioutil"
	"github.com/skycoin/skywire/pkg/cipher"
	"github.com/skycoin/skywire/pkg/transport"
)

// Errors related to REQUEST frames.
var (
	ErrRequestRejected    = errors.New("failed to create transport: request rejected")
	ErrRequestCheckFailed = errors.New("failed to create transport: request check failed")
	ErrAcceptCheckFailed  = errors.New("failed to create transport: accept check failed")
)

// Transport represents a connection from dmsg.Client to remote dmsg.Client (via dmsg.Server intermediary).
// It implements transport.Transport
type Transport struct {
	net.Conn // link with server.
	log      *logging.Logger

	id     uint16
	local  cipher.PubKey
	remote cipher.PubKey // remote PK

	inCh chan Frame
	inMx sync.RWMutex

	ackWaiter ioutil.Uint16AckWaiter
	ackBuf    []byte
	buf       net.Buffers
	bufCh     chan struct{}
	bufSize   int
	bufMx     sync.Mutex // protects 'buf' and 'bufCh'

	once     sync.Once
	done     chan struct{}
	doneFunc func(id uint16)
}

// NewTransport creates a new dms_tp.
func NewTransport(conn net.Conn, log *logging.Logger, local, remote cipher.PubKey, id uint16, doneFunc func(id uint16)) *Transport {
	tp := &Transport{
		Conn:     conn,
		log:      log,
		id:       id,
		local:    local,
		remote:   remote,
		inCh:     make(chan Frame),
		ackBuf:   make([]byte, 0, tpAckCap),
		buf:      make(net.Buffers, 0, tpBufFrameCap),
		bufCh:    make(chan struct{}, 1),
		done:     make(chan struct{}),
		doneFunc: doneFunc,
	}
	if err := tp.ackWaiter.RandSeq(); err != nil {
		log.Fatalln("failed to set ack_waiter seq:", err)
	}
	return tp
}

func (tp *Transport) close() (closed bool) {
	tp.once.Do(func() {
		closed = true

		close(tp.done)
		tp.doneFunc(tp.id)

		tp.bufMx.Lock()
		close(tp.bufCh)
		tp.bufMx.Unlock()

		tp.inMx.Lock()
		close(tp.inCh)
		tp.inMx.Unlock()

	})

	tp.ackWaiter.StopAll()
	return closed
}

// Close closes the dmsg_tp.
func (tp *Transport) Close() error {
	if tp.close() {
		_ = writeFrame(tp.Conn, MakeFrame(CloseType, tp.id, []byte{0})) //nolint:errcheck
	}
	return nil
}

// IsClosed returns whether dms_tp is closed.
func (tp *Transport) IsClosed() bool {
	select {
	case <-tp.done:
		return true
	default:
		return false
	}
}

// Edges returns the local/remote edges of the transport (dms_client to dms_client).
func (tp *Transport) Edges() [2]cipher.PubKey {
	return transport.SortPubKeys(tp.local, tp.remote)
}

// Type returns the transport type.
func (tp *Transport) Type() string {
	return Type
}

// Inject injects a frame from 'ClientConn' to transport.
// Frame is then handled by 'tp.Serve'.
func (tp *Transport) Inject(f Frame) error {
	if tp.IsClosed() {
		return io.ErrClosedPipe
	}

	tp.inMx.RLock()
	defer tp.inMx.RUnlock()

	select {
	case <-tp.done:
		return io.ErrClosedPipe
	case tp.inCh <- f:
		return nil
	}
}

// WriteRequest writes a REQUEST frame to dmsg_server to be forwarded to associated client.
func (tp *Transport) WriteRequest() error {
	f := MakeFrame(RequestType, tp.id, combinePKs(tp.local, tp.remote))
	if err := writeFrame(tp.Conn, f); err != nil {
		tp.log.WithError(err).Error("HandshakeFailed")
		tp.close()
		return err
	}
	return nil
}

// WriteAccept writes an ACCEPT frame to dmsg_server to be forwarded to associated client.
func (tp *Transport) WriteAccept() error {
	f := MakeFrame(AcceptType, tp.id, combinePKs(tp.remote, tp.local))
	if err := writeFrame(tp.Conn, f); err != nil {
		tp.log.WithError(err).Error("HandshakeFailed")
		tp.close()
		return err
	}
	tp.log.WithField("sent", f).Infoln("HandshakeCompleted")
	return nil
}

// ReadAccept awaits for an ACCEPT frame to be read from the remote client.
// TODO(evanlinjin): Cleanup errors.
func (tp *Transport) ReadAccept(ctx context.Context) (err error) {
	defer func() {
		tp.log.WithError(err).WithField("success", err == nil).Infoln("HandshakeDone")
	}()

	select {
	case <-tp.done:
		tp.close()
		return io.ErrClosedPipe

	case <-ctx.Done():
		_ = tp.Close() //nolint:errcheck
		return ctx.Err()

	case f, ok := <-tp.inCh:
		if !ok {
			tp.close()
			return io.ErrClosedPipe
		}
		switch ft, id, p := f.Disassemble(); ft {
		case AcceptType:
			// locally-initiated tps should:
			// - have a payload structured as 'init_pk:resp_pk'.
			// - init_pk should be of local client.
			// - resp_pk should be of remote client.
			// - use an even number with the intermediary dmsg_server.
			initPK, respPK, ok := splitPKs(p)
			if !ok || initPK != tp.local || respPK != tp.remote || !isInitiatorID(id) {
				_ = tp.Close() //nolint:errcheck
				return ErrAcceptCheckFailed
			}
			return nil

		case CloseType:
			tp.close()
			return ErrRequestRejected

		default:
			_ = tp.Close() //nolint:errcheck
			return ErrAcceptCheckFailed
		}
	}
}

// Serve handles received frames.
func (tp *Transport) Serve() {
	defer func() {
		if tp.close() {
			_ = writeCloseFrame(tp.Conn, tp.id, 0) //nolint:errcheck
		}
	}()

	for {
		select {
		case <-tp.done:
			return

		case f, ok := <-tp.inCh:
			if !ok {
				return
			}
			log := tp.log.
				WithField("remoteClient", tp.remote).
				WithField("received", f)

			switch p := f.Pay(); f.Type() {
			case FwdType:
				if len(p) < 2 {
					log.Warnln("Rejected [FWD]: Invalid payload size.")
					return
				}
				ack := MakeFrame(AckType, tp.id, p[:2])

				tp.bufMx.Lock()
				if tp.bufSize += len(p[2:]); tp.bufSize > tpBufCap {
					tp.ackBuf = append(tp.ackBuf, ack...)
				} else {
					go func() {
						if err := writeFrame(tp.Conn, ack); err != nil {
							tp.close()
						}
					}()
				}
				tp.buf = append(tp.buf, p[2:])
				select {
				case <-tp.done:
				case tp.bufCh <- struct{}{}:
				default:
				}
				log.WithField("bufSize", fmt.Sprintf("%d/%d", tp.bufSize, tpBufCap)).Infoln("Injected [FWD]")
				tp.bufMx.Unlock()

			case AckType:
				if len(p) != 2 {
					log.Warnln("Rejected [ACK]: Invalid payload size.")
					return
				}
				tp.ackWaiter.Done(ioutil.DecodeUint16Seq(p[:2]))
				log.Infoln("Injected [ACK]")

			case CloseType:
				log.Infoln("Injected [CLOSE]: Closing transport...")
				return

			case RequestType:
				log.Warnln("Rejected [REQUEST]: ID already occupied, malicious server.")
				_ = tp.Conn.Close()
				return

			default:
				tp.log.Infof("Rejected [%s]: Unexpected frame, malicious server (ignored for now).", f.Type())
			}
		}
	}
}

// Read implements io.Reader
// TODO(evanlinjin): read deadline.
func (tp *Transport) Read(p []byte) (n int, err error) {
startRead:
	tp.bufMx.Lock()
	n, err = tp.buf.Read(p)
	go func() {
		if tp.bufSize -= n; tp.bufSize < tpBufCap {
			if err := writeFrame(tp.Conn, tp.ackBuf); err != nil {
				tp.close()
			}
			tp.ackBuf = make([]byte, 0, tpAckCap)
		}
		tp.bufMx.Unlock()
	}()

	if tp.IsClosed() {
		return n, err
	}
	if n > 0 {
		return n, nil
	}
	<-tp.bufCh
	goto startRead
}

// Write implements io.Writer
// TODO(evanlinjin): write deadline.
func (tp *Transport) Write(p []byte) (int, error) {
	if tp.IsClosed() {
		return 0, io.ErrClosedPipe
	}
	err := tp.ackWaiter.Wait(context.Background(), func(seq ioutil.Uint16Seq) error {
		if err := writeFwdFrame(tp.Conn, tp.id, seq, p); err != nil {
			tp.close()
			return err
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return len(p), nil
}
