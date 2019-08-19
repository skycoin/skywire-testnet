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
	"github.com/skycoin/skywire/pkg/routing"
)

func TestAppManagerInit(t *testing.T) {
	in, out := net.Pipe()
	am := &appManager{
		logging.MustGetLogger("routesetup"),
		nil,
		app.NewProtocol(out),
		&app.Config{AppName: "foo", AppVersion: "0.0.1"},
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
		tc := tc
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
		Logger:  logging.MustGetLogger("routesetup"),
		router:  new(MockRouter),
		proto:   app.NewProtocol(out),
		appConf: &app.Config{AppName: "foo", AppVersion: "0.0.1"},
	}

	srvCh := make(chan error)
	go func() { srvCh <- am.Serve() }()

	proto := app.NewProtocol(in)
	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- proto.Serve(nil)
	}()

	var laddr routing.Addr
	pk, _ := cipher.GenerateKeyPair()
	raddr := routing.Addr{PubKey: pk, Port: 3}
	err := proto.Send(app.FrameCreateLoop, &raddr, &laddr)
	require.NoError(t, err)
	assert.Equal(t, raddr, laddr)

	require.NoError(t, in.Close())
	require.NoError(t, <-srvCh)
	require.NoError(t, proto.Close())
	require.NoError(t, testhelpers.NoErrorWithinTimeout(serveErrCh))
}

func TestAppManagerCloseLoop(t *testing.T) {
	in, out := net.Pipe()
	// var inLoop routing.AddrLoop
	r := new(MockRouter)
	am := &appManager{
		Logger:  logging.MustGetLogger("routesetup"),
		router:  r,
		proto:   app.NewProtocol(out),
		appConf: &app.Config{AppName: "foo", AppVersion: "0.0.1"},
		// &appCallbacks{
		// 	CloseLoop: func(conn *app.Protocol, loop routing.AddrLoop) error {
		// 		inLoop = loop
		// 		return nil
		// 	},
		// },
	}

	srvCh := make(chan error)
	go func() { srvCh <- am.Serve() }()

	proto := app.NewProtocol(in)
	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- proto.Serve(nil)
	}()

	lpk, _ := cipher.GenerateKeyPair()
	rpk, _ := cipher.GenerateKeyPair()
	loop := routing.AddrLoop{
		Local:  routing.Addr{PubKey: lpk, Port: 2},
		Remote: routing.Addr{PubKey: rpk, Port: 3}}
	err := proto.Send(app.FrameClose, loop, nil)
	require.NoError(t, err)
	assert.Equal(t, loop, r.inLoop)

	require.NoError(t, in.Close())
	require.NoError(t, <-srvCh)
	require.NoError(t, proto.Close())
	require.NoError(t, testhelpers.NoErrorWithinTimeout(serveErrCh))
}

func TestAppManagerForward(t *testing.T) {
	in, out := net.Pipe()
	r := new(MockRouter)
	am := &appManager{
		Logger:  logging.MustGetLogger("routesetup"),
		router:  r,
		proto:   app.NewProtocol(out),
		appConf: &app.Config{AppName: "foo", AppVersion: "0.0.1"},
	}

	srvCh := make(chan error)
	go func() { srvCh <- am.Serve() }()

	proto := app.NewProtocol(in)
	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- proto.Serve(nil)
	}()

	lpk, _ := cipher.GenerateKeyPair()
	rpk, _ := cipher.GenerateKeyPair()
	packet := &app.Packet{Payload: []byte("foo"),
		Loop: routing.AddrLoop{
			Local:  routing.Addr{PubKey: lpk, Port: 2},
			Remote: routing.Addr{PubKey: rpk, Port: 3}}}
	err := proto.Send(app.FrameSend, packet, nil)
	require.NoError(t, err)
	assert.Equal(t, packet, r.inPacket)

	require.NoError(t, in.Close())
	require.NoError(t, <-srvCh)
	require.NoError(t, proto.Close())
	require.NoError(t, testhelpers.NoErrorWithinTimeout(serveErrCh))
}
