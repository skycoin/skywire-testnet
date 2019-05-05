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

func Example_router() {
	logger := logging.MustGetLogger("router")
	pk, sk := cipher.GenerateKeyPair()
	conf := &Config{
		PubKey:     pk,
		SecKey:     sk,
		SetupNodes: []cipher.PubKey{},
	}

	// Look into CfgStep AddProcManagerAndRouter() for more correct example
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

func Example_router_setupProto() {
	env := &TEnv{}
	_, err := env.Run(
		GenerateDeterministicKeys(),
		AddTransportManagers(),
		AddProcManagerAndRouter(),
		StartSetupTransportManager(),
	)
	fmt.Printf("env.Run success: %v\n", err == nil)

	pr, tr, err := env.R.setupProto(context.TODO())

	fmt.Printf("router.setupProto:\n\tProtocol: %T\n\tTransport %T\n\terror: %v\n", pr, tr, err)
	env.PrintTearDown()

	// Output: env.Run success: true
	// router.setupProto:
	// 	Protocol: *setup.Protocol
	// 	Transport *transport.ManagedTransport
	// 	error: <nil>
	// env.TearDown() success: true

}

// TODO(alex): test for existing transport
func Example_router_ForwardPacket() {

	env := &TEnv{}
	_, err := env.Run(
		GenerateDeterministicKeys(),
		AddTransportManagers(),
		AddProcManagerAndRouter(),
		StartSetupTransportManager(),
	)
	fmt.Printf("env.Run success: %v\n", err == nil)

	trID := uuid.New()
	expireAt := time.Now().Add(2 * time.Minute)
	fwdRule := routing.ForwardRule(expireAt, 2, trID)

	payload := []byte("ForwardPacket")
	if err := env.R.ForwardPacket(fwdRule.TransportID(), fwdRule.RouteID(), payload); err != nil {
		fmt.Printf("router.ForwardPacket error: %v\n", err)
	}

	// Output: env.Run success: true
	// router.ForwardPacket error: transport not found
}

func Example_router_handleSetup() {

	env := &TEnv{}
	_, err := env.Run(
		GenerateDeterministicKeys(),
		AddTransportManagers(),
		AddProcManagerAndRouter(),
		AddSetupHandlersEnv(),
		StartSetupTransportManager(),
	)
	fmt.Printf("env.Run success: %v\n", err == nil)

	errCh := make(chan error, 1)
	go func() {
		errCh <- env.R.handleSetup(env.procMgr, env.connInit)
	}()

	sprotoInit := setup.NewSetupProtocol(env.connResp)
	sprotoInit.WritePacket(setup.PacketType(0), "Hello") //nolint: errcheck
	pt, data, err := sprotoInit.ReadPacket()
	fmt.Printf("handle success: %v\n", <-errCh == nil)
	fmt.Printf("response: %v %v %v\n", pt, string(data), err)

	// Output: env.Run success: true
	// handle success: true
	// response: RespFailure "json: cannot unmarshal string into Go value of type []routing.Rule" <nil>

}

// Old test. Does not make sense now
func TestRouterCloseLoop(t *testing.T) {
	env := &TEnv{}
	_, err := env.Run(
		GenerateDeterministicKeys(),
		AddTransportManagers(),
		AddProcManagerAndRouter(),
	)
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

// This test is mostly meaningless now - expiration is done by RoutingTableManager
func TestRouterRouteExpiration(t *testing.T) {

	env := &TEnv{}
	_, err := env.Run(
		GenerateKeys(),
		AddTransportManagers(),
		AddProcManagerAndRouter(),
		StartSetupTransportManager(),
		AddSetupHandlersEnv(),
	)
	fmt.Printf("TEnv success: %v\n", err == nil)

	// Add expired ForwardRule
	trID := uuid.New()
	expireAt := time.Now().Add(-10 * time.Millisecond)
	rules := []routing.Rule{
		routing.ForwardRule(expireAt, 2, trID),
	}
	_, err = env.stpHandlers.addRules(rules)

	assert.NoError(t, err)
	assert.Equal(t, 1, env.routingTable.Count())

	// Set RoutingTableManager ticker to fast cleanup
	env.rtm.ticker = time.NewTicker(10 * time.Millisecond)
	go env.R.Serve(context.TODO(), env.procMgr) // nolint

	time.Sleep(time.Second)

	assert.Equal(t, 0, env.routingTable.Count())

	env.NoErrorTearDown(t)
}
