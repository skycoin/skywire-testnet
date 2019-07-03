package therealssh

import (
	"encoding/binary"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/skycoin/dmsg/cipher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/pkg/app"
)

func TestChannelSendWrite(t *testing.T) {
	pk, _ := cipher.GenerateKeyPair()
	in, out := net.Pipe()
	ch := OpenChannel(1, &app.Addr{PubKey: pk, Port: Port}, in)

	errCh := make(chan error)
	go func() {
		errCh <- ch.Send(CmdChannelOpen, []byte("foo"))
	}()

	buf := make([]byte, 8)
	_, err := out.Read(buf)
	require.NoError(t, err)
	require.NoError(t, <-errCh)
	assert.Equal(t, byte(CmdChannelOpen), buf[0])
	assert.Equal(t, uint32(1), binary.BigEndian.Uint32(buf[1:]))
	assert.Equal(t, []byte("foo"), buf[5:])

	go func() {
		_, err := ch.Write([]byte("foo"))
		errCh <- err
	}()

	buf = make([]byte, 8)
	_, err = out.Read(buf)
	require.NoError(t, err)
	require.NoError(t, <-errCh)
	assert.Equal(t, byte(CmdChannelData), buf[0])
	assert.Equal(t, []byte("foo"), buf[5:])
}

func TestChannelRead(t *testing.T) {
	ch := OpenChannel(1, &app.Addr{PubKey: cipher.PubKey{}, Port: Port}, nil)

	buf := make([]byte, 3)
	go func() {
		ch.dataCh <- []byte("foo")
	}()

	_, err := ch.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, []byte("foo"), buf)

	close(ch.dataCh)
	_, err = ch.Read(buf)
	require.Error(t, err)
	assert.Equal(t, io.EOF, err)
}

func TestChannelRequest(t *testing.T) {
	in, out := net.Pipe()
	ch := OpenChannel(1, &app.Addr{PubKey: cipher.PubKey{}, Port: Port}, in)

	type data struct {
		res []byte
		err error
	}
	resCh := make(chan data)
	go func() {
		res, err := ch.Request(RequestPTY, []byte("foo"))
		resCh <- data{res, err}
	}()

	buf := make([]byte, 9)
	_, err := out.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, byte(CmdChannelOpen), buf[5])
	assert.Equal(t, []byte("foo"), buf[6:])

	ch.msgCh <- []byte{ResponseConfirm, 0x4}
	d := <-resCh
	require.NoError(t, d.err)
	assert.Equal(t, []byte{0x4}, d.res)
}

func TestChannelServeSocket(t *testing.T) {
	in, out := net.Pipe()
	ch := OpenChannel(1, &app.Addr{PubKey: cipher.PubKey{}, Port: Port}, in)

	assert.Equal(t, filepath.Join(os.TempDir(), "therealsshd-1"), ch.SocketPath())

	go func() { ch.ServeSocket() }() // nolint

	time.Sleep(100 * time.Millisecond)
	conn, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: ch.SocketPath(), Net: "unix"})
	require.NoError(t, err)

	_, err = conn.Write([]byte("foo"))
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)

	buf := make([]byte, 8)
	_, err = out.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, byte(CmdChannelData), buf[0])
	assert.Equal(t, []byte("foo"), buf[5:])

	ch.dataCh <- []byte("bar")
	time.Sleep(100 * time.Millisecond)

	buf = make([]byte, 3)
	_, err = conn.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, []byte("bar"), buf)

	require.NoError(t, ch.Close())
}
