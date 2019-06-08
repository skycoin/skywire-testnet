package dmsg

import (
	"bufio"
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

	pWrite *bufio.Writer
	pRead  *io.PipeReader
	mx     sync.RWMutex

	closed ioutil.AtomicBool
}

// NewTransport creates a new dms_tp.
func NewTransport(conn net.Conn, log *logging.Logger, local, remote cipher.PubKey, id uint16) *Transport {
	pRead, pWrite := io.Pipe()
	tp := &Transport{
		Conn:   conn,
		log:    log,
		id:     id,
		local:  local,
		remote: remote,
		pWrite: bufio.NewWriter(pWrite),
		pRead:  pRead,
	}
	if err := tp.ackWaiter.RandSeq(); err != nil {
		log.Fatalln("failed to set ack_waiter seq:", err)
	}
	tp.closed.Set(false)
	return tp
}

func (tp *Transport) close() bool {
	closed := tp.closed.Set(true)
	_ = tp.pRead.Close() //nolint:errcheck
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
	return tp.closed.Get()
}

// Edges returns the local/remote edges of the transport (dms_client to dms_client).
func (tp *Transport) Edges() [2]cipher.PubKey {
	return transport.SortPubKeys(tp.local, tp.remote)
}

// Type returns the transport type.
func (tp *Transport) Type() string {
	return Type
}

func (tp *Transport) WriteRequest() error {
	f := MakeFrame(RequestType, tp.id, combinePKs(tp.local, tp.remote))
	if err := writeFrame(tp.Conn, f); err != nil {
		tp.close()
		return err
	}
	return nil
}

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

func (tp *Transport) InjectFwd(d []byte) error {
	if tp.IsClosed() {
		return io.ErrClosedPipe
	}
	// TODO(evanlinjin): find a better solution.
	go func() {
		tp.mx.Lock()
		tp.pWrite.Write(d) //nolint:errcheck
		tp.mx.Unlock()
		go func() {
			tp.mx.Lock()
			tp.pWrite.Flush() //nolint:errcheck
			tp.mx.Unlock()
		}()
	}()
	return nil
}

func (tp *Transport) InjectAck(seq ioutil.Uint16Seq) error {
	if tp.IsClosed() {
		return io.ErrClosedPipe
	}
	tp.ackWaiter.Done(seq)
	return nil
}

// Read implements io.Reader
func (tp *Transport) Read(p []byte) (n int, err error) {
	return tp.pRead.Read(p)
}

// Write implements io.Writer
func (tp *Transport) Write(p []byte) (int, error) {
	if tp.IsClosed() {
		return 0, io.ErrClosedPipe
	}

	ctx, cancel := context.WithTimeout(context.Background(), readTimeout)
	defer cancel()

	err := tp.ackWaiter.Wait(ctx, func(seq ioutil.Uint16Seq) error {
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
