package router

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/skycoin/dmsg"
	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/skycoin/src/util/logging"
	"github.com/stretchr/testify/require"

	th "github.com/skycoin/skywire/internal/testhelpers"
	"github.com/skycoin/skywire/pkg/app"
	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/setup"
	"github.com/skycoin/skywire/pkg/transport"
)

func TestRouterCloseLoop(t *testing.T) {
	client := transport.NewDiscoveryMock()
	logStore := transport.InMemoryTransportLogStore()

	pk1, sk1 := cipher.GenerateKeyPair()
	pk2, sk2 := cipher.GenerateKeyPair()
	pk3, _ := cipher.GenerateKeyPair()

	f1, f2 := transport.NewMockFactoryPair(pk1, pk2)
	f1.SetType(dmsg.Type)

	m1, err := transport.NewManager(&transport.ManagerConfig{PubKey: pk1, SecKey: sk1, DiscoveryClient: client, LogStore: logStore, Logger: log}, f1)
	require.NoError(t, err)
	m1.SetSetupNodes([]cipher.PubKey{pk2})

	m2, err := transport.NewManager(&transport.ManagerConfig{PubKey: pk2, SecKey: sk2, DiscoveryClient: client, LogStore: logStore, Logger: log}, f2)
	require.NoError(t, err)
	m2.SetSetupNodes([]cipher.PubKey{pk1})

	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- m2.Serve(context.TODO())
	}()

	rt := routing.InMemoryRoutingTable()
	rule := routing.AppRule(time.Now().Add(time.Hour), 4, pk3, 6, 5)
	routeID, err := rt.AddRule(rule)
	require.NoError(t, err)

	conf := &Config{
		Logger:           logging.MustGetLogger("routesetup"),
		PubKey:           pk1,
		SecKey:           sk1,
		TransportManager: m1,
		RoutingTable:     rt,
		SetupNodes:       []cipher.PubKey{pk2},
	}
	r := New(conf)
	errCh := make(chan error)
	go func() {
		tr := <-m2.SetupTpChan

		proto := setup.NewSetupProtocol(tr)
		p, data, err := proto.ReadPacket()
		if err != nil {
			errCh <- err
			return
		}

		if p != setup.PacketCloseLoop {
			errCh <- errors.New("unknown command")
			return
		}

		var ld routing.LoopData
		if err := json.Unmarshal(data, &ld); err != nil {
			errCh <- err
			return
		}

		if ld.Loop.Local.Port != 5 || ld.Loop.Remote.Port != 6 || ld.Loop.Remote.PubKey != pk3 {
			errCh <- errors.New("invalid payload")
			return
		}

		errCh <- proto.WritePacket(setup.RespSuccess, []byte{})
	}()

	rw, rwIn := net.Pipe()
	serveAppErrCh := make(chan error, 1)
	go func() {
		serveAppErrCh <- r.ServeApp(rwIn, 5, &app.Config{})
	}()
	proto := app.NewProtocol(rw)
	protoServeErrCh := make(chan error, 1)
	go func() {
		protoServeErrCh <- proto.Serve(nil)
	}()

	time.Sleep(100 * time.Millisecond)

	raddr := routing.Addr{PubKey: pk3, Port: 6}
	require.NoError(t, r.pm.SetLoop(5, raddr, &loop{}))
	require.NoError(t, proto.Send(app.FrameClose, routing.AddressPair{Local: routing.Addr{Port: 5}, Remote: raddr}, nil))

	time.Sleep(100 * time.Millisecond)

	// Tests
	_ = routeID
	require.NoError(t, <-errCh)
	_, err = r.pm.GetLoop(5, routing.Addr{PubKey: pk3, Port: 6})
	require.Error(t, err)
	_, err = r.pm.Get(5)
	require.NoError(t, err)

	rule, err = rt.Rule(routeID)
	require.NoError(t, err)
	require.Nil(t, rule)

	require.NoError(t,
		th.NoErrorWithinTimeoutN(serveErrCh, serveAppErrCh, protoServeErrCh))
}

func TestRouterCloseLoopOnAppClose(t *testing.T) {
	client := transport.NewDiscoveryMock()
	logStore := transport.InMemoryTransportLogStore()

	pk1, sk1 := cipher.GenerateKeyPair()
	pk2, sk2 := cipher.GenerateKeyPair()
	pk3, _ := cipher.GenerateKeyPair()

	f1, f2 := transport.NewMockFactoryPair(pk1, pk2)
	f1.SetType(dmsg.Type)

	m1, err := transport.NewManager(&transport.ManagerConfig{PubKey: pk1, SecKey: sk1, DiscoveryClient: client, LogStore: logStore, Logger: log}, f1)
	require.NoError(t, err)
	m1.SetSetupNodes([]cipher.PubKey{pk2})

	m2, err := transport.NewManager(&transport.ManagerConfig{PubKey: pk2, SecKey: sk2, DiscoveryClient: client, LogStore: logStore, Logger: log}, f2)
	require.NoError(t, err)
	m2.SetSetupNodes([]cipher.PubKey{pk1})

	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- m2.Serve(context.TODO())
	}()

	rt := routing.InMemoryRoutingTable()
	rule := routing.AppRule(time.Now().Add(time.Hour), 4, pk3, 6, 5)
	routeID, err := rt.AddRule(rule)
	require.NoError(t, err)

	conf := &Config{
		Logger:           logging.MustGetLogger("routesetup"),
		PubKey:           pk1,
		SecKey:           sk1,
		TransportManager: m1,
		RoutingTable:     rt,
		SetupNodes:       []cipher.PubKey{pk2},
	}
	r := New(conf)
	errCh := make(chan error)
	go func() {
		tr := <-m2.SetupTpChan

		proto := setup.NewSetupProtocol(tr)
		p, data, err := proto.ReadPacket()
		if err != nil {
			errCh <- err
			return
		}

		if p != setup.PacketCloseLoop {
			errCh <- errors.New("unknown command")
			return
		}

		var ld routing.LoopData
		if err := json.Unmarshal(data, &ld); err != nil {
			errCh <- err
			return
		}

		if ld.Loop.Local.Port != 5 || ld.Loop.Remote.Port != 6 || ld.Loop.Remote.PubKey != pk3 {
			errCh <- errors.New("invalid payload")
			return
		}

		errCh <- proto.WritePacket(setup.RespSuccess, []byte{})
	}()

	rw, rwIn := net.Pipe()
	serveAppErrCh := make(chan error, 1)
	go func() {
		serveAppErrCh <- r.ServeApp(rwIn, 5, &app.Config{})
	}()
	proto := app.NewProtocol(rw)
	protoServeErrCh := make(chan error, 1)
	go func() {
		protoServeErrCh <- proto.Serve(nil)
	}()

	time.Sleep(100 * time.Millisecond)

	raddr := routing.Addr{PubKey: pk3, Port: 6}
	require.NoError(t, r.pm.SetLoop(5, raddr, &loop{}))
	require.NoError(t, rw.Close())

	time.Sleep(100 * time.Millisecond)

	require.NoError(t, <-errCh)
	_, err = r.pm.Get(5)
	require.Error(t, err)

	rule, err = rt.Rule(routeID)
	require.NoError(t, err)
	require.Nil(t, rule)

	require.NoError(t,
		th.NoErrorWithinTimeoutN(
			serveErrCh,
			serveAppErrCh,
			protoServeErrCh))
}
