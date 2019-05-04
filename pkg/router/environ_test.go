package router

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"runtime"
	"testing"

	"github.com/skycoin/skycoin/src/util/logging"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/pkg/app"
	"github.com/skycoin/skywire/pkg/cipher"
	routeFinder "github.com/skycoin/skywire/pkg/route-finder/client"
	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/setup"
	"github.com/skycoin/skywire/pkg/transport"
)

type TEnv struct {
	// Keys
	pkLocal  cipher.PubKey
	skLocal  cipher.SecKey
	pkSetup  cipher.PubKey
	skSetup  cipher.SecKey
	pkRemote cipher.PubKey

	// TransportManagers
	tpmLocal *transport.Manager
	tpmSetup *transport.Manager
	rfc      routeFinder.Client

	// routing.Table
	routingTable routing.Table
	rtm          *RoutingTableManager

	// ProcManager and router
	procMgr ProcManager
	R       *router

	// Setup
	stpHandlers setupHandlers

	connResp   net.Conn
	connInit   net.Conn
	sprotoInit *setup.Protocol

	// loopEnv
	appRule    routing.Rule
	appRtID    routing.RouteID
	fwdRule    routing.Rule
	fwdRouteID routing.RouteID

	loopMeta app.LoopMeta
	loopData setup.LoopData

	// Apps
	proc *AppProc
}

// CfgStep defines steps in creating test environments
type CfgStep func(*TEnv) (testName string, err error)

func (env *TEnv) runSteps(steps ...CfgStep) (stepName string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("error in %v", stepName)
		}
	}()
	for _, step := range steps {
		stepName, err = step(env)
	}
	return
}

// GenKeys generates Local, Setup and Remote keys for test environments
func GenKeys() CfgStep {
	return func(env *TEnv) (testName string, err error) {
		testName = "GenKeys"
		env.pkLocal, env.skLocal, _ = cipher.GenerateDeterministicKeyPair([]byte("local")) // nolint: errcheck
		env.pkSetup, env.skSetup, _ = cipher.GenerateDeterministicKeyPair([]byte("setup")) // nolint: errcheck
		env.pkRemote, _, _ = cipher.GenerateDeterministicKeyPair([]byte("remote"))         // nolint: errcheck
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
			LogStore:        logStore}, fLocal)
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
func (env *TEnv) TearDown() error {
	err := env.R.Close()
	if err != nil {
		return err
	}
	return env.procMgr.Close()
}

// TearDown - closes open connections, stops running processes
func TearDown() CfgStep {
	return func(env *TEnv) (testName string, err error) {
		testName = "TearDown"
		err = env.TearDown()
		return
	}
}

func ChangeLogLevel(logLevel string) CfgStep {
	return func(env *TEnv) (testName string, err error) {
		testName = "ChangeLogLevel"
		lvl, err := logging.LevelFromString(logLevel) // nolint: errcheck
		logging.SetLevel(lvl)
		return
	}
}

func makeMockRouterEnv() (env *TEnv, err error) {
	env = &TEnv{}
	_, err = env.runSteps(
		ChangeLogLevel("error"),
		GenKeys(),
		AddTransportManagers(),
		AddProcManagerAndRouter(),
	)
	return
}

func Example_TEnv_runStepsAsExamples() {
	env := &TEnv{}
	_, err := env.runStepsAsExamples(true,
		ChangeLogLevel("info"),
		GenKeys(),
		AddTransportManagers(),
		AddProcManagerAndRouter(),
		TearDown(),
	)
	fmt.Printf("Success: %v\n", err == nil)

	// Output: ChangeLogLevel success: true
	// GenKeys success: true
	// AddTransportManagers success: true
	// ProcManagerAndRouter success: true
	// TearDown success: true
	// Success: true
}

// runStepsAsTests - runs CfgSteps in Test-form
func (env *TEnv) runStepsAsTests(t *testing.T, steps ...CfgStep) (stepName string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("error in %v", stepName)
		}
	}()
	for _, step := range steps {
		t.Run(GetFunctionName(step), func(t *testing.T) {
			stepName, err = step(env)
			require.NoError(t, err, stepName)
		})
	}
	return
}

// GetFunctionName gets a reflected function name
func GetFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

func Test_TEnv_runStepsAsTests(t *testing.T) {
	env := &TEnv{}
	_, err := env.runStepsAsTests(t,
		ChangeLogLevel("info"),
		GenKeys(),
		AddTransportManagers(),
		StartSetupTransportManager(),
		AddProcManagerAndRouter(),
		TearDown(),
	)
	fmt.Printf("Success: %v\n", err == nil)
}

// runStepsAsExamples - runs CfgSteps in Example-form
func (env *TEnv) runStepsAsExamples(verbose bool, steps ...CfgStep) (stepName string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("error in %v", stepName)
		}
	}()
	for _, step := range steps {
		stepName, err = step(env)
		if verbose {
			fmt.Printf("%v success: %v\n", stepName, err == nil)
		}
		if err != nil {
			return
		}
	}
	return
}
