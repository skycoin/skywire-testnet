package dmsg

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/SkycoinProject/skycoin/src/util/logging"

	"github.com/SkycoinProject/dmsg/cipher"
	"github.com/SkycoinProject/dmsg/ioutil"
)

// Errors related to REQUEST frames.
var (
	ErrRequestRejected    = errors.New("failed to create transport: request rejected")
	ErrRequestCheckFailed = errors.New("failed to create transport: request check failed")
	ErrAcceptCheckFailed  = errors.New("failed to create transport: accept check failed")
	ErrPortNotListening   = errors.New("failed to create transport: port not listening")
)

// Transport represents communication between two nodes via a single hop:
// a connection from dmsg.Client to remote dmsg.Client (via dmsg.Server intermediary).
type Transport struct {
	net.Conn // underlying connection to dmsg.Server
	log      *logging.Logger

	id     uint16 // tp ID that identifies this dmsg.transport
	local  Addr   // local PK
	remote Addr   // remote PK

	inCh chan Frame // handles incoming frames (from dmsg.Client)
	inMx sync.Mutex // protects 'inCh'

	ackWaiter ioutil.Uint16AckWaiter // awaits for associated ACK frames
	ackBuf    []byte                 // buffer for unsent ACK frames
	buf       net.Buffers            // buffer for non-read FWD frames
	bufCh     chan struct{}          // chan for indicating whether this is a new FWD frame
	bufSize   int                    // keeps track of the total size of 'buf'
	bufMx     sync.Mutex             // protects fields responsible for handling FWD and ACK frames
	rMx       sync.Mutex             // TODO: (WORKAROUND) concurrent reads seem problematic right now.

	serving     chan struct{}   // chan which closes when serving begins
	servingOnce sync.Once       // ensures 'serving' only closes once
	done        chan struct{}   // chan which closes when transport stops serving
	doneOnce    sync.Once       // ensures 'done' only closes once
	doneFunc    func(id uint16) // contains a method to remove the transport from dmsg.Client
}

// NewTransport creates a new dms_tp.
func NewTransport(conn net.Conn, log *logging.Logger, local, remote Addr, id uint16, doneFunc func(id uint16)) *Transport {
	tp := &Transport{
		Conn:      conn,
		log:       log,
		id:        id,
		local:     local,
		remote:    remote,
		inCh:      make(chan Frame),
		ackWaiter: ioutil.NewUint16AckWaiter(),
		ackBuf:    make([]byte, 0, tpAckCap),
		buf:       make(net.Buffers, 0, tpBufFrameCap),
		bufCh:     make(chan struct{}, 1),
		serving:   make(chan struct{}),
		done:      make(chan struct{}),
		doneFunc:  doneFunc,
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

// Regarding the use of mutexes:
// 1. `done` is always closed before `inCh`/`bufCh` is closed.
// 2. mutexes protect `inCh`/`bufCh` to ensure that closing and writing to these chans does not happen concurrently.
// 3. Our worry now, is writing to `inCh`/`bufCh` AFTER they have been closed.
// 4. But as, under the mutexes protecting `inCh`/`bufCh`, checking `done` comes first,
// and we know that `done` is closed before `inCh`/`bufCh`, we can guarantee that it avoids writing to closed chan.
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
	if tp.close() {
		if err := writeCloseFrame(tp.Conn, tp.id, PlaceholderReason); err != nil {
			log.WithError(err).Warn("Failed to write frame")
		}
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

// LocalPK returns the local public key of the transport.
func (tp *Transport) LocalPK() cipher.PubKey {
	return tp.local.PK
}

// RemotePK returns the remote public key of the transport.
func (tp *Transport) RemotePK() cipher.PubKey {
	return tp.remote.PK
}

// LocalAddr returns local address in from <public-key>:<port>
func (tp *Transport) LocalAddr() net.Addr { return tp.local }

// RemoteAddr returns remote address in form <public-key>:<port>
func (tp *Transport) RemoteAddr() net.Addr { return tp.remote }

// Type returns the transport type.
func (tp *Transport) Type() string {
	return Type
}

// HandleFrame allows 'tp.Serve' to handle the frame (typically from 'ClientConn').
func (tp *Transport) HandleFrame(f Frame) error {
	tp.inMx.Lock()
	defer tp.inMx.Unlock()
	for {
		if tp.IsClosed() {
			return io.ErrClosedPipe
		}
		select {
		case tp.inCh <- f:
			return nil
		default:
		}
	}
}

// WriteRequest writes a REQUEST frame to dmsg_server to be forwarded to associated client.
func (tp *Transport) WriteRequest(port uint16) error {
	payload := HandshakePayload{
		Version: HandshakePayloadVersion,
		InitPK:  tp.local.PK,
		RespPK:  tp.remote.PK,
		Port:    port,
	}
	payloadBytes, err := marshalHandshakePayload(payload)
	if err != nil {
		return err
	}
	f := MakeFrame(RequestType, tp.id, payloadBytes)
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

	f := MakeFrame(AcceptType, tp.id, combinePKs(tp.remote.PK, tp.local.PK))
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
		if err := tp.Close(); err != nil {
			log.WithError(err).Warn("Failed to close transport")
		}
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
			if !ok || initPK != tp.local.PK || respPK != tp.remote.PK || !isInitiatorID(id) {
				if err := tp.Close(); err != nil {
					log.WithError(err).Warn("Failed to close transport")
				}
				return ErrAcceptCheckFailed
			}
			return nil

		case CloseType:
			tp.close()
			return ErrRequestRejected

		default:
			if err := tp.Close(); err != nil {
				log.WithError(err).Warn("Failed to close transport")
			}
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
			if err := writeCloseFrame(tp.Conn, tp.id, PlaceholderReason); err != nil {
				log.WithError(err).Warn("Failed to write close frame")
			}
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
				pay := p[2:]
				tp.buf = append(tp.buf, pay)

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
				if err := tp.Conn.Close(); err != nil {
					log.WithError(err).Warn("Failed to close connection")
				}
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

	tp.rMx.Lock()
	defer tp.rMx.Unlock()

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

	if n > 0 || len(p) == 0 {
		if !tp.IsClosed() {
			err = nil
		}
		return n, err
	}

	if _, ok := <-tp.bufCh; !ok {
		return n, err
	}
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
