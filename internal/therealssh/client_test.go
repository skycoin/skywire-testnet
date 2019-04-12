package therealssh

import (
	"encoding/binary"
	"net"
	"testing"
	"time"

	"github.com/skycoin/skywire/pkg/app"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/pkg/cipher"
)

func TestClientOpenChannel(t *testing.T) {
	pk, _ := cipher.GenerateKeyPair()
	conn, dialer := newPipeDialer()
	c := &Client{dial: dialer.Dial, chans: newChanList()}

	type data struct {
		ch  *Channel
		err error
	}
	resCh := make(chan data)
	go func() {
		_, ch, err := c.OpenChannel(pk)
		resCh <- data{ch, err}
	}()

	time.Sleep(100 * time.Millisecond)
	buf := make([]byte, 5)
	_, err := conn.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, byte(CmdChannelOpen), buf[0])
	assert.Equal(t, uint32(0), binary.BigEndian.Uint32(buf[1:]))

	ch := c.chans.getChannel(0)
	require.NotNil(t, ch)
	ch.msgCh <- appendU32([]byte{ResponseConfirm}, 4)

	d := <-resCh
	require.NoError(t, d.err)
	require.NotNil(t, d.ch)
	assert.Equal(t, uint32(4), d.ch.RemoteID)
	assert.Equal(t, pk, d.ch.RemoteAddr.PubKey)
}

func TestClientHandleResponse(t *testing.T) {
	pk, _ := cipher.GenerateKeyPair()
	c := &Client{nil, newChanList()}
	in, out := net.Pipe()
	errCh := make(chan error)
	go func() {
		errCh <- c.serveConn(&mockConn{out, &app.LoopAddr{PubKey: pk, Port: Port}})
	}()

	_, err := in.Write(appendU32([]byte{byte(CmdChannelResponse)}, 0))
	require.NoError(t, err)
	assert.Equal(t, "channel is not opened", (<-errCh).Error())

	go func() {
		errCh <- c.serveConn(&mockConn{out, &app.LoopAddr{PubKey: cipher.PubKey{}, Port: Port}})
	}()

	ch := OpenChannel(4, &app.LoopAddr{PubKey: pk, Port: Port}, nil)
	c.chans.add(ch)

	_, err = in.Write(appendU32([]byte{byte(CmdChannelResponse)}, 0))
	require.NoError(t, err)
	assert.Equal(t, "unauthorized", (<-errCh).Error())

	go func() {
		errCh <- c.serveConn(&mockConn{out, &app.LoopAddr{PubKey: pk, Port: Port}})
	}()
	dataCh := make(chan []byte)
	go func() {
		dataCh <- <-ch.msgCh
	}()

	data := append(appendU32([]byte{byte(CmdChannelResponse)}, 0), []byte("foo")...)
	_, err = in.Write(data)
	require.NoError(t, err)
	assert.Equal(t, []byte("foo"), <-dataCh)
}

func TestClientHandleData(t *testing.T) {
	pk, _ := cipher.GenerateKeyPair()
	c := &Client{nil, newChanList()}
	in, out := net.Pipe()
	errCh := make(chan error)
	go func() {
		errCh <- c.serveConn(&mockConn{out, &app.LoopAddr{PubKey: pk, Port: Port}})
	}()

	_, err := in.Write(appendU32([]byte{byte(CmdChannelData)}, 0))
	require.NoError(t, err)
	assert.Equal(t, "channel is not opened", (<-errCh).Error())

	go func() {
		errCh <- c.serveConn(&mockConn{out, &app.LoopAddr{PubKey: cipher.PubKey{}, Port: Port}})
	}()

	ch := OpenChannel(4, &app.LoopAddr{PubKey: pk, Port: Port}, nil)
	c.chans.add(ch)

	_, err = in.Write(appendU32([]byte{byte(CmdChannelData)}, 0))
	require.NoError(t, err)
	assert.Equal(t, "unauthorized", (<-errCh).Error())

	go func() {
		errCh <- c.serveConn(&mockConn{out, &app.LoopAddr{PubKey: pk, Port: Port}})
	}()
	dataCh := make(chan []byte)
	go func() {
		dataCh <- <-ch.dataCh
	}()

	data := append(appendU32([]byte{byte(CmdChannelData)}, 0), []byte("foo")...)
	_, err = in.Write(data)
	require.NoError(t, err)
	assert.Equal(t, []byte("foo"), <-dataCh)
}

type pipeDialer struct {
	conn net.Conn
}

func newPipeDialer() (net.Conn, *pipeDialer) {
	in, out := net.Pipe()
	return out, &pipeDialer{in}
}

func (d *pipeDialer) Dial(raddr app.LoopAddr) (net.Conn, error) {
	return d.conn, nil
}

type mockConn struct {
	net.Conn
	addr *app.LoopAddr
}

func (conn *mockConn) RemoteAddr() net.Addr {
	return conn.addr
}
