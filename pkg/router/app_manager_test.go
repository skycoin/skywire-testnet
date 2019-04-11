package router

import (
	"net"
	"testing"

	"github.com/skycoin/skywire/internal/appnet"

	"github.com/skycoin/skycoin/src/util/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/pkg/app"
	"github.com/skycoin/skywire/pkg/cipher"
)

func TestAppManagerInit(t *testing.T) {
	in, out := net.Pipe()
	am := &appManager{
		logging.MustGetLogger("routesetup"),
		appnet.NewProtocol(out),
		&app.Config{AppName: "foo", AppVersion: "0.0.1"},
		nil,
	}

	srvCh := make(chan error)
	go func() { srvCh <- am.Serve() }()

	proto := appnet.NewProtocol(in)
	go proto.Serve(nil) // nolint: errcheck

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
			err := proto.Send(appnet.FrameInit, tc.conf, nil)
			require.Error(t, err)
			assert.Equal(t, tc.err, err.Error())
		})
	}

	err := proto.Send(appnet.FrameInit, &app.Config{AppName: "foo", AppVersion: "0.0.1", ProtocolVersion: "0.0.1"}, nil)
	require.NoError(t, err)

	require.NoError(t, in.Close())
	require.NoError(t, <-srvCh)
}

func TestAppManagerSetupLoop(t *testing.T) {
	in, out := net.Pipe()
	am := &appManager{
		logging.MustGetLogger("routesetup"),
		appnet.NewProtocol(out),
		&app.Config{AppName: "foo", AppVersion: "0.0.1"},
		&appCallbacks{
			CreateLoop: func(conn *appnet.Protocol, raddr *appnet.LoopAddr) (laddr *appnet.LoopAddr, err error) {
				return raddr, nil
			},
		},
	}

	srvCh := make(chan error)
	go func() { srvCh <- am.Serve() }()

	proto := appnet.NewProtocol(in)
	go proto.Serve(nil) // nolint: errcheck

	var laddr *appnet.LoopAddr
	pk, _ := cipher.GenerateKeyPair()
	raddr := &appnet.LoopAddr{PubKey: pk, Port: 3}
	err := proto.Send(appnet.FrameCreateLoop, raddr, &laddr)
	require.NoError(t, err)
	assert.Equal(t, raddr, laddr)

	require.NoError(t, in.Close())
	require.NoError(t, <-srvCh)
}

func TestAppManagerCloseLoop(t *testing.T) {
	in, out := net.Pipe()
	var inAddr *app.LoopMeta
	am := &appManager{
		logging.MustGetLogger("routesetup"),
		appnet.NewProtocol(out),
		&app.Config{AppName: "foo", AppVersion: "0.0.1"},
		&appCallbacks{
			CloseLoop: func(conn *appnet.Protocol, addr *app.LoopMeta) error {
				inAddr = addr
				return nil
			},
		},
	}

	srvCh := make(chan error)
	go func() { srvCh <- am.Serve() }()

	proto := appnet.NewProtocol(in)
	go proto.Serve(nil) // nolint: errcheck

	pk, _ := cipher.GenerateKeyPair()
	addr := &app.LoopMeta{LocalPort: 2, Remote: appnet.LoopAddr{PubKey: pk, Port: 3}}
	err := proto.Send(appnet.FrameCloseLoop, addr, nil)
	require.NoError(t, err)
	assert.Equal(t, addr, inAddr)

	require.NoError(t, in.Close())
	require.NoError(t, <-srvCh)
}

func TestAppManagerForward(t *testing.T) {
	in, out := net.Pipe()
	var inPacket *app.DataFrame
	am := &appManager{
		logging.MustGetLogger("routesetup"),
		appnet.NewProtocol(out),
		&app.Config{AppName: "foo", AppVersion: "0.0.1"},
		&appCallbacks{
			Forward: func(conn *appnet.Protocol, packet *app.DataFrame) error {
				inPacket = packet
				return nil
			},
		},
	}

	srvCh := make(chan error)
	go func() { srvCh <- am.Serve() }()

	proto := appnet.NewProtocol(in)
	go proto.Serve(nil) // nolint: errcheck

	pk, _ := cipher.GenerateKeyPair()
	packet := &app.DataFrame{Data: []byte("foo"), Meta: &app.LoopMeta{LocalPort: 2, Remote: appnet.LoopAddr{PubKey: pk, Port: 3}}}
	err := proto.Send(appnet.FrameData, packet, nil)
	require.NoError(t, err)
	assert.Equal(t, packet, inPacket)

	require.NoError(t, in.Close())
	require.NoError(t, <-srvCh)
}
