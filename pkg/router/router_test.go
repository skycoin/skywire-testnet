package router

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/SkycoinProject/dmsg"
	"github.com/SkycoinProject/dmsg/cipher"
	"github.com/SkycoinProject/skycoin/src/util/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SkycoinProject/skywire-mainnet/internal/testhelpers"
	"github.com/SkycoinProject/skywire-mainnet/pkg/app"
	routeFinder "github.com/SkycoinProject/skywire-mainnet/pkg/route-finder/client"
	"github.com/SkycoinProject/skywire-mainnet/pkg/routing"
	"github.com/SkycoinProject/skywire-mainnet/pkg/setup"
	"github.com/SkycoinProject/skywire-mainnet/pkg/transport"
)

func TestMain(m *testing.M) {
	loggingLevel, ok := os.LookupEnv("TEST_LOGGING_LEVEL")
	if ok {
		lvl, err := logging.LevelFromString(loggingLevel)
		if err != nil {
			log.Fatal(err)
		}
		logging.SetLevel(lvl)
	} else {
		logging.SetLevel(logrus.TraceLevel)
	}
	os.Exit(m.Run())
}

// TODO(evanlinjin): Fix test.
//func TestRouterForwarding(t *testing.T) {
//	client := transport.NewDiscoveryMock()
//	logStore := transport.InMemoryTransportLogStore()
//
//	pk1, sk1 := cipher.GenerateKeyPair()
//	pk2, sk2 := cipher.GenerateKeyPair()
//	pk3, sk3 := cipher.GenerateKeyPair()
//
//	c1 := &transport.ManagerConfig{PubKey: pk1, SecKey: sk1, DiscoveryClient: client, LogStore: logStore}
//	c2 := &transport.ManagerConfig{PubKey: pk2, SecKey: sk2, DiscoveryClient: client, LogStore: logStore}
//	c3 := &transport.ManagerConfig{PubKey: pk3, SecKey: sk3, DiscoveryClient: client, LogStore: logStore}
//
//	f1, f2 := transport.NewMockFactoryPair(pk1, pk2)
//	f3, f4 := transport.NewMockFactoryPair(pk2, pk3)
//	f3.SetType("mock2")
//	f4.SetType("mock2")
//
//	m1, err := transport.NewManager(c1, nil, f1)
//	require.NoError(t, err)
//	go func() { _ = m1.Serve(context.TODO()) }() //nolint:errcheck
//
//	m2, err := transport.NewManager(c2, nil, f2, f3)
//	require.NoError(t, err)
//	go func() { _ = m2.Serve(context.TODO()) }() //nolint:errcheck
//
//	m3, err := transport.NewManager(c3, nil, f4)
//	require.NoError(t, err)
//	go func() { _ = m3.Serve(context.TODO()) }() //nolint:errcheck
//
//	rt := routing.InMemoryRoutingTable()
//	conf := &Config{
//		Logger:           logging.MustGetLogger("router"),
//		PubKey:           pk2,
//		SecKey:           sk2,
//		TransportManager: m2,
//		RoutingTable:     rt,
//	}
//	r := New(conf)
//	errCh := make(chan error)
//	go func() {
//		errCh <- r.Serve(context.TODO())
//	}()
//
//	tr1, err := m1.SaveTransport(context.TODO(), pk2, "mock")
//	require.NoError(t, err)
//
//	tr3, err := m3.SaveTransport(context.TODO(), pk2, "mock2")
//	require.NoError(t, err)
//
//	rule := routing.ForwardRule(time.Now().Add(time.Hour), 4, tr3.Entry.ID)
//	routeID, err := rt.AddRule(rule)
//	require.NoError(t, err)
//
//	time.Sleep(100 * time.Millisecond)
//
//	require.NoError(t, tr1.WritePacket(context.TODO(), routeID, []byte("foo")))
//
//	packet, err := m3.ReadPacket()
//	require.NoError(t, err)
//	assert.Equal(t, uint16(3), packet.Size())
//	assert.Equal(t, routing.RouteID(4), packet.RouteID())
//	assert.Equal(t, []byte("foo"), packet.Payload())
//
//	require.NoError(t, m1.Close())
//	require.NoError(t, m3.Close())
//
//	time.Sleep(100 * time.Millisecond)
//
//	require.NoError(t, r.Close())
//	require.NoError(t, <-errCh)
//}

func TestRouterAppInit(t *testing.T) {
	client := transport.NewDiscoveryMock()
	logStore := transport.InMemoryTransportLogStore()

	pk1, sk1 := cipher.GenerateKeyPair()
	c1 := &transport.ManagerConfig{PubKey: pk1, SecKey: sk1, DiscoveryClient: client, LogStore: logStore}

	m1, err := transport.NewManager(c1, nil)
	require.NoError(t, err)
	go func() { _ = m1.Serve(context.TODO()) }() //nolint:errcheck

	conf := &Config{
		Logger:           logging.MustGetLogger("routesetup"),
		PubKey:           pk1,
		SecKey:           sk1,
		TransportManager: m1,
	}
	r := New(conf)
	rw, rwIn := net.Pipe()
	errCh := make(chan error, 1)
	go func() {
		errCh <- r.ServeApp(rwIn, 10, &app.Config{AppName: "foo", AppVersion: "0.0.1"})
		close(errCh)
	}()

	proto := app.NewProtocol(rw)
	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- proto.Serve(nil)
	}()

	require.NoError(t, proto.Send(app.FrameInit, &app.Config{AppName: "foo", AppVersion: "0.0.1", ProtocolVersion: "0.0.1"}, nil))
	require.Error(t, proto.Send(app.FrameInit, &app.Config{AppName: "foo1", AppVersion: "0.0.1", ProtocolVersion: "0.0.1"}, nil))

	require.NoError(t, proto.Close())
	require.NoError(t, testhelpers.NoErrorWithinTimeout(serveErrCh))
	require.NoError(t, r.Close())
	require.NoError(t, <-errCh)
}

// TODO(evanlinjin): Figure out what this is testing and fix it.
//func TestRouterApp(t *testing.T) {
//	client := transport.NewDiscoveryMock()
//	logStore := transport.InMemoryTransportLogStore()
//
//	pk1, sk1 := cipher.GenerateKeyPair()
//	pk2, sk2 := cipher.GenerateKeyPair()
//
//	c1 := &transport.ManagerConfig{PubKey: pk1, SecKey: sk1, DiscoveryClient: client, LogStore: logStore}
//	c2 := &transport.ManagerConfig{PubKey: pk2, SecKey: sk2, DiscoveryClient: client, LogStore: logStore}
//
//	f1, f2 := transport.NewMockFactoryPair(pk1, pk2)
//
//	m1, err := transport.NewManager(c1, nil, f1)
//	require.NoError(t, err)
//	//go func() {_ = m1.Serve(context.TODO())}()
//
//	m2, err := transport.NewManager(c2, nil, f2)
//	require.NoError(t, err)
//	go func() {_ = m2.Serve(context.TODO())}()
//
//	rt := routing.InMemoryRoutingTable()
//	conf := &Config{
//		Logger:           logging.MustGetLogger("routesetup"),
//		PubKey:           pk1,
//		SecKey:           sk1,
//		TransportManager: m1,
//		RoutingTable:     rt,
//	}
//	r := New(conf)
//	errCh := make(chan error, 1)
//	go func() {
//		errCh <- r.Serve(context.TODO())
//		close(errCh)
//	}()
//
//	rw, rwIn := net.Pipe()
//	serveAppErrCh := make(chan error, 1)
//	go func() {
//		serveAppErrCh <- r.ServeApp(rwIn, 6, &app.Config{})
//		close(serveAppErrCh)
//	}()
//
//	proto := app.NewProtocol(rw)
//	dataCh := make(chan []byte)
//	protoServeErrCh := make(chan error, 1)
//	go func() {
//		f := func(_ app.Frame, p []byte) (interface{}, error) {
//			go func() { dataCh <- p }()
//			return nil, nil
//		}
//		protoServeErrCh <- proto.Serve(f)
//	}()
//
//	time.Sleep(100 * time.Millisecond)
//
//	tr, err := m1.SaveTransport(context.TODO(), pk2, "mock")
//	require.NoError(t, err)
//
//	rule := routing.AppRule(time.Now().Add(time.Hour), 4, pk2, 5, 6)
//	routeID, err := rt.AddRule(rule)
//	require.NoError(t, err)
//
//	raddr := routing.Addr{PubKey: pk2, Port: 5}
//	require.NoError(t, r.pm.SetLoop(6, raddr, &loop{tr.Entry.ID, 4}))
//
//	tr2 := m2.Transport(tr.Entry.ID)
//	sendErrCh := make(chan error, 1)
//	go func() {
//		sendErrCh <- proto.Send(app.FrameSend, &app.Packet{Loop: routing.Loop{Local: routing.Addr{Port: 6}, Remote: raddr}, Payload: []byte("bar")}, nil)
//		close(sendErrCh)
//	}()
//
//	packet, err := m2.ReadPacket()
//	require.NoError(t, err)
//	assert.Equal(t, uint16(3), packet.Size())
//	assert.Equal(t, routing.RouteID(4), packet.RouteID())
//	assert.Equal(t, []byte("bar"), packet.Payload())
//
//	require.NoError(t, tr2.WritePacket(context.TODO(), routeID, []byte("foo")))
//
//	time.Sleep(100 * time.Millisecond)
//
//	var aPacket app.Packet
//	require.NoError(t, json.Unmarshal(<-dataCh, &aPacket))
//	assert.Equal(t, pk2, aPacket.Loop.Remote.PubKey)
//	assert.Equal(t, routing.Port(5), aPacket.Loop.Remote.Port)
//	assert.Equal(t, routing.Port(6), aPacket.Loop.Local.Port)
//	assert.Equal(t, []byte("foo"), aPacket.Payload)
//
//	require.NoError(t, r.Close())
//	require.NoError(t, <-errCh)
//
//	require.NoError(t, m2.Close())
//	require.NoError(t, testhelpers.NoErrorWithinTimeout(protoServeErrCh))
//	require.NoError(t, testhelpers.NoErrorWithinTimeout(sendErrCh))
//	require.NoError(t, testhelpers.NoErrorWithinTimeout(serveAppErrCh))
//}

func TestRouterLocalApp(t *testing.T) {
	client := transport.NewDiscoveryMock()
	logStore := transport.InMemoryTransportLogStore()

	pk, sk := cipher.GenerateKeyPair()
	m, err := transport.NewManager(&transport.ManagerConfig{PubKey: pk, SecKey: sk, DiscoveryClient: client, LogStore: logStore}, nil)
	require.NoError(t, err)

	conf := &Config{
		Logger:           logging.MustGetLogger("routesetup"),
		PubKey:           pk,
		SecKey:           sk,
		TransportManager: m,
		RoutingTable:     routing.InMemoryRoutingTable(),
	}
	r := New(conf)
	errCh := make(chan error)
	go func() {
		errCh <- r.Serve(context.TODO())
	}()

	rw1, rw1In := net.Pipe()
	serveAppErr1Ch := make(chan error, 1)
	go func() {
		serveAppErr1Ch <- r.ServeApp(rw1In, 5, &app.Config{})
	}()
	proto1 := app.NewProtocol(rw1)
	protoServeErr1Ch := make(chan error, 1)
	go func() {
		protoServeErr1Ch <- proto1.Serve(nil)
	}()

	rw2, rw2In := net.Pipe()
	serveAppErr2Ch := make(chan error, 1)
	go func() {
		serveAppErr2Ch <- r.ServeApp(rw2In, 6, &app.Config{})
	}()
	proto2 := app.NewProtocol(rw2)
	dataCh := make(chan []byte)
	protoServeErr2Ch := make(chan error, 1)
	go func() {
		f := func(_ app.Frame, p []byte) (interface{}, error) {
			go func() { dataCh <- p }()
			return nil, nil
		}
		protoServeErr2Ch <- proto2.Serve(f)
	}()

	sendErrCh := make(chan error, 1)
	go func() {
		packet := &app.Packet{
			Loop: routing.Loop{Local: routing.Addr{Port: 5}, Remote: routing.Addr{PubKey: pk, Port: 6}}, Payload: []byte("foo"),
		}
		sendErrCh <- proto1.Send(app.FrameSend, packet, nil)
	}()

	time.Sleep(100 * time.Millisecond)

	packet := &app.Packet{}
	require.NoError(t, json.Unmarshal(<-dataCh, packet))
	require.NoError(t, err)
	assert.Equal(t, pk, packet.Loop.Remote.PubKey)
	assert.Equal(t, routing.Port(5), packet.Loop.Remote.Port)
	assert.Equal(t, routing.Port(6), packet.Loop.Local.Port)
	assert.Equal(t, []byte("foo"), packet.Payload)

	require.NoError(t, r.Close())
	require.NoError(t, <-errCh)

	require.NoError(t, testhelpers.NoErrorWithinTimeout(protoServeErr1Ch))
	require.NoError(t, testhelpers.NoErrorWithinTimeout(protoServeErr2Ch))
	require.NoError(t, testhelpers.NoErrorWithinTimeout(sendErrCh))
	require.NoError(t, testhelpers.NoErrorWithinTimeout(serveAppErr1Ch))
	require.NoError(t, testhelpers.NoErrorWithinTimeout(serveAppErr2Ch))
}

func TestRouterSetup(t *testing.T) {
	client := transport.NewDiscoveryMock()
	logStore := transport.InMemoryTransportLogStore()

	pk1, sk1 := cipher.GenerateKeyPair()
	pk2, sk2 := cipher.GenerateKeyPair()

	c1 := &transport.ManagerConfig{PubKey: pk1, SecKey: sk1, DiscoveryClient: client, LogStore: logStore}
	c2 := &transport.ManagerConfig{PubKey: pk2, SecKey: sk2, DiscoveryClient: client, LogStore: logStore}

	f1, f2 := transport.NewMockFactoryPair(pk1, pk2)
	m1, err := transport.NewManager(c1, []cipher.PubKey{pk2}, f1)
	require.NoError(t, err)

	m2, err := transport.NewManager(c2, nil, f2)
	require.NoError(t, err)
	go func() { _ = m2.Serve(context.TODO()) }() //nolint:errcheck

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

	tr, err := m2.DialSetupConn(context.TODO(), pk1, "mock")
	require.NoError(t, err)
	trID := transport.MakeTransportID(tr.LocalPK(), tr.RemotePK(), tr.Type())
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

	var routeID routing.RouteID
	t.Run("add rule", func(t *testing.T) {
		routeID, err = setup.AddRule(context.TODO(), sProto, routing.ForwardRule(time.Now().Add(time.Hour), 2, trID))
		require.NoError(t, err)

		rule, err := rt.Rule(routeID)
		require.NoError(t, err)
		assert.Equal(t, routing.RouteID(2), rule.RouteID())
		assert.Equal(t, trID, rule.TransportID())
	})

	t.Run("confirm loop - responder", func(t *testing.T) {
		appRouteID, err := setup.AddRule(context.TODO(), sProto, routing.AppRule(time.Now().Add(time.Hour), 0, pk2, 1, 2))
		require.NoError(t, err)

		err = setup.ConfirmLoop(context.TODO(), sProto, routing.LoopData{
			Loop: routing.Loop{
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
		time.Sleep(100 * time.Millisecond)

		require.NoError(t, r.pm.SetLoop(4, routing.Addr{PubKey: pk2, Port: 3}, &loop{}))

		appRouteID, err := setup.AddRule(context.TODO(), sProto, routing.AppRule(time.Now().Add(time.Hour), 0, pk2, 3, 4))
		require.NoError(t, err)

		err = setup.ConfirmLoop(context.TODO(), sProto, routing.LoopData{
			Loop: routing.Loop{
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
		rule, err := rt.Rule(3)
		require.NoError(t, err)
		require.NotNil(t, rule)
		assert.Equal(t, routing.RuleApp, rule.Type())

		require.NoError(t, setup.LoopClosed(context.TODO(), sProto, routing.LoopData{
			Loop: routing.Loop{
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
		require.NoError(t, setup.DeleteRule(context.TODO(), sProto, routeID))

		rule, err := rt.Rule(routeID)
		require.NoError(t, err)
		assert.Nil(t, rule)
	})

	require.NoError(t, testhelpers.NoErrorWithinTimeout(protoServeErr1Ch))
	require.NoError(t, testhelpers.NoErrorWithinTimeout(protoServeErr2Ch))
	require.NoError(t, testhelpers.NoErrorWithinTimeout(serveAppErr1Ch))
	require.NoError(t, testhelpers.NoErrorWithinTimeout(serveAppErr2Ch))
}

func TestRouterSetupLoop(t *testing.T) {
	client := transport.NewDiscoveryMock()
	logStore := transport.InMemoryTransportLogStore()

	pk1, sk1 := cipher.GenerateKeyPair()
	pk2, sk2 := cipher.GenerateKeyPair()

	f1, f2 := transport.NewMockFactoryPair(pk1, pk2)
	f1.SetType(dmsg.Type)
	f2.SetType(dmsg.Type)

	m1, err := transport.NewManager(
		&transport.ManagerConfig{PubKey: pk1, SecKey: sk1, DiscoveryClient: client, LogStore: logStore},
		[]cipher.PubKey{pk2},
		f1)
	require.NoError(t, err)

	m2, err := transport.NewManager(
		&transport.ManagerConfig{PubKey: pk2, SecKey: sk2, DiscoveryClient: client, LogStore: logStore},
		[]cipher.PubKey{pk1},
		f2)
	require.NoError(t, err)

	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- m2.Serve(context.TODO())
		close(serveErrCh)
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
		tr, err := m2.AcceptSetupConn()
		if err != nil {
			errCh <- err
			return
		}

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

	require.NoError(t, testhelpers.NoErrorWithinTimeout(serveErrCh))
	require.NoError(t, testhelpers.NoErrorWithinTimeout(serveAppErrCh))
	require.NoError(t, testhelpers.NoErrorWithinTimeout(protoServeErrCh))
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

	require.NoError(t, testhelpers.NoErrorWithinTimeout(serveAppErrCh))
	require.NoError(t, testhelpers.NoErrorWithinTimeout(protoServeErrCh))
}

func TestRouterCloseLoop(t *testing.T) {
	client := transport.NewDiscoveryMock()
	logStore := transport.InMemoryTransportLogStore()

	pk1, sk1 := cipher.GenerateKeyPair()
	pk2, sk2 := cipher.GenerateKeyPair()
	pk3, _ := cipher.GenerateKeyPair()

	f1, f2 := transport.NewMockFactoryPair(pk1, pk2)
	f1.SetType(dmsg.Type)

	m1, err := transport.NewManager(
		&transport.ManagerConfig{PubKey: pk1, SecKey: sk1, DiscoveryClient: client, LogStore: logStore},
		[]cipher.PubKey{pk2},
		f1)
	require.NoError(t, err)

	m2, err := transport.NewManager(
		&transport.ManagerConfig{PubKey: pk2, SecKey: sk2, DiscoveryClient: client, LogStore: logStore},
		[]cipher.PubKey{pk1},
		f2)
	require.NoError(t, err)

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
		tr, err := m2.AcceptSetupConn()
		if err != nil {
			errCh <- err
			return
		}

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

	require.NoError(t, proto.Send(app.FrameClose, routing.Loop{Local: routing.Addr{Port: 5}, Remote: raddr}, nil))

	time.Sleep(100 * time.Millisecond)

	require.NoError(t, <-errCh)
	_, err = r.pm.GetLoop(5, routing.Addr{PubKey: pk3, Port: 6})
	require.Error(t, err)
	_, err = r.pm.Get(5)
	require.NoError(t, err)

	rule, err = rt.Rule(routeID)
	require.NoError(t, err)
	require.Nil(t, rule)

	require.NoError(t, testhelpers.NoErrorWithinTimeout(serveErrCh))
	require.NoError(t, testhelpers.NoErrorWithinTimeout(serveAppErrCh))
	require.NoError(t, testhelpers.NoErrorWithinTimeout(protoServeErrCh))
}

func TestRouterCloseLoopOnAppClose(t *testing.T) {
	client := transport.NewDiscoveryMock()
	logStore := transport.InMemoryTransportLogStore()

	pk1, sk1 := cipher.GenerateKeyPair()
	pk2, sk2 := cipher.GenerateKeyPair()
	pk3, _ := cipher.GenerateKeyPair()

	f1, f2 := transport.NewMockFactoryPair(pk1, pk2)
	f1.SetType(dmsg.Type)

	m1, err := transport.NewManager(
		&transport.ManagerConfig{PubKey: pk1, SecKey: sk1, DiscoveryClient: client, LogStore: logStore},
		[]cipher.PubKey{pk2},
		f1)
	require.NoError(t, err)

	m2, err := transport.NewManager(
		&transport.ManagerConfig{PubKey: pk2, SecKey: sk2, DiscoveryClient: client, LogStore: logStore},
		[]cipher.PubKey{pk1},
		f2)
	require.NoError(t, err)

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
		tr, err := m2.AcceptSetupConn()
		if err != nil {
			errCh <- err
			return
		}

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

	require.NoError(t, testhelpers.NoErrorWithinTimeout(serveErrCh))
	require.NoError(t, testhelpers.NoErrorWithinTimeout(serveAppErrCh))
	require.NoError(t, testhelpers.NoErrorWithinTimeout(protoServeErrCh))
}

func TestRouterRouteExpiration(t *testing.T) {
	client := transport.NewDiscoveryMock()
	logStore := transport.InMemoryTransportLogStore()

	pk, sk := cipher.GenerateKeyPair()
	m, err := transport.NewManager(&transport.ManagerConfig{PubKey: pk, SecKey: sk, DiscoveryClient: client, LogStore: logStore}, nil)
	require.NoError(t, err)
	go func() { _ = m.Serve(context.TODO()) }() //nolint:errcheck

	rt := routing.InMemoryRoutingTable()
	_, err = rt.AddRule(routing.AppRule(time.Now().Add(-time.Hour), 4, pk, 6, 5))
	require.NoError(t, err)
	assert.Equal(t, 1, rt.Count())

	conf := &Config{
		Logger:           logging.MustGetLogger("routesetup"),
		PubKey:           pk,
		SecKey:           sk,
		TransportManager: m,
		RoutingTable:     rt,
	}
	r := New(conf)
	r.expiryTicker = time.NewTicker(100 * time.Millisecond)
	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- r.Serve(context.TODO())
	}()

	time.Sleep(110 * time.Millisecond)

	assert.Equal(t, 0, rt.Count())
	require.NoError(t, r.Close())

	require.NoError(t, testhelpers.NoErrorWithinTimeout(serveErrCh))
}
