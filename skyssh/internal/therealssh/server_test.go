package therealssh

import (
	"encoding/binary"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/pkg/cipher"

	"github.com/skycoin/skywire/pkg/app"
)

func TestServerOpenChannel(t *testing.T) {
	pk, _ := cipher.GenerateKeyPair()
	s := NewServer(&ListAuthorizer{[]cipher.PubKey{pk}})

	in, out := net.Pipe()
	errCh := make(chan error)
	go func() {
		errCh <- s.OpenChannel(&app.Addr{PubKey: cipher.PubKey{}, Port: Port}, 4, in)
	}()

	buf := make([]byte, 18)
	_, err := out.Read(buf)
	require.NoError(t, err)
	require.NoError(t, <-errCh)
	assert.Equal(t, byte(CmdChannelOpenResponse), buf[0])
	assert.Equal(t, byte(ResponseFail), buf[5])
	assert.Equal(t, []byte("unauthorised"), buf[6:])

	go func() {
		errCh <- s.OpenChannel(&app.Addr{PubKey: pk, Port: Port}, 4, in)
	}()

	buf = make([]byte, 10)
	_, err = out.Read(buf)
	require.NoError(t, err)
	require.NoError(t, <-errCh)
	assert.Equal(t, byte(CmdChannelOpenResponse), buf[0])
	assert.Equal(t, byte(ResponseConfirm), buf[5])
	assert.Equal(t, uint32(0), binary.BigEndian.Uint32(buf[6:]))
}

func TestServerHandleRequest(t *testing.T) {
	pk, _ := cipher.GenerateKeyPair()
	s := NewServer(&ListAuthorizer{[]cipher.PubKey{pk}})

	err := s.HandleRequest(pk, 0, []byte("foo"))
	require.Error(t, err)
	assert.Equal(t, "channel is not opened", err.Error())

	in, out := net.Pipe()
	ch := OpenChannel(4, &app.Addr{PubKey: pk, Port: Port}, in)
	s.chans.add(ch)

	errCh := make(chan error)
	go func() {
		errCh <- s.HandleRequest(cipher.PubKey{}, 0, []byte("foo"))
	}()

	buf := make([]byte, 18)
	_, err = out.Read(buf)
	require.NoError(t, err)
	require.NoError(t, <-errCh)
	assert.Equal(t, byte(CmdChannelResponse), buf[0])
	assert.Equal(t, byte(ResponseFail), buf[5])
	assert.Equal(t, []byte("unauthorised"), buf[6:])

	dataCh := make(chan []byte)
	go func() {
		dataCh <- <-ch.msgCh
	}()

	require.NoError(t, s.HandleRequest(pk, 0, []byte("foo")))
	assert.Equal(t, []byte("foo"), <-dataCh)
}

func TestServerHandleData(t *testing.T) {
	pk, _ := cipher.GenerateKeyPair()
	s := NewServer(&ListAuthorizer{[]cipher.PubKey{pk}})

	err := s.HandleData(pk, 0, []byte("foo"))
	require.Error(t, err)
	assert.Equal(t, "channel is not opened", err.Error())

	ch := OpenChannel(4, &app.Addr{PubKey: pk, Port: Port}, nil)
	s.chans.add(ch)

	err = s.HandleData(cipher.PubKey{}, 0, []byte("foo"))
	require.Error(t, err)
	assert.Equal(t, "unauthorised", err.Error())

	err = s.HandleData(pk, 0, []byte("foo"))
	require.Error(t, err)
	assert.Equal(t, "session is not started", err.Error())

	ch.session = &Session{}
	dataCh := make(chan []byte)
	go func() {
		dataCh <- <-ch.dataCh
	}()

	require.NoError(t, s.HandleData(pk, 0, []byte("foo")))
	assert.Equal(t, []byte("foo"), <-dataCh)
}
