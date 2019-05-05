package router

import (
	"context"

	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/cipher"
	routeFinder "github.com/skycoin/skywire/pkg/route-finder/client"
	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/transport"
)

// GenerateDeterministicKeys generates  deterministic Local, Setup and Remote keys for test environments
func GenerateDeterministicKeys() CfgStep {
	return func(env *TEnv) (testName string, err error) {
		testName = "GenerateDeterministicKeys"
		env.pkLocal, env.skLocal, _ = cipher.GenerateDeterministicKeyPair([]byte("local")) // nolint: errcheck
		env.pkSetup, env.skSetup, _ = cipher.GenerateDeterministicKeyPair([]byte("setup")) // nolint: errcheck
		env.pkRemote, _, _ = cipher.GenerateDeterministicKeyPair([]byte("remote"))         // nolint: errcheck
		return
	}
}

// GenerateKeys generates Local, Setup and Remote keys for test environments
func GenerateKeys() CfgStep {
	return func(env *TEnv) (testName string, err error) {
		testName = "GenerateKeys"
		env.pkLocal, env.skLocal = cipher.GenerateKeyPair()
		env.pkSetup, env.skSetup = cipher.GenerateKeyPair()
		env.pkRemote, _ = cipher.GenerateKeyPair()
		return
	}
}

// AddTransportManagers adds transport.Manager for Local and Setup
func AddTransportManagers() CfgStep {
	return func(env *TEnv) (testName string, err error) {
		testName = "AddTransportManagers"

		dClient := transport.NewDiscoveryMock()
		env.rfc = routeFinder.NewMock()
		logStore := transport.InMemoryTransportLogStore()
		// TransportFactories
		fLocal, fSetup := transport.NewMockFactoryPair(env.pkLocal, env.pkSetup)
		fLocal.SetType("messaging")
		// TransportManagers
		env.tpmLocal, err = transport.NewManager(&transport.ManagerConfig{
			PubKey:          env.pkLocal,
			SecKey:          env.skLocal,
			DiscoveryClient: dClient,
			LogStore:        logStore}, fLocal,
		)
		if err != nil {
			return
		}

		env.tpmSetup, err = transport.NewManager(&transport.ManagerConfig{
			PubKey:          env.pkSetup,
			SecKey:          env.skSetup,
			DiscoveryClient: dClient,
			LogStore:        logStore}, fSetup)

		return
	}
}

// StartSetupTransportManager - starts TransportManager of Setup
func StartSetupTransportManager() CfgStep {
	return func(env *TEnv) (testName string, err error) {
		go env.tpmSetup.Serve(context.TODO()) // nolint: errcheck
		testName = "StartSetupTransportManager"
		return
	}
}

// AddProcManagerAndRouter adds to environment ProcManager and router with RoutingTableManager
func AddProcManagerAndRouter() CfgStep {
	return func(env *TEnv) (testName string, err error) {
		testName = "ProcManagerAndRouter"

		env.procMgr = NewProcManager(10)
		logger := logging.MustGetLogger("TEnv")
		conf := &Config{
			PubKey:     env.pkLocal,
			SecKey:     env.skLocal,
			SetupNodes: []cipher.PubKey{env.pkSetup},
		}
		env.routingTable = routing.InMemoryRoutingTable()
		env.rtm = NewRoutingTableManager(logger, env.routingTable, DefaultRouteKeepalive, DefaultRouteCleanupDuration)
		env.R = &router{
			log:  logger,
			conf: conf,
			tpm:  env.tpmLocal,
			rtm:  env.rtm,
			rfc:  env.rfc,
		}
		return
	}
}

// TearDown - closes open connections, stops running processes
func TearDown() CfgStep {
	return func(env *TEnv) (testName string, err error) {
		testName = "TearDown"
		err = env.TearDown()
		return
	}
}
