package messaging

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"time"

	"github.com/skycoin/skywire/internal/ioutil"
	"github.com/skycoin/skywire/internal/noise"
	"github.com/skycoin/skywire/pkg/cipher"
)

type channel struct {
	ID       byte
	remotePK cipher.PubKey
	link     *Link
	buf      *bytes.Buffer

	deadline time.Time
	closed   bool

	waitChan  chan bool
	readChan  chan []byte
	closeChan chan struct{}
	doneChan  chan struct{}

	noise *noise.Noise
}

func (ch *channel) Edges() [2]cipher.PubKey {
	return [2]cipher.PubKey{ch.link.Local(), ch.remotePK}
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
		waitChan:  make(chan bool),
		readChan:  make(chan []byte),
		closeChan: make(chan struct{}),
		doneChan:  make(chan struct{}),
		noise:     noiseInstance,
	}, nil
}

func (c *channel) Read(p []byte) (n int, err error) {
	if c.buf.Len() != 0 {
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
	if c.closed {
		return 0, ErrChannelClosed
	}

	ctx := context.Background()
	var cancel context.CancelFunc
	if time.Until(c.deadline) > 0 {
		ctx, cancel = context.WithDeadline(ctx, c.deadline)
		defer cancel()
	}

	writeChan := make(chan struct {
		int
		error
	})
	go func() {
		data := c.noise.Encrypt(p)
		buf := make([]byte, 2)
		binary.BigEndian.PutUint16(buf, uint16(len(data)))
		n, err := c.link.Send(c.ID, append(buf, data...))
		writeChan <- struct {
			int
			error
		}{n - (len(data) - len(p) + 2), err}
	}()

	select {
	case w := <-writeChan:
		return w.int, w.error
	case <-ctx.Done():
		return 0, ErrDeadlineExceeded
	}
}

func (c *channel) Close() error {
	if c.closed {
		return ErrChannelClosed
	}

	if _, err := c.link.SendCloseChannel(c.ID); err != nil {
		return err
	}

	c.closed = true

	select {
	case <-c.closeChan:
	case <-time.After(time.Second):
	}

	c.close()
	return nil
}

func (c *channel) Local() cipher.PubKey {
	return c.link.Local()
}

func (c *channel) Remote() cipher.PubKey {
	return c.remotePK
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
		close(c.closeChan)
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

	data, err := c.noise.Decrypt(encrypted)
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

type ackedChannel struct {
	*channel
	rw *ioutil.AckReadWriter
}

func newAckedChannel(c *channel) *ackedChannel {
	return &ackedChannel{c, ioutil.NewAckReadWriter(c, 100*time.Millisecond)}
}

func (c *ackedChannel) Write(p []byte) (n int, err error) {
	return c.rw.Write(p)
}

func (c *ackedChannel) Read(p []byte) (n int, err error) {
	return c.rw.Read(p)
}

func (c *ackedChannel) Close() error {
	return c.rw.Close()
}
