// +build !no_ci

package therealssh

import (
	"net"
	"os/user"
	"testing"
	"time"

	"github.com/skycoin/dmsg/cipher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/pkg/routing"
)

func TestChannelServe(t *testing.T) {
	in, out := net.Pipe()
	ch := OpenChannel(1, routing.Addr{PubKey: cipher.PubKey{}, Port: Port}, in)

	errCh := make(chan error)
	go func() {
		errCh <- ch.Serve()
	}()

	bypassTests := 8 | 16

	var (
		err error
		req []byte
		buf []byte
	)

	t.Run("I", func(t *testing.T) {
		if bypassTests&1 == 1 {
			t.Skip()
		}
		req := appendU32([]byte{byte(RequestPTY)}, 10)
		req = appendU32(req, 20)
		req = appendU32(req, 0)
		req = appendU32(req, 0)
		u, err := user.Current()
		require.NoError(t, err)
		req = append(req, []byte(u.Username)...)
		ch.msgCh <- req
		time.Sleep(100 * time.Millisecond)
	})

	t.Run("II", func(t *testing.T) {
		if bypassTests&2 == 2 {
			t.Skip()
		}
		buf := make([]byte, 6)
		_, err = out.Read(buf)
		require.NoError(t, err)
		assert.EqualValues(t, CmdChannelResponse, buf[0])
		assert.EqualValues(t, ResponseConfirm, buf[5])

		require.NotNil(t, ch.session)

		ch.msgCh <- []byte{byte(RequestShell)}
		time.Sleep(100 * time.Millisecond)
	})

	t.Run("III", func(t *testing.T) {
		if bypassTests&4 == 4 {
			t.Skip()
		}
		buf := make([]byte, 6)
		_, err = out.Read(buf)
		require.NoError(t, err)
		assert.EqualValues(t, CmdChannelResponse, buf[0])
		assert.EqualValues(t, ResponseConfirm, buf[5])

		buf = make([]byte, 10)
		_, err = out.Read(buf)
		require.NoError(t, err)
		assert.EqualValues(t, CmdChannelData, buf[0])
		assert.NotNil(t, buf[5:])

		require.NotNil(t, ch.dataCh)
		ch.dataCh <- []byte("echo foo\n")
		time.Sleep(100 * time.Millisecond)
	})

	t.Run("IV", func(t *testing.T) {
		if bypassTests&8 == 8 {
			t.Skip()
		}
		buf = make([]byte, 15)
		_, err = out.Read(buf)
		require.NoError(t, err)
		assert.EqualValues(t, CmdChannelData, buf[0])
		assert.Contains(t, string(buf[5:]), "echo foo")

		buf = make([]byte, 15)
		_, err = out.Read(buf)
		require.NoError(t, err)
		assert.EqualValues(t, CmdChannelData, buf[0])
		assert.Contains(t, string(buf[5:]), "foo")

		req = appendU32([]byte{byte(RequestWindowChange)}, 40)
		req = appendU32(req, 50)
		req = appendU32(req, 0)
		req = appendU32(req, 0)
		ch.msgCh <- req
		time.Sleep(100 * time.Millisecond)
	})

	t.Run("V", func(t *testing.T) {
		if bypassTests&16 == 16 {
			t.Skip()
		}
		buf = make([]byte, 6)
		_, err = out.Read(buf)
		require.NoError(t, err)
		assert.EqualValues(t, CmdChannelResponse, buf[0])
		assert.EqualValues(t, ResponseConfirm, buf[5])

		require.NoError(t, ch.Close())
		require.NoError(t, <-errCh)
	})
}
