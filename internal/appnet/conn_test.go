package appnet

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPipeConn(t *testing.T) {
	srv, client, err := OpenPipeConn()
	require.NoError(t, err)

	t.Run("server can communicate with client", func(t *testing.T) {
		n, err := srv.Write([]byte("foo"))
		require.NoError(t, err)
		assert.Equal(t, 3, n)

		buf := make([]byte, 3)
		n, err = client.Read(buf)
		require.NoError(t, err)
		assert.Equal(t, 3, n)
		assert.Equal(t, []byte("foo"), buf)
	})

	t.Run("client can communicate with server", func(t *testing.T) {
		n, err := client.Write([]byte("foo"))
		require.NoError(t, err)
		assert.Equal(t, 3, n)

		buf := make([]byte, 3)
		n, err = srv.Read(buf)
		require.NoError(t, err)
		assert.Equal(t, 3, n)
		assert.Equal(t, []byte("foo"), buf)
	})

	t.Run("returns valid addresses", func(t *testing.T) {
		require.NotNil(t, srv.LocalAddr())
		require.NotNil(t, srv.RemoteAddr())

		require.NotNil(t, client.LocalAddr())
		require.NotNil(t, client.RemoteAddr())
	})

	t.Run("can set deadlines", func(t *testing.T) {
		deadline := time.Now().Add(500 * time.Millisecond)

		require.NoError(t, srv.SetDeadline(deadline))
		require.NoError(t, srv.SetReadDeadline(deadline))
		require.NoError(t, srv.SetWriteDeadline(deadline))

		require.NoError(t, client.SetDeadline(deadline))
		require.NoError(t, client.SetReadDeadline(deadline))
		require.NoError(t, client.SetWriteDeadline(deadline))
	})

	t.Run("deadline failures", func(t *testing.T) {
		buf := make([]byte, 4)
		_, err := srv.Read(buf)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "i/o timeout")

		_, err = client.Read(buf)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "i/o timeout")
	})

	t.Run("returns Fds", func(t *testing.T) {
		in, out := srv.Fd()
		assert.NotNil(t, in)
		assert.NotNil(t, out)

		in, out = client.Fd()
		assert.NotNil(t, in)
		assert.NotNil(t, out)
	})

	t.Run("can re-init from Fds", func(t *testing.T) {
		_, err := NewPipeConn(srv.Fd())
		assert.NoError(t, err)

		_, err = NewPipeConn(client.Fd())
		assert.NoError(t, err)
	})

	t.Run("can close", func(t *testing.T) {
		require.NoError(t, srv.Close())
		require.NoError(t, client.Close())
	})
}
