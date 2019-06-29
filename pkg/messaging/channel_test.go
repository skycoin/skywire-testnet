package messaging

import (
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"

	"github.com/skycoin/dmsg/cipher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/internal/noise"
)

func TestChannelRead(t *testing.T) {
	remotePK, remoteSK := cipher.GenerateKeyPair()
	pk, sk := cipher.GenerateKeyPair()

	in, _ := net.Pipe()
	l, err := NewLink(in, &LinkConfig{Public: pk}, nil)
	require.NoError(t, err)

	c, err := newChannel(true, sk, remotePK, l)
	require.NoError(t, err)

	rn := handshakeChannel(t, c, remotePK, remoteSK)

	buf := make([]byte, 2)
	c.SetDeadline(time.Now().Add(100 * time.Millisecond)) // nolint
	_, err = c.Read(buf)
	require.Equal(t, ErrDeadlineExceeded, err)

	go func() {
		data := rn.EncryptUnsafe([]byte("foo"))
		buf := make([]byte, 2)
		binary.BigEndian.PutUint16(buf, uint16(len(data)))
		buf = append(buf, data...)
		c.readChan <- buf[0:3]
		c.readChan <- buf[3:]

		data = rn.EncryptUnsafe([]byte("foo"))
		buf = make([]byte, 2)
		binary.BigEndian.PutUint16(buf, uint16(len(data)))
		buf = append(buf, data...)
		c.readChan <- buf
		c.close()
	}()

	buf = make([]byte, 3)
	n, err := c.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Equal(t, []byte("foo"), buf)

	buf = make([]byte, 2)
	n, err = c.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, 2, n)
	assert.Equal(t, []byte("fo"), buf)

	buf = make([]byte, 2)
	n, err = c.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, []byte("o"), buf[:n])

	_, err = c.Read(buf)
	require.Equal(t, io.EOF, err)
}

func TestChannelWrite(t *testing.T) {
	remotePK, remoteSK := cipher.GenerateKeyPair()
	pk, sk := cipher.GenerateKeyPair()

	in, out := net.Pipe()

	l, err := NewLink(in, &LinkConfig{Public: pk}, nil)
	require.NoError(t, err)
	c, err := newChannel(true, sk, remotePK, l)
	require.NoError(t, err)
	c.SetID(10)

	rn := handshakeChannel(t, c, remotePK, remoteSK)

	var (
		readBuf  = make([]byte, 29)
		readErr  error
		readDone = make(chan struct{})
	)
	go func() {
		_, readErr = out.Read(readBuf)
		close(readDone)
	}()

	n, err := c.Write([]byte("foo"))
	require.NoError(t, err)
	assert.Equal(t, 3, n)

	<-readDone
	assert.NoError(t, readErr)
	assert.Equal(t, FrameTypeSend, FrameType(readBuf[2]))
	assert.Equal(t, byte(10), readBuf[3])

	// Encoded length should be length of encrypted payload "foo".
	require.Equal(t, uint16(23), binary.BigEndian.Uint16(readBuf[4:]))

	data, err := rn.DecryptUnsafe(readBuf[6:])
	require.NoError(t, err)
	assert.Equal(t, []byte("foo"), data)

	c.SetDeadline(time.Now().Add(100 * time.Millisecond)) // nolint
	_, err = c.Write([]byte("foo"))
	require.Equal(t, ErrDeadlineExceeded, err)

	closed := c.close()
	require.True(t, closed)

	_, err = c.Write([]byte("foo"))
	require.Equal(t, ErrChannelClosed, err)
}

func TestChannelClose(t *testing.T) {
	remotePK, remoteSK := cipher.GenerateKeyPair()
	pk, sk := cipher.GenerateKeyPair()

	in, out := net.Pipe()
	l, err := NewLink(in, &LinkConfig{Public: pk}, nil)
	require.NoError(t, err)

	c, err := newChannel(true, sk, remotePK, l)
	require.NoError(t, err)
	c.SetID(10)

	handshakeChannel(t, c, remotePK, remoteSK)

	require.NoError(t, c.SetDeadline(time.Now().Add(100*time.Millisecond)))

	buf := make([]byte, 4)
	go out.Read(buf) // nolint
	require.NoError(t, c.Close())
	assert.Equal(t, FrameTypeCloseChannel, FrameType(buf[2]))
	assert.Equal(t, byte(10), buf[3])
}

func handshakeChannel(t *testing.T, mCh *msgChannel, pk cipher.PubKey, sk cipher.SecKey) *noise.Noise {
	t.Helper()

	noiseConf := noise.Config{
		LocalSK:   sk,
		LocalPK:   pk,
		RemotePK:  mCh.link.Local(),
		Initiator: false,
	}

	n, err := noise.KKAndSecp256k1(noiseConf)
	require.NoError(t, err)

	msg, err := mCh.noise.HandshakeMessage()
	require.NoError(t, err)

	require.NoError(t, n.ProcessMessage(msg))
	msg, err = n.HandshakeMessage()
	require.NoError(t, err)

	require.NoError(t, mCh.noise.ProcessMessage(msg))
	return n
}
