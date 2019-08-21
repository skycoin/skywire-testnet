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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	th "github.com/skycoin/skywire/internal/testhelpers"
	"github.com/skycoin/skywire/pkg/app"
	routeFinder "github.com/skycoin/skywire/pkg/route-finder/client"
	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/setup"
	"github.com/skycoin/skywire/pkg/transport"
)

func TestRouterSetup(t *testing.T) {
	// Environment
	client := transport.NewDiscoveryMock()
	logStore := transport.InMemoryTransportLogStore()

	pk1, sk1 := cipher.GenerateKeyPair()
	pk2, sk2 := cipher.GenerateKeyPair()

	c1 := &transport.ManagerConfig{PubKey: pk1, SecKey: sk1, DiscoveryClient: client, LogStore: logStore, Logger: log}
	c2 := &transport.ManagerConfig{PubKey: pk2, SecKey: sk2, DiscoveryClient: client, LogStore: logStore, Logger: log}

	f1, f2 := transport.NewMockFactoryPair(pk1, pk2)
	m1, err := transport.NewManager(c1, f1)
	require.NoError(t, err)

	m2, err := transport.NewManager(c2, f2)
	require.NoError(t, err)

	m1.SetSetupNodes([]cipher.PubKey{pk2})

	rt := routing.InMemoryRoutingTable()
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
		errCh <- r.Serve(context.TODO())
	}()

	tr, err := m2.CreateSetupTransport(context.TODO(), pk1, "mock")
	require.NoError(t, err)
	trID := transport.MakeTransportID(tr.LocalPK(), tr.RemotePK(), tr.Type(), false)
	sProto := setup.NewSetupProtocol(tr)

	rw1, rwIn1 := net.Pipe()
	serveAppErr1Ch := make(chan error, 1)
	go func() {
		serveAppErr1Ch <- r.ServeApp(rwIn1, 2, &app.Config{})
	}()
	appProto1 := app.NewProtocol(rw1)
	dataCh := make(chan []byte)
	protoServeErr1Ch := make(chan error, 1)
	go func() {
		f := func(_ app.Frame, p []byte) (interface{}, error) {
			go func() { dataCh <- p }()
			return nil, nil
		}
		protoServeErr1Ch <- appProto1.Serve(f)
	}()

	rw2, rwIn2 := net.Pipe()
	serveAppErr2Ch := make(chan error, 1)
	go func() {
		serveAppErr2Ch <- r.ServeApp(rwIn2, 4, &app.Config{})
	}()
	appProto2 := app.NewProtocol(rw2)
	protoServeErr2Ch := make(chan error, 1)
	go func() {
		f := func(_ app.Frame, p []byte) (interface{}, error) {
			go func() { dataCh <- p }()
			return nil, nil
		}
		protoServeErr2Ch <- appProto2.Serve(f)
	}()

	// Start of tests

	skiptests := 0 //| 1 | 2 | 4 | 8 | 16

	var routeID routing.RouteID
	t.Run("add rule", func(t *testing.T) {
		if skiptests&1 == 1 {
			t.Skip("skipping add rule")
		}
		routeID, err = setup.AddRule(sProto, routing.ForwardRule(time.Now().Add(time.Hour), 2, trID))
		require.NoError(t, err)

		rule, err := rt.Rule(routeID)
		require.NoError(t, err)
		assert.Equal(t, routing.RouteID(2), rule.RouteID())
		assert.Equal(t, trID, rule.TransportID())
	})

	t.Run("confirm loop - responder", func(t *testing.T) {
		if skiptests&2 == 2 {
			t.Skip("skipping confirm loop - responder")
		}

		appRouteID, err := setup.AddRule(sProto, routing.AppRule(time.Now().Add(time.Hour), 0, pk2, 1, 2))
		require.NoError(t, err)

		err = setup.ConfirmLoop(sProto, routing.LoopData{
			Loop: routing.AddrLoop{
				Remote: routing.Addr{
					PubKey: pk2,
					Port:   1,
				},
				Local: routing.Addr{
					Port: 2,
				},
			},
			RouteID: routeID,
		})
		require.NoError(t, err)

		rule, err := rt.Rule(appRouteID)
		require.NoError(t, err)
		assert.Equal(t, routeID, rule.RouteID())
		_, err = r.pm.Get(2)
		require.NoError(t, err)
		loop, err := r.pm.GetLoop(2, routing.Addr{PubKey: pk2, Port: 1})
		require.NoError(t, err)
		require.NotNil(t, loop)
		assert.Equal(t, trID, loop.trID)
		assert.Equal(t, routing.RouteID(2), loop.routeID)

		var addrs [2]routing.Addr
		require.NoError(t, json.Unmarshal(<-dataCh, &addrs))
		require.NoError(t, err)
		assert.Equal(t, pk1, addrs[0].PubKey)
		assert.Equal(t, routing.Port(2), addrs[0].Port)
		assert.Equal(t, pk2, addrs[1].PubKey)
		assert.Equal(t, routing.Port(1), addrs[1].Port)
	})

	t.Run("confirm loop - initiator", func(t *testing.T) {
		if skiptests&4 == 4 {
			t.Skip()
		}

		time.Sleep(100 * time.Millisecond)

		require.NoError(t, r.pm.SetLoop(4, routing.Addr{PubKey: pk2, Port: 3}, &loop{}))

		appRouteID, err := setup.AddRule(sProto, routing.AppRule(time.Now().Add(time.Hour), 0, pk2, 3, 4))
		require.NoError(t, err)

		err = setup.ConfirmLoop(sProto, routing.LoopData{
			Loop: routing.AddrLoop{
				Remote: routing.Addr{
					PubKey: pk2,
					Port:   3,
				},
				Local: routing.Addr{
					Port: 4,
				},
			},
			RouteID: routeID,
		})
		require.NoError(t, err)

		rule, err := rt.Rule(appRouteID)
		require.NoError(t, err)
		assert.Equal(t, routeID, rule.RouteID())
		l, err := r.pm.GetLoop(2, routing.Addr{PubKey: pk2, Port: 1})
		require.NoError(t, err)
		require.NotNil(t, l)
		assert.Equal(t, trID, l.trID)
		assert.Equal(t, routing.RouteID(2), l.routeID)

		var addrs [2]routing.Addr
		require.NoError(t, json.Unmarshal(<-dataCh, &addrs))
		require.NoError(t, err)
		assert.Equal(t, pk1, addrs[0].PubKey)
		assert.Equal(t, routing.Port(4), addrs[0].Port)
		assert.Equal(t, pk2, addrs[1].PubKey)
		assert.Equal(t, routing.Port(3), addrs[1].Port)
	})

	t.Run("loop closed", func(t *testing.T) {
		if skiptests&8 == 8 {
			t.Skip("skipping loop closed")
		}

		rule, err := rt.Rule(3)
		require.NoError(t, err)
		require.NotNil(t, rule)
		assert.Equal(t, routing.RuleApp, rule.Type())

		require.NoError(t, setup.LoopClosed(sProto, routing.LoopData{
			Loop: routing.AddrLoop{
				Remote: routing.Addr{
					PubKey: pk2,
					Port:   3,
				},
				Local: routing.Addr{
					Port: 4,
				},
			},
		}))
		time.Sleep(100 * time.Millisecond)

		_, err = r.pm.GetLoop(4, routing.Addr{PubKey: pk2, Port: 3})
		require.Error(t, err)
		_, err = r.pm.Get(4)
		require.NoError(t, err)

		rule, err = rt.Rule(3)
		require.NoError(t, err)
		require.Nil(t, rule)
	})

	t.Run("delete rule", func(t *testing.T) {
		if skiptests&16 == 16 {
			t.Skip("skipping delete rule")
		}

		require.NoError(t, setup.DeleteRule(sProto, routeID))

		rule, err := rt.Rule(routeID)
		require.NoError(t, err)
		assert.Nil(t, rule)
	})

	// th.Timeout = time.Second * 5
	require.NoError(t, th.NoErrorWithinTimeoutN(
		protoServeErr1Ch,
		protoServeErr2Ch,
		serveAppErr1Ch,
		serveAppErr2Ch))
}

func TestRouterSetupLoop(t *testing.T) {
	// Environment
	client := transport.NewDiscoveryMock()
	logStore := transport.InMemoryTransportLogStore()

	pk1, sk1 := cipher.GenerateKeyPair()
	pk2, sk2 := cipher.GenerateKeyPair()

	f1, f2 := transport.NewMockFactoryPair(pk1, pk2)
	f1.SetType(dmsg.Type)
	f2.SetType(dmsg.Type)

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

	conf := &Config{
		Logger:           logging.MustGetLogger("routesetup"),
		PubKey:           pk1,
		SecKey:           sk1,
		TransportManager: m1,
		RoutingTable:     routing.InMemoryRoutingTable(),
		RouteFinder:      routeFinder.NewMock(),
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

		if p != setup.PacketCreateLoop {
			errCh <- errors.New("unknown command")
			return
		}

		var ld routing.LoopDescriptor
		if err := json.Unmarshal(data, &ld); err != nil {
			errCh <- err
			return
		}

		if ld.Loop.Local.Port != 10 || ld.Loop.Remote.Port != 6 {
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
	appProto := app.NewProtocol(rw)
	protoServeErrCh := make(chan error, 1)
	go func() {
		protoServeErrCh <- appProto.Serve(nil)
	}()

	addr := routing.Addr{}
	require.NoError(t, appProto.Send(app.FrameCreateLoop, routing.Addr{PubKey: pk2, Port: 6}, &addr))

	require.NoError(t, <-errCh)
	ll, err := r.pm.GetLoop(10, routing.Addr{PubKey: pk2, Port: 6})
	require.NoError(t, err)
	require.NotNil(t, ll)

	assert.Equal(t, pk1, addr.PubKey)
	assert.Equal(t, routing.Port(10), addr.Port)

	require.NoError(t, th.NoErrorWithinTimeoutN(serveErrCh, serveAppErrCh, protoServeErrCh))
}

func TestRouterSetupLoopLocal(t *testing.T) {
	pk, sk := cipher.GenerateKeyPair()
	conf := &Config{
		Logger: logging.MustGetLogger("routesetup"),
		PubKey: pk,
		SecKey: sk,
	}
	r := New(conf)

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

	addr := routing.Addr{}
	require.NoError(t, proto.Send(app.FrameCreateLoop, routing.Addr{PubKey: pk, Port: 5}, &addr))

	ll, err := r.pm.GetLoop(10, routing.Addr{PubKey: pk, Port: 5})
	require.NoError(t, err)
	require.NotNil(t, ll)

	assert.Equal(t, pk, addr.PubKey)
	assert.Equal(t, routing.Port(10), addr.Port)

	require.NoError(t, th.NoErrorWithinTimeoutN(serveAppErrCh, protoServeErrCh))

}
