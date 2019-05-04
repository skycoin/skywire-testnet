package router

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/skycoin/skycoin/src/util/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/pkg/cipher"
	routeFinder "github.com/skycoin/skywire/pkg/route-finder/client"
	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/setup"
	"github.com/skycoin/skywire/pkg/transport"
)

func TestMain(m *testing.M) {
	logging.SetOutputTo(ioutil.Discard)
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

func Example_router_setupProto() {

	env, err := makeMockRouterEnv()
	fmt.Printf("Environment created: %v\n", err == nil)
	env.runStepsAsExamples(true, StartSetupTransportManager())

	pr, tr, err := env.R.setupProto(context.TODO())

	fmt.Printf("Protocol: %T\nTransport %T\nerror: %v\n", pr, tr, err)

	// Output: Environment created: true
	// Protocol: *setup.Protocol
	// Transport *transport.ManagedTransport
	// error: <nil>

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

	env, err := makeMockRouterEnv()
	fmt.Printf("makeMockRouterEnv success: %v\n", err == nil)

	trID := uuid.New()
	expireAt := time.Now().Add(2 * time.Minute)
	fwdRule := routing.ForwardRule(expireAt, 2, trID)

	payload := []byte("ForwardPacket")
	if err := env.R.ForwardPacket(fwdRule.TransportID(), fwdRule.RouteID(), payload); err != nil {
		fmt.Printf("router.ForwardPacket error: %v\n", err)
	}

	// Output: makeMockRouterEnv success: true
	// router.ForwardPacket error: transport not found
}

func Example_router_handleSetup() {

	env := &testEnv{}
	_, err := env.runSteps(
		GenKeys(),
		AddTransportManagers(),
		AddProcManagerAndRouter(),
		AddSetupHandlersEnv(),
	)
	fmt.Printf("testEnv.runSteps success: %v\n", err == nil)

	errCh := make(chan error, 1)
	go func() {
		errCh <- env.R.handleSetup(env.procMgr, env.SH.connInit)
	}()

	sprotoInit := setup.NewSetupProtocol(env.SH.connResp)
	sprotoInit.WritePacket(setup.PacketType(0), "Hello") //nolint: errcheck
	pt, data, err := sprotoInit.ReadPacket()
	fmt.Printf("handle success: %v\n", <-errCh == nil)
	fmt.Printf("response: %v %v %v\n", pt, string(data), err)

	// Output: makeSetupHandlersEnv success: true
	// handle success: true
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

		if ld.LocalPort != 5 || ld.RemotePort != 6 || ld.RemotePK != env.pkRemote {
			errCh <- errors.New("invalid payload")
			return
		}

		errCh <- proto.WritePacket(setup.RespSuccess, []byte{})
	}()

	time.Sleep(100 * time.Millisecond)

}

// This test is mostly mmeaningless now - expiration is done by RoutingTableManager
func TestRouterRouteExpiration(t *testing.T) {

	env := &testEnv{}
	_, err := env.runSteps(
		GenKeys(),
		AddTransportManagers(),
		AddProcManagerAndRouter(),
		StartSetupTransportManager(),
		AddSetupHandlersEnv(),
	)
	fmt.Printf("testEnv success: %v\n", err == nil)
	defer env.TearDown()

	// Add expired ForwardRule
	trID := uuid.New()
	expireAt := time.Now().Add(-10 * time.Millisecond)
	rules := []routing.Rule{
		routing.ForwardRule(expireAt, 2, trID),
	}
	_, err = env.SH.stpHandlers.addRules(rules)

	assert.NoError(t, err)
	assert.Equal(t, 1, env.routingTable.Count())

	// // Set RoutingTableManager ticker to fast cleanup
	// env.rtm.ticker = time.NewTicker(10 * time.Millisecond)
	// go env.R.Serve(context.TODO(), env.procMgr) // nolint

	// time.Sleep(time.Second)

	// assert.Equal(t, 0, env.routingTable.Count())
	// require.NoError(t, envSh.env.R.Close())
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

		if ld.LocalPort != 5 || ld.RemotePort != 6 || ld.RemotePK != env.pkRemote {
			errCh <- errors.New("invalid payload")
			return
		}

		errCh <- proto.WritePacket(setup.RespSuccess, []byte{})
	}()

	// rw, rwIn := net.Pipe()
	// // go r.ServeApp(rwIn, 5) // nolint: errcheck

	go env.R.Serve(context.TODO(), env.procMgr) // nolint: errcheck

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
