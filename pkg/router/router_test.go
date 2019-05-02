package router

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/skycoin/skycoin/src/util/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/internal/noise"
	"github.com/skycoin/skywire/pkg/app"
	"github.com/skycoin/skywire/pkg/cipher"
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

func ExampleNew() {
	logger := logging.MustGetLogger("router")
	_, tpm, _ := transport.MockTransportManager() //nolint: errcheck
	rtm := NewRoutingTableManager(
		logging.MustGetLogger("rt_manager"),
		routing.InMemoryRoutingTable(),
		DefaultRouteKeepalive,
		DefaultRouteCleanupDuration)

	rfc := routeFinder.NewMock()

	pk, sk := cipher.GenerateKeyPair()
	rConf := &Config{
		PubKey:     pk,
		SecKey:     sk,
		SetupNodes: []cipher.PubKey{},
	}

	r := New(logger, tpm, rtm, rfc, rConf)
	fmt.Printf("Router created: %v\n", r != nil)

	// Output: Router created: true
}

func Example_router() {

	logger := logging.MustGetLogger("router")

	pk, sk := cipher.GenerateKeyPair()
	conf := &Config{
		PubKey:     pk,
		SecKey:     sk,
		SetupNodes: []cipher.PubKey{},
	}

	// TODO(alex): substitute with cleaner implementation
	_, tpm, _ := transport.MockTransportManager() //nolint: errcheck
	rtm := NewRoutingTableManager(
		logging.MustGetLogger("rt_manager"),
		routing.InMemoryRoutingTable(),
		DefaultRouteKeepalive,
		DefaultRouteCleanupDuration)

	r := router{
		log:  logger,
		conf: conf,
		tpm:  tpm,
		rtm:  rtm,
		rfc:  routeFinder.NewMock(),
	}

	fmt.Printf("r.conf is empty: %v\n", r.conf == &Config{})

	//Output: r.conf is empty: false
}

// TODO(alex): test for existing transport
func Example_router_ForwardPacket() {

	r := makeMockRouter()

	trID := uuid.New()
	expireAt := time.Now().Add(2 * time.Minute)
	fwdRule := routing.ForwardRule(expireAt, 2, trID)

	payload := []byte("ForwardPacket")
	if err := r.ForwardPacket(fwdRule.TransportID(), fwdRule.RouteID(), payload); err != nil {
		fmt.Printf("router.ForwardPacket error: %v\n", err)
	}

	// Output: router.ForwardPacket error: transport not found
}

func Example_router_FetchRouteAndSetupLoop() {

	r := makeMockRouter()
	initPK, initSK, _ := cipher.GenerateDeterministicKeyPair([]byte("init")) // nolint: errcheck
	respPK, _, _ := cipher.GenerateDeterministicKeyPair([]byte("resp"))      // nolint: errcheck

	// prepare noise
	ns, err := noise.KKAndSecp256k1(noise.Config{
		LocalPK:   initPK,
		LocalSK:   initSK,
		RemotePK:  respPK,
		Initiator: true,
	})
	if err != nil {
		fmt.Printf("noise.KKAndSecp256k1 error: %v\n", err)
	}
	msg, err := ns.HandshakeMessage()
	if err != nil {
		fmt.Printf("ns.HandshakeMessage error: %v\n", err)
	}

	// allocate local listening port for the new loop
	// lPort := ar.pm.AllocPort(ar.pid)

	lm := app.LoopMeta{
		Local:  app.LoopAddr{PubKey: initPK, Port: 0},
		Remote: app.LoopAddr{PubKey: respPK, Port: 0},
	}

	err = r.FindRoutesAndSetupLoop(lm, msg)
	fmt.Printf("FindRoutesAndSetupLoop error: %v\n", err)

	// Output: FindRoutesAndSetupLoop error: route setup: no nodes
}

func Example_router_CloseLoop() {
	r := makeMockRouter()
	initPK, _, _ := cipher.GenerateDeterministicKeyPair([]byte("init")) // nolint: errcheck
	respPK, _, _ := cipher.GenerateDeterministicKeyPair([]byte("resp")) // nolint: errcheck

	lm := app.LoopMeta{
		Local:  app.LoopAddr{PubKey: initPK, Port: 0},
		Remote: app.LoopAddr{PubKey: respPK, Port: 0},
	}

	err := r.CloseLoop(lm)
	fmt.Printf("CloseLoop error: %v\n", err)

	// Output: CloseLoop error: route setup: no nodes
}

func Example_router_handleTransport() {

	//
}

func Example_router_Serve() {

	//
}

func Example_router_handleSetup() {

	env, err := makeMockEnv()
	if err != nil {
		fmt.Printf("makeMockEnv: %v\n", err)
	}

	// // trInitCh, _ := env.t
	// trLocalCh, _ := env.tpmLocal.Observe()
	// trLocal := <-trLocalCh
	// trSetupCh, _ := env.tpmSetup.Observe()
	// trSetup := <-trSetupCh

	errCh := make(chan error, 1)
	go func() {
		errCh <- env.r.handleSetup(env.pm, env.connInit)
	}()

	sprotoInit := setup.NewSetupProtocol(env.connResp)
	sprotoInit.WritePacket(setup.PacketType(0), "Hello") //nolint: errcheck
	pt, data, err := sprotoInit.ReadPacket()
	fmt.Printf("handle success: %v\n", <-errCh == nil)
	fmt.Printf("response: %v %v %v\n", pt, string(data), err)

	// Output: handle success: true
	// response: RespFailure "json: cannot unmarshal string into Go value of type []routing.Rule" <nil>

}

type mockRouterEnv struct {
	r  *router
	pm ProcManager

	tpmLocal *transport.Manager
	tpmSetup *transport.Manager

	pkSetup   cipher.PubKey
	pkRespond cipher.PubKey

	rt routing.Table

	routeID routing.RouteID
}

func makeMockRouter() *router {
	logger := logging.MustGetLogger("router")

	pk, sk := cipher.GenerateKeyPair()

	// TODO(alexyu): SetupNodes

	conf := &Config{
		PubKey:     pk,
		SecKey:     sk,
		SetupNodes: []cipher.PubKey{},
	}

	// TODO(alexyu):  This mock must be simplified
	_, tpm, _ := transport.MockTransportManager() //nolint: errcheck
	rtm := NewRoutingTableManager(
		logging.MustGetLogger("rt_manager"),
		routing.InMemoryRoutingTable(),
		DefaultRouteKeepalive,
		DefaultRouteCleanupDuration)

	return &router{
		log:  logger,
		conf: conf,
		tpm:  tpm,
		rtm:  rtm,
		rfc:  routeFinder.NewMock(),
	}
}

func makeMockEnv() (*mockEnv, error) {
	connInit, connResp := net.Pipe()
	// r := makeMockRouter()
	env, err := makeMockRouterEnv()
	if err != nil {
		return &mockEnv{}, err
	}
	r := env.r

	pm := NewProcManager(10) //IDK why it's 10
	sprotoInit := setup.NewSetupProtocol(connInit)

	errCh := make(chan error, 1)
	go func() {
		errCh <- sprotoInit.WritePacket(setup.PacketType(0), []byte{})
	}()

	sh, err := makeSetupHandlers(r, pm, connResp)
	if err != nil {
		return &mockEnv{}, err
	}

	return &mockEnv{r, pm, connResp, connInit, sh, <-errCh}, nil
}

func makeMockRouterEnv() (*mockRouterEnv, error) {
	dClient := transport.NewDiscoveryMock()
	rfc := routeFinder.NewMock()
	logStore := transport.InMemoryTransportLogStore()

	pkLocal, skLocal, _ := cipher.GenerateDeterministicKeyPair([]byte("local"))
	pkSetup, skSetup, _ := cipher.GenerateDeterministicKeyPair([]byte("setup"))
	pkRespond, _, _ := cipher.GenerateDeterministicKeyPair([]byte("respond"))

	fLocal, fSetup := transport.NewMockFactoryPair(pkLocal, pkSetup)
	fLocal.SetType("messaging")

	tpmLocal, err := transport.NewManager(&transport.ManagerConfig{PubKey: pkLocal, SecKey: skLocal, DiscoveryClient: dClient, LogStore: logStore}, fLocal)
	if err != nil {
		return &mockRouterEnv{}, err
	}

	tpmSetup, err := transport.NewManager(&transport.ManagerConfig{PubKey: pkSetup, SecKey: skSetup, DiscoveryClient: dClient, LogStore: logStore}, fSetup)
	if err != nil {
		return &mockRouterEnv{}, err
	}

	go tpmSetup.Serve(context.TODO()) // nolint: errcheck

	rt := routing.InMemoryRoutingTable()

	rule := routing.AppRule(time.Now().Add(time.Hour), 4, pkRespond, 6, 5)
	routeID, err := rt.AddRule(rule)
	if err != nil {
		return &mockRouterEnv{}, err
	}

	logger := logging.MustGetLogger("mockRouterEnv")

	conf := &Config{
		PubKey:     pkLocal,
		SecKey:     skLocal,
		SetupNodes: []cipher.PubKey{pkSetup},
	}

	r := &router{
		log:  logger,
		conf: conf,
		tpm:  tpmLocal,
		rtm:  NewRoutingTableManager(logger, rt, DefaultRouteKeepalive, DefaultRouteCleanupDuration),
		rfc:  rfc,
	}
	pm := NewProcManager(10)

	return &mockRouterEnv{
		r:  r,
		pm: pm,

		tpmSetup: tpmSetup,
		tpmLocal: tpmLocal,

		pkSetup:   pkSetup,
		pkRespond: pkRespond,

		routeID: routeID,
		rt:      rt,
	}, nil
}

func (rEnv *mockRouterEnv) TearDown() error {
	errCh := make(chan error, 4)
	errCh <- rEnv.tpmLocal.Close()
	errCh <- rEnv.tpmSetup.Close()
	errCh <- rEnv.r.Close()
	errCh <- rEnv.pm.Close()

	for err := range errCh {
		if err != nil {
			close(errCh)
			return err
		}
	}
	close(errCh)
	return nil
}

// func TestRouterSetupLoopLocal(t *testing.T) {

// 	env, err := makeMockRouterEnv()
// 	pk, sk := cipher.GenerateKeyPair()
// 	conf := &Config{
// 		Logger: logging.MustGetLogger("routesetup"),
// 		PubKey: pk,
// 		SecKey: sk,
// 	}
// 	r := New(conf)

// 	appConn, hostConn := net.Pipe()
// 	go r.ServeApp(hostConn, 5) // nolint: errcheck
// 	fmt.Println("TEST: Served router.")

// 	// Emulate App.
// 	appProto := appnet.NewProtocol(appConn)
// 	go appProto.Serve(appnet.HandlerMap{
// 		appnet.FrameConfirmLoop: func(*appnet.Protocol, []byte) ([]byte, error) {
// 			return nil, nil
// 		},
// 	}) // nolint: errcheck
// 	fmt.Println("TEST: Served app.")

// 	rAddr := app.LoopAddr{PubKey: pk, Port: 5}

// 	resp, err := appProto.Call(appnet.FrameCreateLoop, rAddr.Encode())
// 	require.NoError(t, err)
// 	fmt.Println("TEST: App call okay!")

// 	var lm app.LoopMeta
// 	require.NoError(t, lm.Decode(resp))

// 	ll, err := r.pm.GetLoop(10, &app.LoopAddr{PubKey: pk, Port: 5})
// 	require.NoError(t, err)
// 	require.NotNil(t, ll)

// 	assert.Equal(t, pk, lm.Local.PubKey)
// 	assert.Equal(t, uint16(10), lm.Local.Port)
// }

// Old test. Does not make sense now
func TestRouterCloseLoop(t *testing.T) {
	env, err := makeMockRouterEnv()
	require.NoError(t, err)

	errCh := make(chan error)
	go func() {
		acceptCh, _ := env.tpmSetup.Observe()
		tr := <-acceptCh

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

		ld := &setup.LoopData{}
		if err := json.Unmarshal(data, ld); err != nil {
			errCh <- err
			return
		}

		if ld.LocalPort != 5 || ld.RemotePort != 6 || ld.RemotePK != env.pkRespond {
			errCh <- errors.New("invalid payload")
			return
		}

		errCh <- proto.WritePacket(setup.RespSuccess, []byte{})
	}()

	time.Sleep(100 * time.Millisecond)

}

// This test is mmeaningless now
func TestRouterRouteExpiration(t *testing.T) {
	env, err := makeMockRouterEnv()
	require.NoError(t, err)

	go env.r.Serve(context.TODO(), env.pm) // nolint

	time.Sleep(110 * time.Millisecond)

	assert.Equal(t, 1, env.rt.Count())
	require.NoError(t, env.r.Close())
}

func TestRouterAncientTest(t *testing.T) {

	env, err := makeMockRouterEnv()
	require.NoError(t, err)

	errCh := make(chan error)
	go func() {
		acceptCh, _ := env.tpmSetup.Observe()
		tr := <-acceptCh

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

		ld := &setup.LoopData{}
		if err := json.Unmarshal(data, ld); err != nil {
			errCh <- err
			return
		}

		if ld.LocalPort != 5 || ld.RemotePort != 6 || ld.RemotePK != env.pkRespond {
			errCh <- errors.New("invalid payload")
			return
		}

		errCh <- proto.WritePacket(setup.RespSuccess, []byte{})
	}()

	// rw, rwIn := net.Pipe()
	// // go r.ServeApp(rwIn, 5) // nolint: errcheck

	go env.r.Serve(context.TODO(), env.pm)
	// proto := appnet.NewProtocol(rw)
	// go proto.Serve(nil) // nolint: errcheck

	time.Sleep(100 * time.Millisecond)

	// raddr := &app.LoopAddr{PubKey: pk3, Port: 6}
	// require.NoError(t, r.pm.SetLoop(5, raddr, &loop{}))

	// require.NoError(t, rw.Close())

	// time.Sleep(100 * time.Millisecond)

	// require.NoError(t, <-errCh)
	// _, err = r.pm.Get(5)
	// require.Error(t, err)

	// rule, err = rt.Rule(routeID)
	// require.NoError(t, err)
	// require.Nil(t, rule)
}
