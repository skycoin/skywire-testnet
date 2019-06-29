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
	net.Conn // underlying connection to dmsg.Server
	log      *logging.Logger

	id     uint16        // tp ID that identifies this dmsg.Transport
	local  cipher.PubKey // local PK
	remote cipher.PubKey // remote PK

	inCh chan Frame // handles incoming frames (from dmsg.Client)
	inMx sync.Mutex // protects 'inCh'

	ackWaiter ioutil.Uint16AckWaiter // awaits for associated ACK frames
	ackBuf    []byte                 // buffer for unsent ACK frames
	buf       net.Buffers            // buffer for non-read FWD frames
	bufCh     chan struct{}          // chan for indicating whether this is a new FWD frame
	bufSize   int                    // keeps track of the total size of 'buf'
	bufMx     sync.Mutex             // protects fields responsible for handling FWD and ACK frames

	serving     chan struct{}   // chan which closes when serving begins
	servingOnce sync.Once       // ensures 'serving' only closes once
	done        chan struct{}   // chan which closes when transport stops serving
	doneOnce    sync.Once       // ensures 'done' only closes once
	doneFunc    func(id uint16) // contains a method to remove the transport from dmsg.Client
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
		serving:  make(chan struct{}),
		done:     make(chan struct{}),
		doneFunc: doneFunc,
	}
	if err := tp.ackWaiter.RandSeq(); err != nil {
		log.Fatalln("failed to set ack_waiter seq:", err)
	}
	return tp
}

func (tp *Transport) serve() (started bool) {
	tp.servingOnce.Do(func() {
		started = true
		close(tp.serving)
	})
	return started
}

func (tp *Transport) close() (closed bool) {
	if tp == nil {
		return false
	}

	tp.doneOnce.Do(func() {
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

	tp.serve() // just in case.
	tp.ackWaiter.StopAll()
	return closed
}

// Close closes the dmsg_tp.
func (tp *Transport) Close() error {
	if tp == nil {
		return nil
	}
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

// HandleFrame allows 'tp.Serve' to handle the frame (typically from 'ClientConn').
func (tp *Transport) HandleFrame(f Frame) error {
	tp.inMx.Lock()
	defer tp.inMx.Unlock()

handleFrame:
	if tp.IsClosed() {
		return io.ErrClosedPipe
	}
	select {
	case tp.inCh <- f:
		return nil
	default:
		goto handleFrame
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
func (tp *Transport) WriteAccept() (err error) {
	defer func() {
		if err != nil {
			tp.log.WithError(err).WithField("remote", tp.remote).Warnln("(HANDSHAKE) Rejected locally.")
		} else {
			tp.log.WithField("remote", tp.remote).Infoln("(HANDSHAKE) Accepted locally.")
		}
	}()

	f := MakeFrame(AcceptType, tp.id, combinePKs(tp.remote, tp.local))
	if err = writeFrame(tp.Conn, f); err != nil {
		tp.close()
		return err
	}
	return nil
}

// ReadAccept awaits for an ACCEPT frame to be read from the remote client.
// TODO(evanlinjin): Cleanup errors.
func (tp *Transport) ReadAccept(ctx context.Context) (err error) {
	defer func() {
		if err != nil {
			tp.log.WithError(err).WithField("remote", tp.remote).Warnln("(HANDSHAKE) Rejected by remote.")
		} else {
			tp.log.WithField("remote", tp.remote).Infoln("(HANDSHAKE) Accepted by remote.")
		}
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

	// return is transport is already being served, or is closed
	if !tp.serve() {
		return
	}

	// ensure transport closes when serving stops
	// also write CLOSE frame if this is the first time 'close' is triggered
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
			log := tp.log.WithField("remoteClient", tp.remote).WithField("received", f)

			switch p := f.Pay(); f.Type() {
			case FwdType:
				if len(p) < 2 {
					log.Warnln("Rejected [FWD]: Invalid payload size.")
					return
				}

				tp.bufMx.Lock()

				// Acknowledgement logic: if read buffer has free space, send ACK. If not, add to 'ackBuf'.
				ack := MakeFrame(AckType, tp.id, p[:2])
				if tp.bufSize += len(p[2:]); tp.bufSize > tpBufCap {
					tp.ackBuf = append(tp.ackBuf, ack...)
				} else {
					go func() {
						if err := writeFrame(tp.Conn, ack); err != nil {
							tp.close()
						}
					}()
				}

				// add payload to 'buf'
				tp.buf = append(tp.buf, p[2:])

				// notify of new data via 'bufCh' (only if not closed)
				if !tp.IsClosed() {
					select {
					case tp.bufCh <- struct{}{}:
					default:
					}
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
				tp.close() // ensure there is no sending of CLOSE frame
				return

			case RequestType:
				log.Warnln("Rejected [REQUEST]: ID already occupied, possibly malicious server.")
				_ = tp.Conn.Close()
				return

			default:
				tp.log.Infof("Rejected [%s]: Unexpected frame, possibly malicious server (ignored for now).", f.Type())
			}
		}
	}
}

// Read implements io.Reader
// TODO(evanlinjin): read deadline.
func (tp *Transport) Read(p []byte) (n int, err error) {
	<-tp.serving

startRead:
	tp.bufMx.Lock()
	n, err = tp.buf.Read(p)
	if tp.bufSize -= n; tp.bufSize < tpBufCap && len(tp.ackBuf) > 0 {
		acks := tp.ackBuf
		tp.ackBuf = make([]byte, 0, tpAckCap)
		go func() {
			if err := writeFrame(tp.Conn, acks); err != nil {
				tp.close()
			}
		}()
	}
	tp.bufMx.Unlock()

	if tp.IsClosed() {
		return n, err
	}
	if n > 0 || len(p) == 0 {
		return n, nil
	}

	<-tp.bufCh
	goto startRead
}

// Write implements io.Writer
// TODO(evanlinjin): write deadline.
func (tp *Transport) Write(p []byte) (int, error) {
	<-tp.serving

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
