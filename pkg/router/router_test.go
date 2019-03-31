package router

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"testing"
	"time"

	"github.com/skycoin/skycoin/src/util/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/pkg/cipher"

	"github.com/skycoin/skywire/internal/noise"
	"github.com/skycoin/skywire/pkg/app"
	routeFinder "github.com/skycoin/skywire/pkg/route-finder/client"
	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/setup"
	"github.com/skycoin/skywire/pkg/transport"
)

func TestMain(m *testing.M) {
	lvl, _ := logging.LevelFromString("error") // nolint: errcheck
	logging.SetLevel(lvl)
	os.Exit(m.Run())
}

func TestRouterForwarding(t *testing.T) {
	client := transport.NewDiscoveryMock()
	logStore := transport.InMemoryTransportLogStore()

	pk1, sk1 := cipher.GenerateKeyPair()
	pk2, sk2 := cipher.GenerateKeyPair()
	pk3, sk3 := cipher.GenerateKeyPair()

	c1 := &transport.ManagerConfig{PubKey: pk1, SecKey: sk1, DiscoveryClient: client, LogStore: logStore}
	c2 := &transport.ManagerConfig{PubKey: pk2, SecKey: sk2, DiscoveryClient: client, LogStore: logStore}
	c3 := &transport.ManagerConfig{PubKey: pk3, SecKey: sk3, DiscoveryClient: client, LogStore: logStore}

	f1, f2 := transport.NewMockFactory(pk1, pk2)
	f3, f4 := transport.NewMockFactory(pk2, pk3)
	f3.SetType("mock2")
	f4.SetType("mock2")

	m1, err := transport.NewManager(c1, f1)
	require.NoError(t, err)

	m2, err := transport.NewManager(c2, f2, f3)
	require.NoError(t, err)

	m3, err := transport.NewManager(c3, f4)
	require.NoError(t, err)

	rt := routing.InMemoryRoutingTable()
	conf := &Config{
		Logger:           logging.MustGetLogger("routesetup"),
		PubKey:           pk2,
		SecKey:           sk2,
		TransportManager: m2,
		RoutingTable:     rt,
	}
	r := New(conf)
	errCh := make(chan error)
	go func() {
		errCh <- r.Serve(context.TODO())
	}()

	tr1, err := m1.CreateTransport(context.TODO(), pk2, "mock", true)
	require.NoError(t, err)

	tr3, err := m3.CreateTransport(context.TODO(), pk2, "mock2", true)
	require.NoError(t, err)

	rule := routing.ForwardRule(time.Now().Add(time.Hour), 4, tr3.ID)
	routeID, err := rt.AddRule(rule)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	_, err = tr1.Write(routing.MakePacket(routeID, []byte("foo")))
	require.NoError(t, err)

	packet := make(routing.Packet, 9)
	_, err = tr3.Read(packet)
	require.NoError(t, err)
	assert.Equal(t, uint16(3), packet.Size())
	assert.Equal(t, routing.RouteID(4), packet.RouteID())
	assert.Equal(t, []byte("foo"), packet.Payload())

	require.NoError(t, m1.Close())
	require.NoError(t, m3.Close())

	time.Sleep(100 * time.Millisecond)

	require.NoError(t, r.Close())
	require.NoError(t, <-errCh)
}

func TestRouterAppInit(t *testing.T) {
	client := transport.NewDiscoveryMock()
	logStore := transport.InMemoryTransportLogStore()

	pk1, sk1 := cipher.GenerateKeyPair()
	c1 := &transport.ManagerConfig{PubKey: pk1, SecKey: sk1, DiscoveryClient: client, LogStore: logStore}

	m1, err := transport.NewManager(c1)
	require.NoError(t, err)

	conf := &Config{
		Logger:           logging.MustGetLogger("routesetup"),
		PubKey:           pk1,
		SecKey:           sk1,
		TransportManager: m1,
	}
	r := New(conf)
	rw, rwIn := net.Pipe()
	errCh := make(chan error)
	go func() {
		errCh <- r.ServeApp(rwIn, 10, &app.Config{AppName: "foo", AppVersion: "0.0.1"})
	}()

	proto := app.NewProtocol(rw)
	go proto.Serve(nil) // nolint: errcheck

	require.NoError(t, proto.Send(app.FrameInit, &app.Config{AppName: "foo", AppVersion: "0.0.1", ProtocolVersion: "0.0.1"}, nil))
	require.Error(t, proto.Send(app.FrameInit, &app.Config{AppName: "foo1", AppVersion: "0.0.1", ProtocolVersion: "0.0.1"}, nil))

	require.NoError(t, proto.Close())
	require.NoError(t, r.Close())
	require.NoError(t, <-errCh)
}

func TestRouterApp(t *testing.T) {
	client := transport.NewDiscoveryMock()
	logStore := transport.InMemoryTransportLogStore()

	pk1, sk1 := cipher.GenerateKeyPair()
	pk2, sk2 := cipher.GenerateKeyPair()

	c1 := &transport.ManagerConfig{PubKey: pk1, SecKey: sk1, DiscoveryClient: client, LogStore: logStore}
	c2 := &transport.ManagerConfig{PubKey: pk2, SecKey: sk2, DiscoveryClient: client, LogStore: logStore}

	f1, f2 := transport.NewMockFactory(pk1, pk2)
	m1, err := transport.NewManager(c1, f1)
	require.NoError(t, err)

	m2, err := transport.NewManager(c2, f2)
	require.NoError(t, err)

	go m2.Serve(context.TODO()) // nolint

	rt := routing.InMemoryRoutingTable()
	conf := &Config{
		Logger:           logging.MustGetLogger("routesetup"),
		PubKey:           pk1,
		SecKey:           sk1,
		TransportManager: m1,
		RoutingTable:     rt,
	}
	r := New(conf)
	errCh := make(chan error)
	go func() {
		errCh <- r.Serve(context.TODO())
	}()

	rw, rwIn := net.Pipe()
	go r.ServeApp(rwIn, 6, &app.Config{}) // nolint: errcheck
	proto := app.NewProtocol(rw)
	dataCh := make(chan []byte)
	go proto.Serve(func(_ app.Frame, p []byte) (interface{}, error) { // nolint: errcheck,unparam
		go func() { dataCh <- p }()
		return nil, nil
	})

	time.Sleep(100 * time.Millisecond)

	tr, err := m1.CreateTransport(context.TODO(), pk2, "mock", true)
	require.NoError(t, err)

	rule := routing.AppRule(time.Now().Add(time.Hour), 4, pk2, 5, 6)
	routeID, err := rt.AddRule(rule)
	require.NoError(t, err)

	ni1, ni2 := noiseInstances(t, pk1, pk2, sk1, sk2)
	raddr := &app.Addr{PubKey: pk2, Port: 5}
	require.NoError(t, r.pm.SetLoop(6, raddr, &loop{tr.ID, 4, ni1}))

	tr2 := m2.Transport(tr.ID)
	go proto.Send(app.FrameSend, &app.Packet{Addr: &app.LoopAddr{Port: 6, Remote: *raddr}, Payload: []byte("bar")}, nil) // nolint: errcheck

	packet := make(routing.Packet, 25)
	_, err = tr2.Read(packet)
	require.NoError(t, err)
	assert.Equal(t, uint16(19), packet.Size())
	assert.Equal(t, routing.RouteID(4), packet.RouteID())
	decrypted, err := ni2.Decrypt(packet.Payload())
	require.NoError(t, err)
	assert.Equal(t, []byte("bar"), decrypted)

	_, err = tr2.Write(routing.MakePacket(routeID, ni2.Encrypt([]byte("foo"))))
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	aPacket := &app.Packet{}
	require.NoError(t, json.Unmarshal(<-dataCh, aPacket))
	assert.Equal(t, pk2, aPacket.Addr.Remote.PubKey)
	assert.Equal(t, uint16(5), aPacket.Addr.Remote.Port)
	assert.Equal(t, uint16(6), aPacket.Addr.Port)
	assert.Equal(t, []byte("foo"), aPacket.Payload)

	require.NoError(t, r.Close())
	require.NoError(t, <-errCh)
}

func TestRouterLocalApp(t *testing.T) {
	client := transport.NewDiscoveryMock()
	logStore := transport.InMemoryTransportLogStore()

	pk, sk := cipher.GenerateKeyPair()
	m, err := transport.NewManager(&transport.ManagerConfig{PubKey: pk, SecKey: sk, DiscoveryClient: client, LogStore: logStore})
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
	go r.ServeApp(rw1In, 5, &app.Config{}) // nolint: errcheck
	proto1 := app.NewProtocol(rw1)
	go proto1.Serve(nil) // nolint: errcheck

	rw2, rw2In := net.Pipe()
	go r.ServeApp(rw2In, 6, &app.Config{}) // nolint: errcheck
	proto2 := app.NewProtocol(rw2)
	dataCh := make(chan []byte)
	go proto2.Serve(func(_ app.Frame, p []byte) (interface{}, error) { // nolint: errcheck,unparam
		go func() { dataCh <- p }()
		return nil, nil
	})

	go proto1.Send(app.FrameSend, &app.Packet{Addr: &app.LoopAddr{Port: 5, Remote: app.Addr{PubKey: pk, Port: 6}}, Payload: []byte("foo")}, nil) // nolint: errcheck

	time.Sleep(100 * time.Millisecond)

	packet := &app.Packet{}
	require.NoError(t, json.Unmarshal(<-dataCh, packet))
	require.NoError(t, err)
	assert.Equal(t, pk, packet.Addr.Remote.PubKey)
	assert.Equal(t, uint16(5), packet.Addr.Remote.Port)
	assert.Equal(t, uint16(6), packet.Addr.Port)
	assert.Equal(t, []byte("foo"), packet.Payload)

	require.NoError(t, r.Close())
	require.NoError(t, <-errCh)
}

func TestRouterSetup(t *testing.T) {
	client := transport.NewDiscoveryMock()
	logStore := transport.InMemoryTransportLogStore()

	pk1, sk1 := cipher.GenerateKeyPair()
	pk2, sk2 := cipher.GenerateKeyPair()

	c1 := &transport.ManagerConfig{PubKey: pk1, SecKey: sk1, DiscoveryClient: client, LogStore: logStore}
	c2 := &transport.ManagerConfig{PubKey: pk2, SecKey: sk2, DiscoveryClient: client, LogStore: logStore}

	f1, f2 := transport.NewMockFactory(pk1, pk2)
	m1, err := transport.NewManager(c1, f1)
	require.NoError(t, err)

	m2, err := transport.NewManager(c2, f2)
	require.NoError(t, err)

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

	tr, err := m2.CreateTransport(context.TODO(), pk1, "mock", false)
	require.NoError(t, err)
	sProto := setup.NewProtocol(tr)

	rw1, rwIn1 := net.Pipe()
	go r.ServeApp(rwIn1, 2, &app.Config{}) // nolint: errcheck
	proto1 := app.NewProtocol(rw1)
	dataCh := make(chan []byte)
	go proto1.Serve(func(_ app.Frame, p []byte) (interface{}, error) { // nolint: errcheck,unparam
		go func() { dataCh <- p }()
		return nil, nil
	})

	rw2, rwIn2 := net.Pipe()
	go r.ServeApp(rwIn2, 4, &app.Config{}) // nolint: errcheck
	proto2 := app.NewProtocol(rw2)
	go proto2.Serve(func(_ app.Frame, p []byte) (interface{}, error) { // nolint: errcheck,unparam
		go func() { dataCh <- p }()
		return nil, nil
	})

	var routeID routing.RouteID
	t.Run("add route", func(t *testing.T) {
		routeID, err = setup.AddRule(sProto, routing.ForwardRule(time.Now().Add(time.Hour), 2, tr.ID))
		require.NoError(t, err)

		rule, err := rt.Rule(routeID)
		require.NoError(t, err)
		assert.Equal(t, routing.RouteID(2), rule.RouteID())
		assert.Equal(t, tr.ID, rule.TransportID())
	})

	t.Run("confirm loop - responder", func(t *testing.T) {
		confI := noise.Config{
			LocalSK:   sk2,
			LocalPK:   pk2,
			RemotePK:  pk1,
			Initiator: true,
		}

		ni, err := noise.KKAndSecp256k1(confI)
		require.NoError(t, err)
		msg, err := ni.HandshakeMessage()
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)

		appRouteID, err := setup.AddRule(sProto, routing.AppRule(time.Now().Add(time.Hour), 0, pk2, 1, 2))
		require.NoError(t, err)

		noiseRes, err := setup.ConfirmLoop(sProto, &setup.LoopData{RemotePK: pk2, RemotePort: 1, LocalPort: 2, RouteID: routeID, NoiseMessage: msg})
		require.NoError(t, err)

		rule, err := rt.Rule(appRouteID)
		require.NoError(t, err)
		assert.Equal(t, routeID, rule.RouteID())
		_, err = r.pm.Get(2)
		require.NoError(t, err)
		loop, err := r.pm.GetLoop(2, &app.Addr{PubKey: pk2, Port: 1})
		require.NoError(t, err)
		require.NotNil(t, loop)
		assert.Equal(t, tr.ID, loop.trID)
		assert.Equal(t, routing.RouteID(2), loop.routeID)

		addrs := [2]*app.Addr{}
		require.NoError(t, json.Unmarshal(<-dataCh, &addrs))
		require.NoError(t, err)
		assert.Equal(t, pk1, addrs[0].PubKey)
		assert.Equal(t, uint16(2), addrs[0].Port)
		assert.Equal(t, pk2, addrs[1].PubKey)
		assert.Equal(t, uint16(1), addrs[1].Port)

		require.NoError(t, ni.ProcessMessage(noiseRes))
	})

	t.Run("confirm loop - initiator", func(t *testing.T) {
		confI := noise.Config{
			LocalSK:   sk1,
			LocalPK:   pk1,
			RemotePK:  pk2,
			Initiator: true,
		}

		ni, err := noise.KKAndSecp256k1(confI)
		require.NoError(t, err)
		msg, err := ni.HandshakeMessage()
		require.NoError(t, err)

		confR := noise.Config{
			LocalSK:   sk2,
			LocalPK:   pk2,
			RemotePK:  pk1,
			Initiator: false,
		}

		nr, err := noise.KKAndSecp256k1(confR)
		require.NoError(t, err)
		require.NoError(t, nr.ProcessMessage(msg))
		noiseRes, err := nr.HandshakeMessage()
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)

		require.NoError(t, r.pm.SetLoop(4, &app.Addr{PubKey: pk2, Port: 3}, &loop{noise: ni}))

		appRouteID, err := setup.AddRule(sProto, routing.AppRule(time.Now().Add(time.Hour), 0, pk2, 3, 4))
		require.NoError(t, err)

		_, err = setup.ConfirmLoop(sProto, &setup.LoopData{RemotePK: pk2, RemotePort: 3, LocalPort: 4, RouteID: routeID, NoiseMessage: noiseRes})
		require.NoError(t, err)

		rule, err := rt.Rule(appRouteID)
		require.NoError(t, err)
		assert.Equal(t, routeID, rule.RouteID())
		l, err := r.pm.GetLoop(2, &app.Addr{PubKey: pk2, Port: 1})
		require.NoError(t, err)
		require.NotNil(t, l)
		assert.Equal(t, tr.ID, l.trID)
		assert.Equal(t, routing.RouteID(2), l.routeID)

		addrs := [2]*app.Addr{}
		require.NoError(t, json.Unmarshal(<-dataCh, &addrs))
		require.NoError(t, err)
		assert.Equal(t, pk1, addrs[0].PubKey)
		assert.Equal(t, uint16(4), addrs[0].Port)
		assert.Equal(t, pk2, addrs[1].PubKey)
		assert.Equal(t, uint16(3), addrs[1].Port)
	})

	t.Run("loop closed", func(t *testing.T) {
		rule, err := rt.Rule(3)
		require.NoError(t, err)
		require.NotNil(t, rule)
		assert.Equal(t, routing.RuleApp, rule.Type())

		require.NoError(t, setup.LoopClosed(sProto, &setup.LoopData{RemotePK: pk2, RemotePort: 3, LocalPort: 4}))
		time.Sleep(100 * time.Millisecond)

		_, err = r.pm.GetLoop(4, &app.Addr{PubKey: pk2, Port: 3})
		require.Error(t, err)
		_, err = r.pm.Get(4)
		require.NoError(t, err)

		rule, err = rt.Rule(3)
		require.NoError(t, err)
		require.Nil(t, rule)
	})

	t.Run("delete rule", func(t *testing.T) {
		require.NoError(t, setup.DeleteRule(sProto, routeID))

		rule, err := rt.Rule(routeID)
		require.NoError(t, err)
		assert.Nil(t, rule)
	})
}

func TestRouterSetupLoop(t *testing.T) {
	client := transport.NewDiscoveryMock()
	logStore := transport.InMemoryTransportLogStore()

	pk1, sk1 := cipher.GenerateKeyPair()
	pk2, sk2 := cipher.GenerateKeyPair()

	f1, f2 := transport.NewMockFactory(pk1, pk2)
	f1.SetType("messaging")
	f2.SetType("messaging")

	m1, err := transport.NewManager(&transport.ManagerConfig{PubKey: pk1, SecKey: sk1, DiscoveryClient: client, LogStore: logStore}, f1)
	require.NoError(t, err)

	m2, err := transport.NewManager(&transport.ManagerConfig{PubKey: pk2, SecKey: sk2, DiscoveryClient: client, LogStore: logStore}, f2)
	require.NoError(t, err)
	go m2.Serve(context.TODO()) // nolint: errcheck

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
		acceptCh, _ := m2.Observe()
		tr := <-acceptCh

		proto := setup.NewProtocol(tr)
		p, data, err := proto.ReadPacket()
		if err != nil {
			errCh <- err
			return
		}

		if p != setup.PacketCreateLoop {
			errCh <- errors.New("unknown command")
			return
		}

		l := &routing.Loop{}
		if err := json.Unmarshal(data, l); err != nil {
			errCh <- err
			return
		}

		if l.LocalPort != 10 || l.RemotePort != 6 {
			errCh <- errors.New("invalid payload")
			return
		}

		errCh <- proto.WritePacket(setup.RespSuccess, []byte{})
	}()

	rw, rwIn := net.Pipe()
	go r.ServeApp(rwIn, 5, &app.Config{}) // nolint: errcheck
	proto := app.NewProtocol(rw)
	go proto.Serve(nil) // nolint: errcheck

	addr := &app.Addr{}
	require.NoError(t, proto.Send(app.FrameCreateLoop, &app.Addr{PubKey: pk2, Port: 6}, addr))

	require.NoError(t, <-errCh)
	ll, err := r.pm.GetLoop(10, &app.Addr{PubKey: pk2, Port: 6})
	require.NoError(t, err)
	require.NotNil(t, ll)
	require.NotNil(t, ll.noise)

	assert.Equal(t, pk1, addr.PubKey)
	assert.Equal(t, uint16(10), addr.Port)
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
	go r.ServeApp(rwIn, 5, &app.Config{}) // nolint: errcheck
	proto := app.NewProtocol(rw)
	go proto.Serve(nil) // nolint: errcheck

	addr := &app.Addr{}
	require.NoError(t, proto.Send(app.FrameCreateLoop, &app.Addr{PubKey: pk, Port: 5}, addr))

	ll, err := r.pm.GetLoop(10, &app.Addr{PubKey: pk, Port: 5})
	require.NoError(t, err)
	require.NotNil(t, ll)

	assert.Equal(t, pk, addr.PubKey)
	assert.Equal(t, uint16(10), addr.Port)
}

func TestRouterCloseLoop(t *testing.T) {
	client := transport.NewDiscoveryMock()
	logStore := transport.InMemoryTransportLogStore()

	pk1, sk1 := cipher.GenerateKeyPair()
	pk2, sk2 := cipher.GenerateKeyPair()
	pk3, _ := cipher.GenerateKeyPair()

	f1, f2 := transport.NewMockFactory(pk1, pk2)
	f1.SetType("messaging")

	m1, err := transport.NewManager(&transport.ManagerConfig{PubKey: pk1, SecKey: sk1, DiscoveryClient: client, LogStore: logStore}, f1)
	require.NoError(t, err)

	m2, err := transport.NewManager(&transport.ManagerConfig{PubKey: pk2, SecKey: sk2, DiscoveryClient: client, LogStore: logStore}, f2)
	require.NoError(t, err)
	go m2.Serve(context.TODO()) // nolint: errcheck

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
		acceptCh, _ := m2.Observe()
		tr := <-acceptCh

		proto := setup.NewProtocol(tr)
		p, data, err := proto.ReadPacket()
		if err != nil {
			errCh <- err
			return
		}

		if p != setup.PacketCloseLoop {
			errCh <- errors.New("unknown command")
			return
		}

		ld := &setup.LoopData{}
		if err := json.Unmarshal(data, ld); err != nil {
			errCh <- err
			return
		}

		if ld.LocalPort != 5 || ld.RemotePort != 6 || ld.RemotePK != pk3 {
			errCh <- errors.New("invalid payload")
			return
		}

		errCh <- proto.WritePacket(setup.RespSuccess, []byte{})
	}()

	rw, rwIn := net.Pipe()
	go r.ServeApp(rwIn, 5, &app.Config{}) // nolint: errcheck
	proto := app.NewProtocol(rw)
	go proto.Serve(nil) // nolint: errcheck

	time.Sleep(100 * time.Millisecond)

	raddr := &app.Addr{PubKey: pk3, Port: 6}
	require.NoError(t, r.pm.SetLoop(5, raddr, &loop{}))

	require.NoError(t, proto.Send(app.FrameClose, &app.LoopAddr{Port: 5, Remote: *raddr}, nil))

	time.Sleep(100 * time.Millisecond)

	require.NoError(t, <-errCh)
	_, err = r.pm.GetLoop(5, &app.Addr{PubKey: pk3, Port: 6})
	require.Error(t, err)
	_, err = r.pm.Get(5)
	require.NoError(t, err)

	rule, err = rt.Rule(routeID)
	require.NoError(t, err)
	require.Nil(t, rule)
}

func TestRouterCloseLoopOnAppClose(t *testing.T) {
	client := transport.NewDiscoveryMock()
	logStore := transport.InMemoryTransportLogStore()

	pk1, sk1 := cipher.GenerateKeyPair()
	pk2, sk2 := cipher.GenerateKeyPair()
	pk3, _ := cipher.GenerateKeyPair()

	f1, f2 := transport.NewMockFactory(pk1, pk2)
	f1.SetType("messaging")

	m1, err := transport.NewManager(&transport.ManagerConfig{PubKey: pk1, SecKey: sk1, DiscoveryClient: client, LogStore: logStore}, f1)
	require.NoError(t, err)

	m2, err := transport.NewManager(&transport.ManagerConfig{PubKey: pk2, SecKey: sk2, DiscoveryClient: client, LogStore: logStore}, f2)
	require.NoError(t, err)
	go m2.Serve(context.TODO()) // nolint: errcheck

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
		acceptCh, _ := m2.Observe()
		tr := <-acceptCh

		proto := setup.NewProtocol(tr)
		p, data, err := proto.ReadPacket()
		if err != nil {
			errCh <- err
			return
		}

		if p != setup.PacketCloseLoop {
			errCh <- errors.New("unknown command")
			return
		}

		ld := &setup.LoopData{}
		if err := json.Unmarshal(data, ld); err != nil {
			errCh <- err
			return
		}

		if ld.LocalPort != 5 || ld.RemotePort != 6 || ld.RemotePK != pk3 {
			errCh <- errors.New("invalid payload")
			return
		}

		errCh <- proto.WritePacket(setup.RespSuccess, []byte{})
	}()

	rw, rwIn := net.Pipe()
	go r.ServeApp(rwIn, 5, &app.Config{}) // nolint: errcheck
	proto := app.NewProtocol(rw)
	go proto.Serve(nil) // nolint: errcheck

	time.Sleep(100 * time.Millisecond)

	raddr := &app.Addr{PubKey: pk3, Port: 6}
	require.NoError(t, r.pm.SetLoop(5, raddr, &loop{}))

	require.NoError(t, rw.Close())

	time.Sleep(100 * time.Millisecond)

	require.NoError(t, <-errCh)
	_, err = r.pm.Get(5)
	require.Error(t, err)

	rule, err = rt.Rule(routeID)
	require.NoError(t, err)
	require.Nil(t, rule)
}

func TestRouterCloseLoopOnRouterClose(t *testing.T) {
	client := transport.NewDiscoveryMock()
	logStore := transport.InMemoryTransportLogStore()

	pk1, sk1 := cipher.GenerateKeyPair()
	pk2, sk2 := cipher.GenerateKeyPair()
	pk3, _ := cipher.GenerateKeyPair()

	f1, f2 := transport.NewMockFactory(pk1, pk2)
	f1.SetType("messaging")

	m1, err := transport.NewManager(&transport.ManagerConfig{PubKey: pk1, SecKey: sk1, DiscoveryClient: client, LogStore: logStore}, f1)
	require.NoError(t, err)

	m2, err := transport.NewManager(&transport.ManagerConfig{PubKey: pk2, SecKey: sk2, DiscoveryClient: client, LogStore: logStore}, f2)
	require.NoError(t, err)
	go m2.Serve(context.TODO()) // nolint: errcheck

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
		acceptCh, _ := m2.Observe()
		tr := <-acceptCh

		proto := setup.NewProtocol(tr)
		p, data, err := proto.ReadPacket()
		if err != nil {
			errCh <- err
			return
		}

		if p != setup.PacketCloseLoop {
			errCh <- errors.New("unknown command")
			return
		}

		ld := &setup.LoopData{}
		if err := json.Unmarshal(data, ld); err != nil {
			errCh <- err
			return
		}

		if ld.LocalPort != 5 || ld.RemotePort != 6 || ld.RemotePK != pk3 {
			errCh <- errors.New("invalid payload")
			return
		}

		errCh <- proto.WritePacket(setup.RespSuccess, []byte{})
	}()

	rw, rwIn := net.Pipe()
	go r.ServeApp(rwIn, 5, &app.Config{}) // nolint: errcheck
	proto := app.NewProtocol(rw)
	go proto.Serve(nil) // nolint: errcheck

	time.Sleep(100 * time.Millisecond)

	raddr := &app.Addr{PubKey: pk3, Port: 6}
	require.NoError(t, r.pm.SetLoop(5, raddr, &loop{}))

	require.NoError(t, r.Close())

	time.Sleep(100 * time.Millisecond)

	require.NoError(t, <-errCh)
	_, err = r.pm.Get(5)
	require.Error(t, err)

	rule, err = rt.Rule(routeID)
	require.NoError(t, err)
	require.Nil(t, rule)
}

func TestRouterRouteExpiration(t *testing.T) {
	client := transport.NewDiscoveryMock()
	logStore := transport.InMemoryTransportLogStore()

	pk, sk := cipher.GenerateKeyPair()
	m, err := transport.NewManager(&transport.ManagerConfig{PubKey: pk, SecKey: sk, DiscoveryClient: client, LogStore: logStore})
	require.NoError(t, err)

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
	go r.Serve(context.TODO()) // nolint

	time.Sleep(110 * time.Millisecond)

	assert.Equal(t, 0, rt.Count())
	require.NoError(t, r.Close())
}

func noiseInstances(t *testing.T, pkI, pkR cipher.PubKey, skI, skR cipher.SecKey) (ni, nr *noise.Noise) {
	t.Helper()

	var err error
	confI := noise.Config{
		LocalSK:   skI,
		LocalPK:   pkI,
		RemotePK:  pkR,
		Initiator: true,
	}

	confR := noise.Config{
		LocalSK:   skR,
		LocalPK:   pkR,
		RemotePK:  pkI,
		Initiator: false,
	}

	ni, err = noise.KKAndSecp256k1(confI)
	require.NoError(t, err)

	nr, err = noise.KKAndSecp256k1(confR)
	require.NoError(t, err)

	msg, err := ni.HandshakeMessage()
	require.NoError(t, err)
	require.NoError(t, nr.ProcessMessage(msg))

	res, err := nr.HandshakeMessage()
	require.NoError(t, err)
	require.NoError(t, ni.ProcessMessage(res))
	return ni, nr
}
