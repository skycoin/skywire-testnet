package router

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/skycoin/src/util/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/internal/testhelpers"
	"github.com/skycoin/skywire/pkg/app"
	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/transport"
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
		logging.Disable()
	}

	os.Exit(m.Run())
}

func TestRouterForwarding(t *testing.T) {
	client := transport.NewDiscoveryMock()
	logStore := transport.InMemoryTransportLogStore()

	pk1, sk1 := cipher.GenerateKeyPair()
	pk2, sk2 := cipher.GenerateKeyPair()
	pk3, sk3 := cipher.GenerateKeyPair()

	c1 := &transport.ManagerConfig{PubKey: pk1, SecKey: sk1, DiscoveryClient: client, LogStore: logStore, Logger: log}
	c2 := &transport.ManagerConfig{PubKey: pk2, SecKey: sk2, DiscoveryClient: client, LogStore: logStore, Logger: log}
	c3 := &transport.ManagerConfig{PubKey: pk3, SecKey: sk3, DiscoveryClient: client, LogStore: logStore, Logger: log}

	f1, f2 := transport.NewMockFactoryPair(pk1, pk2)
	f3, f4 := transport.NewMockFactoryPair(pk2, pk3)
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
		Logger:           logging.MustGetLogger("router"),
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

	tr1, err := m1.CreateDataTransport(context.TODO(), pk2, "mock", true)
	require.NoError(t, err)

	tr3, err := m3.CreateDataTransport(context.TODO(), pk2, "mock2", true)
	require.NoError(t, err)

	rule := routing.ForwardRule(time.Now().Add(time.Hour), 4, tr3.Entry.ID)
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
	c1 := &transport.ManagerConfig{
		PubKey:          pk1,
		SecKey:          sk1,
		DiscoveryClient: client,
		LogStore:        logStore,
		Logger:          log}

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
	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- proto.Serve(nil)
	}()

	require.NoError(t, proto.Send(app.FrameInit, &app.Config{AppName: "foo", AppVersion: "0.0.1", ProtocolVersion: "0.0.1"}, nil))
	require.Error(t, proto.Send(app.FrameInit, &app.Config{AppName: "foo1", AppVersion: "0.0.1", ProtocolVersion: "0.0.1"}, nil))

	require.NoError(t, proto.Close())
	require.NoError(t, r.Close())
	require.NoError(t, testhelpers.NoErrorWithinTimeoutN(serveErrCh, errCh))
}

func TestRouterApp(t *testing.T) {
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

	trServeErrCh := make(chan error, 1)
	go func() {
		trServeErrCh <- m2.Serve(context.TODO())
	}()

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
	serveAppErrCh := make(chan error, 1)
	go func() {
		serveAppErrCh <- r.ServeApp(rwIn, 6, &app.Config{})
	}()
	proto := app.NewProtocol(rw)
	dataCh := make(chan []byte)
	protoServeErrCh := make(chan error, 1)
	go func() {
		f := func(_ app.Frame, p []byte) (interface{}, error) {
			go func() { dataCh <- p }()
			return nil, nil
		}
		protoServeErrCh <- proto.Serve(f)
	}()

	time.Sleep(100 * time.Millisecond)

	tr, err := m1.CreateDataTransport(context.TODO(), pk2, "mock", true)
	require.NoError(t, err)

	rule := routing.AppRule(time.Now().Add(time.Hour), 4, pk2, 5, 6)
	routeID, err := rt.AddRule(rule)
	require.NoError(t, err)

	raddr := routing.Addr{PubKey: pk2, Port: 5}
	require.NoError(t, r.pm.SetLoop(6, raddr, &loop{tr.Entry.ID, 4}))

	tr2 := m2.Transport(tr.Entry.ID)
	sendErrCh := make(chan error, 1)
	go func() {
		sendErrCh <- proto.Send(app.FrameSend, &app.Packet{Loop: routing.AddrLoop{Local: routing.Addr{Port: 6}, Remote: raddr}, Payload: []byte("bar")}, nil)
	}()

	packet := make(routing.Packet, 9)
	_, err = tr2.Read(packet)
	require.NoError(t, err)
	assert.Equal(t, uint16(3), packet.Size())
	assert.Equal(t, routing.RouteID(4), packet.RouteID())
	assert.Equal(t, []byte("bar"), packet.Payload())

	_, err = tr2.Write(routing.MakePacket(routeID, []byte("foo")))
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	var aPacket app.Packet
	require.NoError(t, json.Unmarshal(<-dataCh, &aPacket))
	assert.Equal(t, pk2, aPacket.Loop.Remote.PubKey)
	assert.Equal(t, routing.Port(5), aPacket.Loop.Remote.Port)
	assert.Equal(t, routing.Port(6), aPacket.Loop.Local.Port)
	assert.Equal(t, []byte("foo"), aPacket.Payload)

	require.NoError(t, r.Close())
	require.NoError(t, <-errCh)

	require.NoError(t, m2.Close())

	require.NoError(t,
		testhelpers.NoErrorWithinTimeoutN(trServeErrCh, protoServeErrCh, sendErrCh, serveAppErrCh))

}

func TestRouterLocalApp(t *testing.T) {
	client := transport.NewDiscoveryMock()
	logStore := transport.InMemoryTransportLogStore()

	pk, sk := cipher.GenerateKeyPair()
	m, err := transport.NewManager(&transport.ManagerConfig{PubKey: pk, SecKey: sk, DiscoveryClient: client, LogStore: logStore, Logger: log})
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
			Loop: routing.AddrLoop{Local: routing.Addr{Port: 5}, Remote: routing.Addr{PubKey: pk, Port: 6}}, Payload: []byte("foo"),
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
	require.NoError(t,
		testhelpers.NoErrorWithinTimeoutN(errCh, protoServeErr1Ch, protoServeErr2Ch, sendErrCh, serveAppErr1Ch, serveAppErr2Ch))
}

func TestRouterRouteExpiration(t *testing.T) {
	client := transport.NewDiscoveryMock()
	logStore := transport.InMemoryTransportLogStore()

	pk, sk := cipher.GenerateKeyPair()
	m, err := transport.NewManager(
		&transport.ManagerConfig{PubKey: pk, SecKey: sk,
			DiscoveryClient: client, LogStore: logStore, Logger: log})
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
	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- r.Serve(context.TODO())
	}()

	time.Sleep(110 * time.Millisecond)

	assert.Equal(t, 0, rt.Count())
	require.NoError(t, r.Close())

	require.NoError(t, testhelpers.NoErrorWithinTimeout(serveErrCh))
}
