package router

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/skycoin/skycoin/src/util/logging"
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

func Example_router_handleSetup() {

	env, err := makeMockEnv()
	if err != nil {
		fmt.Printf("makeMockEnv: %v\n", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- env.r.handleSetup(env.pm, env.connResp)
	}()

	sprotoInit := setup.NewSetupProtocol(env.connInit)
	sprotoInit.WritePacket(setup.PacketType(0), "Hello") //nolint: errcheck
	pt, data, err := sprotoInit.ReadPacket()
	fmt.Printf("handle success: %v\n", <-errCh == nil)
	fmt.Printf("response: %v %v %v\n", pt, string(data), err)

	// Output: handle success: true
	// response: RespFailure "json: cannot unmarshal string into Go value of type []routing.Rule" <nil>

}

func Example_router_handleTransport() {

	//
}

func Example_router_Serve() {

	//
}

func TestRouterCloseLoopOnAppClose(t *testing.T) {
	client := transport.NewDiscoveryMock()
	rfc := routeFinder.NewMock()
	logStore := transport.InMemoryTransportLogStore()

	pk1, sk1 := cipher.GenerateKeyPair()
	pk2, sk2 := cipher.GenerateKeyPair()
	pk3, _ := cipher.GenerateKeyPair()

	f1, f2 := transport.NewMockFactoryPair(pk1, pk2)
	f1.SetType("messaging")

	tpm1, err := transport.NewManager(&transport.ManagerConfig{PubKey: pk1, SecKey: sk1, DiscoveryClient: client, LogStore: logStore}, f1)
	require.NoError(t, err)

	tpm2, err := transport.NewManager(&transport.ManagerConfig{PubKey: pk2, SecKey: sk2, DiscoveryClient: client, LogStore: logStore}, f2)
	require.NoError(t, err)
	go tpm2.Serve(context.TODO()) // nolint: errcheck

	rt := routing.InMemoryRoutingTable()
	rule := routing.AppRule(time.Now().Add(time.Hour), 4, pk3, 6, 5)
	routeID, err := rt.AddRule(rule)
	require.NoError(t, err)
	fmt.Printf("%v\n", routeID)

	conf := &Config{
		PubKey:     pk1,
		SecKey:     sk1,
		SetupNodes: []cipher.PubKey{pk2},
	}
	logger := logging.MustGetLogger("routesetup")

	r := New(logger, tpm1, rt, rfc, conf)
	fmt.Printf("%v\n", r)

	errCh := make(chan error)
	go func() {
		acceptCh, _ := tpm2.Observe()
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

		if ld.LocalPort != 5 || ld.RemotePort != 6 || ld.RemotePK != pk3 {
			errCh <- errors.New("invalid payload")
			return
		}

		errCh <- proto.WritePacket(setup.RespSuccess, []byte{})
	}()

	// rw, rwIn := net.Pipe()
	// // go r.ServeApp(rwIn, 5) // nolint: errcheck
	pm := NewProcManager(10)
	go r.Serve(context.TODO(), pm)
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
