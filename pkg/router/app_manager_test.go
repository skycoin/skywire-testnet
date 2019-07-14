package router

import (
	"net"
	"testing"

	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/skycoin/src/util/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/internal/testhelpers"
	"github.com/skycoin/skywire/pkg/app"
)

func TestAppManagerInit(t *testing.T) {
	in, out := net.Pipe()
	am := &appManager{
		logging.MustGetLogger("routesetup"),
		app.NewProtocol(out),
		&app.Config{AppName: "foo", AppVersion: "0.0.1"},
		nil,
	}

	srvCh := make(chan error)
	go func() { srvCh <- am.Serve() }()

	proto := app.NewProtocol(in)
	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- proto.Serve(nil)
	}()

	tcs := []struct {
		conf *app.Config
		err  string
	}{
		{&app.Config{AppName: "foo", AppVersion: "0.0.1", ProtocolVersion: "0.0.2"}, "unsupported protocol version"},
		{&app.Config{AppName: "foo", AppVersion: "0.0.2", ProtocolVersion: "0.0.1"}, "unexpected app version"},
		{&app.Config{AppName: "bar", AppVersion: "0.0.1", ProtocolVersion: "0.0.1"}, "unexpected app"},
	}

	for _, tc := range tcs {
		t.Run(tc.err, func(t *testing.T) {
			err := proto.Send(app.FrameInit, tc.conf, nil)
			require.Error(t, err)
			assert.Equal(t, tc.err, err.Error())
		})
	}

	err := proto.Send(app.FrameInit, &app.Config{AppName: "foo", AppVersion: "0.0.1", ProtocolVersion: "0.0.1"}, nil)
	require.NoError(t, err)

	require.NoError(t, in.Close())
	require.NoError(t, <-srvCh)
	require.NoError(t, proto.Close())
	require.NoError(t, testhelpers.NoErrorWithinTimeout(serveErrCh))
}

func TestAppManagerSetupLoop(t *testing.T) {
	in, out := net.Pipe()
	am := &appManager{
		logging.MustGetLogger("routesetup"),
		app.NewProtocol(out),
		&app.Config{AppName: "foo", AppVersion: "0.0.1"},
		&appCallbacks{
			CreateLoop: func(conn *app.Protocol, raddr *app.Addr) (laddr *app.Addr, err error) {
				return raddr, nil
			},
		},
	}

	srvCh := make(chan error)
	go func() { srvCh <- am.Serve() }()

	proto := app.NewProtocol(in)
	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- proto.Serve(nil)
	}()

	var laddr *app.Addr
	pk, _ := cipher.GenerateKeyPair()
	raddr := &app.Addr{PubKey: pk, Port: 3}
	err := proto.Send(app.FrameCreateLoop, raddr, &laddr)
	require.NoError(t, err)
	assert.Equal(t, raddr, laddr)

	require.NoError(t, in.Close())
	require.NoError(t, <-srvCh)
	require.NoError(t, proto.Close())
	require.NoError(t, testhelpers.NoErrorWithinTimeout(serveErrCh))
}

func TestAppManagerCloseLoop(t *testing.T) {
	in, out := net.Pipe()
	var inAddr *app.LoopAddr
	am := &appManager{
		logging.MustGetLogger("routesetup"),
		app.NewProtocol(out),
		&app.Config{AppName: "foo", AppVersion: "0.0.1"},
		&appCallbacks{
			CloseLoop: func(conn *app.Protocol, addr *app.LoopAddr) error {
				inAddr = addr
				return nil
			},
		},
	}

	srvCh := make(chan error)
	go func() { srvCh <- am.Serve() }()

	proto := app.NewProtocol(in)
	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- proto.Serve(nil)
	}()

	pk, _ := cipher.GenerateKeyPair()
	addr := &app.LoopAddr{Port: 2, Remote: app.Addr{PubKey: pk, Port: 3}}
	err := proto.Send(app.FrameClose, addr, nil)
	require.NoError(t, err)
	assert.Equal(t, addr, inAddr)

	require.NoError(t, in.Close())
	require.NoError(t, <-srvCh)
	require.NoError(t, proto.Close())
	require.NoError(t, testhelpers.NoErrorWithinTimeout(serveErrCh))
}

func TestAppManagerForward(t *testing.T) {
	in, out := net.Pipe()
	var inPacket *app.Packet
	am := &appManager{
		logging.MustGetLogger("routesetup"),
		app.NewProtocol(out),
		&app.Config{AppName: "foo", AppVersion: "0.0.1"},
		&appCallbacks{
			Forward: func(conn *app.Protocol, packet *app.Packet) error {
				inPacket = packet
				return nil
			},
		},
	}

	srvCh := make(chan error)
	go func() { srvCh <- am.Serve() }()

	proto := app.NewProtocol(in)
	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- proto.Serve(nil)
	}()

	pk, _ := cipher.GenerateKeyPair()
	packet := &app.Packet{Payload: []byte("foo"), Addr: &app.LoopAddr{Port: 2, Remote: app.Addr{PubKey: pk, Port: 3}}}
	err := proto.Send(app.FrameSend, packet, nil)
	require.NoError(t, err)
	assert.Equal(t, packet, inPacket)

	require.NoError(t, in.Close())
	require.NoError(t, <-srvCh)
	require.NoError(t, proto.Close())
	require.NoError(t, testhelpers.NoErrorWithinTimeout(serveErrCh))
}
