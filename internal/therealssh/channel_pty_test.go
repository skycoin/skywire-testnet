// +build !no_ci

package therealssh

import (
	"net"
	"os/user"
	"testing"
	"time"

	"github.com/skycoin/skywire/internal/appnet"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/pkg/cipher"
)

func TestChannelServe(t *testing.T) {
	in, out := net.Pipe()
	ch := OpenChannel(1, &appnet.LoopAddr{PubKey: cipher.PubKey{}, Port: Port}, in)

	errCh := make(chan error)
	go func() {
		errCh <- ch.Serve()
	}()

	req := appendU32([]byte{byte(RequestPTY)}, 10)
	req = appendU32(req, 20)
	req = appendU32(req, 0)
	req = appendU32(req, 0)
	u, err := user.Current()
	require.NoError(t, err)
	req = append(req, []byte(u.Username)...)
	ch.msgCh <- req
	time.Sleep(100 * time.Millisecond)

	buf := make([]byte, 6)
	_, err = out.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, byte(CmdChannelResponse), buf[0])
	assert.Equal(t, byte(ResponseConfirm), buf[5])

	require.NotNil(t, ch.session)

	ch.msgCh <- []byte{byte(RequestShell)}
	time.Sleep(100 * time.Millisecond)

	buf = make([]byte, 6)
	_, err = out.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, byte(CmdChannelResponse), buf[0])
	assert.Equal(t, byte(ResponseConfirm), buf[5])

	buf = make([]byte, 10)
	_, err = out.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, byte(CmdChannelData), buf[0])
	assert.NotNil(t, buf[5:])

	require.NotNil(t, ch.dataCh)
	ch.dataCh <- []byte("echo foo\n")
	time.Sleep(100 * time.Millisecond)

	buf = make([]byte, 15)
	_, err = out.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, byte(CmdChannelData), buf[0])
	assert.Contains(t, string(buf[5:]), "echo foo")

	buf = make([]byte, 15)
	_, err = out.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, byte(CmdChannelData), buf[0])
	assert.Contains(t, string(buf[5:]), "foo")

	req = appendU32([]byte{byte(RequestWindowChange)}, 40)
	req = appendU32(req, 50)
	req = appendU32(req, 0)
	req = appendU32(req, 0)
	ch.msgCh <- req
	time.Sleep(100 * time.Millisecond)

	buf = make([]byte, 6)
	_, err = out.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, byte(CmdChannelResponse), buf[0])
	assert.Equal(t, byte(ResponseConfirm), buf[5])

	require.NoError(t, ch.Close())
	require.NoError(t, <-errCh)
}
