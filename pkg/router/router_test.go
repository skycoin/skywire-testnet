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

	// r := makeMockRouter()
	env, _ := makeMockEnv()
	r := env.r

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
	// env, _ := makeMockEnv()
	// r := env.r

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
	env, err := makeMockEnv()
	if err != nil {
		fmt.Printf("makeMockEnv: %v\n", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- env.r.handleTransport(env.pm, env.connInit)
	}()

	time.Sleep(time.Second)
	close(errCh)

	// Output:
}

func Example_router_Serve() {
	env, err := makeMockEnv()
	if err != nil {
		fmt.Printf("makeMockEnv: %v\n", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- env.r.Serve(context.TODO(), env.pm)
	}()

	time.Sleep(time.Second)
	close(errCh)
	// fmt.Printf("r.Serve: %v\n", <-errCh)
	// Output:
}

func Example_router_handleSetup() {

	env, err := makeMockEnv()
	if err != nil {
		fmt.Printf("makeMockEnv: %v\n", err)
	}

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
