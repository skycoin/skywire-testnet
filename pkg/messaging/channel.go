package messaging

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"sync"
	"time"

	"github.com/skycoin/skywire/internal/noise"
	"github.com/skycoin/skywire/pkg/cipher"
	"github.com/skycoin/skywire/pkg/transport"
)

type msgChannel struct {
	id   byte // This is to be changed.
	idMx sync.RWMutex

	remotePK cipher.PubKey
	link     *Link
	buf      *bytes.Buffer

	deadline time.Time

	waitChan chan bool // waits for remote response (whether msgChannel is accepted or not).
	readChan chan []byte

	doneChan chan struct{}
	doneOnce sync.Once

	noise *noise.Noise
	rMx   sync.Mutex // lock for decrypt cipher state
	wMx   sync.Mutex // lock for encrypt cipher state
}

func newChannel(initiator bool, secKey cipher.SecKey, remote cipher.PubKey, link *Link) (*msgChannel, error) {
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

	return &msgChannel{
		remotePK: remote,
		link:     link,
		buf:      new(bytes.Buffer),
		waitChan: make(chan bool, 1), // should allows receive one reply.
		readChan: make(chan []byte),
		doneChan: make(chan struct{}),
		noise:    noiseInstance,
	}, nil
}

// ID obtains the msgChannel's id.
func (mCh *msgChannel) ID() byte {
	mCh.idMx.RLock()
	id := mCh.id
	mCh.idMx.RUnlock()
	return id
}

// SetID set's the msgChannel's id.
func (mCh *msgChannel) SetID(id byte) {
	mCh.idMx.Lock()
	mCh.id = id
	mCh.idMx.Unlock()
}

// Edges returns the public keys of the msgChannel's edge nodes
func (mCh *msgChannel) Edges() [2]cipher.PubKey {
	return transport.SortPubKeys(mCh.link.Local(), mCh.remotePK)
}

// HandshakeMessage prepares a handshake message safely.
func (mCh *msgChannel) HandshakeMessage() ([]byte, error) {
	mCh.rMx.Lock()
	mCh.wMx.Lock()
	res, err := mCh.noise.HandshakeMessage()
	mCh.rMx.Unlock()
	mCh.wMx.Unlock()
	return res, err
}

// ProcessMessage reads a handshake message safely.
func (mCh *msgChannel) ProcessMessage(msg []byte) error {
	mCh.rMx.Lock()
	mCh.wMx.Lock()
	err := mCh.noise.ProcessMessage(msg)
	mCh.rMx.Unlock()
	mCh.wMx.Unlock()
	return err
}

func (mCh *msgChannel) Read(p []byte) (n int, err error) {
	if mCh.buf.Len() != 0 {
		return mCh.buf.Read(p)
	}

	ctx := context.Background()
	var cancel context.CancelFunc
	if time.Until(mCh.deadline) > 0 {
		ctx, cancel = context.WithDeadline(ctx, mCh.deadline)
		defer cancel()
	}

	return mCh.readEncrypted(ctx, p)
}

func (mCh *msgChannel) Write(p []byte) (n int, err error) {
	select {
	case <-mCh.doneChan:
		return 0, ErrChannelClosed
	default:
	}

	ctx := context.Background()
	var cancel context.CancelFunc
	if time.Until(mCh.deadline) > 0 {
		ctx, cancel = context.WithDeadline(ctx, mCh.deadline)
		defer cancel()
	}

	mCh.wMx.Lock()
	defer mCh.wMx.Unlock()
	data := mCh.noise.EncryptUnsafe(p)

	buf := make([]byte, 2+len(data))
	binary.BigEndian.PutUint16(buf[:2], uint16(len(data)))
	copy(buf[2:], data)

	done := make(chan struct{}, 1)
	defer close(done)
	go func() {
		n, err = mCh.link.Send(mCh.ID(), buf)
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

func (mCh *msgChannel) Close() error {
	select {
	case <-mCh.doneChan:
		return ErrChannelClosed
	default:
	}

	if mCh.close() {
		if _, err := mCh.link.SendCloseChannel(mCh.ID()); err != nil {
			return err
		}
	}

	return nil
}

func (mCh *msgChannel) SetDeadline(t time.Time) error {
	mCh.deadline = t
	return nil
}

func (mCh *msgChannel) Type() string {
	return "messaging"
}

func (mCh *msgChannel) OnChannelClosed() bool {
	return mCh.close()
}

func (mCh *msgChannel) close() bool {
	closed := false
	mCh.doneOnce.Do(func() {
		close(mCh.doneChan)
		closed = true
	})
	return closed
}

func (mCh *msgChannel) readEncrypted(ctx context.Context, p []byte) (n int, err error) {
	mCh.rMx.Lock()
	defer mCh.rMx.Unlock()

	buf := new(bytes.Buffer)
	readAtLeast := func(d []byte) (int, error) {
		for {
			if buf.Len() >= len(d) {
				return buf.Read(d)
			}

			select {
			case <-mCh.doneChan:
				return 0, io.EOF
			case in, more := <-mCh.readChan:
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

	data, err := mCh.noise.DecryptUnsafe(encrypted)
	if err != nil {
		return 0, err
	}

	if len(data) > len(p) {
		if _, err := mCh.buf.Write(data[len(p):]); err != nil {
			return 0, io.ErrShortBuffer
		}

		return copy(p, data[:len(p)]), nil
	}

	return copy(p, data), nil
}
