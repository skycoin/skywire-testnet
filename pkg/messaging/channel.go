package messaging

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/skycoin/skywire/internal/noise"
	"github.com/skycoin/skywire/pkg/cipher"
	"github.com/skycoin/skywire/pkg/transport"
)

type channel struct {
	id   byte // This is to be changed.
	idMx sync.RWMutex

	remotePK cipher.PubKey
	link     *Link
	buf      *bytes.Buffer

	deadline time.Time
	closed   unsafe.Pointer // unsafe.Pointer is used alongside 'atomic' module for fast, thread-safe access.

	waitChan    chan bool // waits for remote response (whether channel is accepted or not).
	readChan    chan []byte
	closeChan   chan struct{}
	closeChanMx sync.RWMutex // TODO(evanlinjin): This is a hack to avoid race conditions when closing channel.
	doneChan    chan struct{}

	noise *noise.Noise
	rMx   sync.Mutex // lock for decrypt cipher state
	wMx   sync.Mutex // lock for encrypt cipher state
	pMx   sync.Mutex // for handshake and processMessage
}

func newChannel(initiator bool, secKey cipher.SecKey, remote cipher.PubKey, link *Link) (*channel, error) {
	noiseConf := noise.Config{
		LocalSK:   secKey,
		LocalPK:   link.Local(),
		RemotePK:  remote,
		Initiator: initiator,
	}
	noiseInstance, err := noise.KKAndSecp256k1(noiseConf)
	if err != nil {
		return nil, err
	}

	return &channel{
		remotePK:  remote,
		link:      link,
		buf:       new(bytes.Buffer),
		closed:    unsafe.Pointer(new(bool)), //nolint:gosec
		waitChan:  make(chan bool, 1),        // should allows receive one reply.
		readChan:  make(chan []byte),
		closeChan: make(chan struct{}),
		doneChan:  make(chan struct{}),
		noise:     noiseInstance,
	}, nil
}

// ID obtains the channel's id.
func (c *channel) ID() byte {
	c.idMx.RLock()
	id := c.id
	c.idMx.RUnlock()
	return id
}

// SetID set's the channel's id.
func (c *channel) SetID(id byte) {
	c.idMx.Lock()
	c.id = id
	c.idMx.Unlock()
}

// Edges returns the public keys of the channel's edge nodes
func (c *channel) Edges() [2]cipher.PubKey {
	return transport.SortPubKeys(c.link.Local(), c.remotePK)
}

// HandshakeMessage prepares a handshake message safely.
func (c *channel) HandshakeMessage() ([]byte, error) {
	c.pMx.Lock()
	defer c.pMx.Unlock()
	res, err := c.noise.HandshakeMessage()
	return res, err
}

// ProcessMessage reads a handshake message safely.
func (c *channel) ProcessMessage(msg []byte) error {
	c.pMx.Lock()
	defer c.pMx.Unlock()
	err := c.noise.ProcessMessage(msg)
	return err
}

func (c *channel) Read(p []byte) (n int, err error) {
	fmt.Println("inside Read previous lock")
	c.rMx.Lock()
	fmt.Println("inside Read after lock")
	defer c.rMx.Unlock()
	if c.buf.Len() != 0 {
		fmt.Println("Read buf len 0, returning previous")
		return c.buf.Read(p)
	}

	ctx := context.Background()
	var cancel context.CancelFunc
	if time.Until(c.deadline) > 0 {
		ctx, cancel = context.WithDeadline(ctx, c.deadline)
		defer cancel()
	}

	return c.readEncrypted(ctx, p)
}

func (c *channel) Write(p []byte) (n int, err error) {
	if c.isClosed() {
		return 0, ErrChannelClosed
	}

	ctx := context.Background()
	var cancel context.CancelFunc
	if time.Until(c.deadline) > 0 {
		ctx, cancel = context.WithDeadline(ctx, c.deadline)
		defer cancel()
	}

	c.wMx.Lock()
	defer c.wMx.Unlock()
	data := c.noise.EncryptUnsafe(p)

	buf := make([]byte, 2+len(data))
	binary.BigEndian.PutUint16(buf[:2], uint16(len(data)))
	copy(buf[2:], data)

	done := make(chan struct{}, 1)
	defer close(done)
	go func() {
		n, err = c.link.Send(c.ID(), buf)
		n = n - (len(data) - len(p) + 2)
		select {
		case done <- struct{}{}:
		default:
		}
	}()

	select {
	case <-done:
		return n, err
	case <-ctx.Done():
		return 0, ErrDeadlineExceeded
	}
}

func (c *channel) Close() error {
	if c.isClosed() {
		return ErrChannelClosed
	}

	if _, err := c.link.SendCloseChannel(c.ID()); err != nil {
		return err
	}

	c.setClosed(true)

	select {
	case <-c.closeChan:
	case <-time.After(time.Second):
	}

	c.close()
	return nil
}

func (c *channel) SetDeadline(t time.Time) error {
	c.deadline = t
	return nil
}

func (c *channel) Type() string {
	return "messaging"
}

func (c *channel) close() {
	select {
	case <-c.doneChan:
	default:
		close(c.doneChan)

		c.closeChanMx.Lock()   // TODO(evanlinjin): START(avoid race condition).
		close(c.closeChan)     // TODO(evanlinjin): data race.
		c.closeChanMx.Unlock() // TODO(evanlinjin): END(avoid race condition).
	}
}

func (c *channel) readEncrypted(ctx context.Context, p []byte) (n int, err error) {
	buf := new(bytes.Buffer)
	readAtLeast := func(d []byte) (int, error) {
		for {
			if buf.Len() >= len(d) {
				return buf.Read(d)
			}

			select {
			case <-c.doneChan:
				return 0, io.EOF
			case in, more := <-c.readChan:
				if !more {
					return 0, io.EOF
				}

				if _, err := buf.Write(in); err != nil {
					return 0, err
				}
			case <-ctx.Done():
				return 0, ErrDeadlineExceeded
			}
		}
	}

	size := make([]byte, 2)
	if _, err := readAtLeast(size); err != nil {
		return 0, err
	}

	encrypted := make([]byte, binary.BigEndian.Uint16(size))
	if _, err := readAtLeast(encrypted); err != nil {
		return 0, err
	}

	data, err := c.noise.DecryptUnsafe(encrypted)
	if err != nil {
		return 0, err
	}

	if len(data) > len(p) {
		if _, err := c.buf.Write(data[len(p):]); err != nil {
			return 0, io.ErrShortBuffer
		}

		return copy(p, data[:len(p)]), nil
	}

	return copy(p, data), nil
}

// for getting and setting the 'closed' status.
func (c *channel) isClosed() bool   { return *(*bool)(atomic.LoadPointer(&c.closed)) }
func (c *channel) setClosed(v bool) { atomic.StorePointer(&c.closed, unsafe.Pointer(&v)) } //nolint:gosec
