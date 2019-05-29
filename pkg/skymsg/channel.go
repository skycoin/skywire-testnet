package skymsg

import (
	"context"
	"encoding/binary"
	"errors"
	"github.com/skycoin/skywire/pkg/cipher"
	"github.com/skycoin/skywire/pkg/transport"
	"io"
	"net"
	"sync"
	"time"
)

const (
	hsTimeout  = time.Second * 10
	readBufLen = 10
	headerLen  = 5 // fType(1 byte), chID(2 byte), payLen(2 byte)
)

func isEven(chID uint16) bool {return chID % 2 == 0}

type FrameType byte

const (
	RequestType = FrameType(0)
	AcceptType  = FrameType(1)
	CloseType   = FrameType(2)
	FwdType     = FrameType(10)
)

type Frame []byte

func MakeFrame(ft FrameType, chID uint16, pay []byte) Frame {
	f := make(Frame, headerLen + len(pay))
	f[0] = byte(ft)
	binary.BigEndian.PutUint16(f[1:3], chID)
	binary.BigEndian.PutUint16(f[3:5], uint16(len(pay)))
	copy(f[5:], pay)
	return f
}

func (f Frame) Type()   FrameType {return FrameType(f[0])}
func (f Frame) ChID()   uint16    {return binary.BigEndian.Uint16(f[1:3])}
func (f Frame) PayLen() int       {return int(binary.BigEndian.Uint16(f[3:5]))}
func (f Frame) Pay()    []byte    {return f[headerLen:]}

func (f Frame) Disassemble() (ft FrameType, id uint16, p []byte) {
	return f.Type(), f.ChID(), f.Pay()
}

func readFrame(r io.Reader) (Frame, error) {
	f := make(Frame, headerLen)
	if _, err := io.ReadFull(r, f[:]); err != nil {
		return nil, err
	}
	f = append(f, make([]byte, f.PayLen())...)
	_, err := io.ReadFull(r, f[headerLen:])
	return f, err
}

func combinePKs(initPK, respPK cipher.PubKey) []byte {
	b := make([]byte, 66)
	copy(b[:33], initPK[:])
	copy(b[33:], respPK[:])
	return b
}

func splitPKs(b []byte) (initPK, respPK cipher.PubKey, ok bool) {
	pkLen := 66
	if len(b) != pkLen*2 {
		ok = false
		return
	}
	copy(initPK[:], b[:pkLen])
	copy(respPK[:], b[pkLen:])
	return initPK, respPK, true
}

// Channel is a channel from a client's perspective.
type Channel struct {
	net.Conn // link with server.
	id           uint16
	local        cipher.PubKey
	remoteClient cipher.PubKey // remote PK
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
		f := MakeFrame(RequestType, c.id, combinePKs(c.local, c.remoteClient))
		if _, err := c.Conn.Write(f); err != nil {
			c.close()
			return err
		}
		if err := c.awaitResponse(ctx); err != nil {
			c.close()
			return err
		}
	} else {
		f := MakeFrame(AcceptType, c.id, combinePKs(c.remoteClient, c.local))
		if _, err := c.Conn.Write(f); err != nil {
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
	return "skymsg"
}

func (c *Channel) Read(p []byte) (n int, err error) {
	select {
	case <-c.doneCh:
		return 0, io.ErrClosedPipe
	case f := <-c.readCh:
		switch f.Type() {
		case FwdType:
			if len(p) >= f.PayLen() {
				return copy(p, f.Pay()), nil
			}
			return 0, io.ErrShortBuffer
		case CloseType:
			if c.close() {
				_, _ = c.Conn.Write(MakeFrame(CloseType, c.id, []byte{0}))
			}
			return 0, io.ErrClosedPipe
		default:
			return 0, errors.New("unexpected frame")
		}
	}
}

func (c *Channel) Write(p []byte) (n int, err error) {
	select {
	case <-c.doneCh:
		return 0, io.ErrClosedPipe
	default:
		f := MakeFrame(FwdType, c.id, p)
		if _, err = c.Conn.Write(f); err != nil {
			c.close()
			f.Pay = nil
		}
		return f.PayLen(), err
	}
}

func (c *Channel) Close() error {
	if c.close() {
		_, _ = c.Conn.Write(MakeFrame(CloseType, c.id, []byte{0}))
		return nil
	}
	return io.ErrClosedPipe
}