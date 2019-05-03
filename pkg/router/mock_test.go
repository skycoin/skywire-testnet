package router

import (
	"context"
	"fmt"

	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/cipher"
	routeFinder "github.com/skycoin/skywire/pkg/route-finder/client"
	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/transport"
)

type mockRouterEnv struct {
	// Keys
	pkLocal  cipher.PubKey
	skLocal  cipher.SecKey
	pkSetup  cipher.PubKey
	pkRemote cipher.PubKey

	// TransportManagers
	tpmLocal *transport.Manager
	tpmSetup *transport.Manager

	// routing.Table
	routingTable routing.Table
	rtm          *RoutingTableManager

	// ProcManager and router
	pm ProcManager
	r  *router
}

func (env *mockRouterEnv) StartSetupTpm() {
	go env.tpmSetup.Serve(context.TODO()) // nolint: errcheck
}

func makeMockRouterEnv() (*mockRouterEnv, error) {
	// PKs
	pkLocal, skLocal, _ := cipher.GenerateDeterministicKeyPair([]byte("local")) // nolint: errcheck
	pkSetup, skSetup, _ := cipher.GenerateDeterministicKeyPair([]byte("setup")) // nolint: errcheck
	pkRemote, _, _ := cipher.GenerateDeterministicKeyPair([]byte("respond"))    // nolint: errcheck

	// discovery client, route finder client, logStore, logger
	dClient := transport.NewDiscoveryMock()
	rfc := routeFinder.NewMock()
	logStore := transport.InMemoryTransportLogStore()

	// TransportFactories
	fLocal, fSetup := transport.NewMockFactoryPair(pkLocal, pkSetup)
	fLocal.SetType("messaging")

	// TransportManagers
	tpmLocal, err := transport.NewManager(&transport.ManagerConfig{PubKey: pkLocal, SecKey: skLocal, DiscoveryClient: dClient, LogStore: logStore}, fLocal)
	if err != nil {
		return &mockRouterEnv{}, err
	}

	tpmSetup, err := transport.NewManager(&transport.ManagerConfig{PubKey: pkSetup, SecKey: skSetup, DiscoveryClient: dClient, LogStore: logStore}, fSetup)
	if err != nil {
		return &mockRouterEnv{}, err
	}

	// RoutingTable
	routingTable := routing.InMemoryRoutingTable()

	// ProcManager & Router
	pm := NewProcManager(10)
	logger := logging.MustGetLogger("mockRouterEnv")
	conf := &Config{
		PubKey:     pkLocal,
		SecKey:     skLocal,
		SetupNodes: []cipher.PubKey{pkSetup},
	}

	rtm := NewRoutingTableManager(logger, routingTable, DefaultRouteKeepalive, DefaultRouteCleanupDuration)
	r := &router{
		log:  logger,
		conf: conf,
		tpm:  tpmLocal,
		rtm:  rtm,
		rfc:  rfc,
	}

	return &mockRouterEnv{
		pkLocal:  pkLocal,
		skLocal:  skLocal,
		pkSetup:  pkSetup,
		pkRemote: pkRemote,

		tpmSetup: tpmSetup,
		tpmLocal: tpmLocal,

		routingTable: routingTable,
		rtm:          rtm,

		r:  r,
		pm: pm,
	}, nil
}

func (env *mockRouterEnv) TearDown() {
	env.r.Close()
	env.pm.Close()
}

func Example_makeMockRouterEnv() {
	env, err := makeMockRouterEnv()
	fmt.Printf("makeMockRouterEnv success: %v\n", err == nil)
	defer env.TearDown()

	fmt.Printf("PKs:\n pkLocal: %v\n pkSetup: %v\n pkRemote: %v\n", env.pkLocal, env.pkSetup, env.pkRemote)

	// Output: makeMockRouterEnv success: true
	// PKs:
	//  pkLocal: 0387935e7035f5bdffb0ec3e4c872bcc4c71d9c3372bf325e71dd5a4879f2939f7
	//  pkSetup: 03c868f201347b705dd7c9282cce5586d61f92aeb0d6edf8c7f1c52b6f447dff94
	//  pkRemote: 02b49835e8b6888bec290026ea81032f4da2c6195d25c74eccf50640bbdf49a3b5
}
