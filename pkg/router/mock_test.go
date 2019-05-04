package router

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"testing"

	"github.com/skycoin/skycoin/src/util/logging"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/pkg/cipher"
	routeFinder "github.com/skycoin/skywire/pkg/route-finder/client"
	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/transport"
)

type testEnv struct {
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

	SH *mockShEnv
}

// envStep defines steps in creating test environments
type envStep func(*testEnv) (testName string, err error)

func (env *testEnv) runSteps(steps ...envStep) (stepName string, err error) {
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

// envSteps collection

// GenKeys generates Local, Setup and Remote keys for test environments
func GenKeys() envStep {
	return func(env *testEnv) (testName string, err error) {
		testName = "GenKeys"
		env.pkLocal, env.skLocal, _ = cipher.GenerateDeterministicKeyPair([]byte("local")) // nolint: errcheck
		env.pkSetup, env.skSetup, _ = cipher.GenerateDeterministicKeyPair([]byte("setup")) // nolint: errcheck
		env.pkRemote, _, _ = cipher.GenerateDeterministicKeyPair([]byte("remote"))         // nolint: errcheck
		return
	}
}

// AddTransportManagers adds transport.Manager for Local and Setup
func AddTransportManagers() envStep {
	return func(env *testEnv) (testName string, err error) {
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
func StartSetupTransportManager() envStep {
	return func(env *testEnv) (testName string, err error) {
		go env.tpmSetup.Serve(context.TODO()) // nolint: errcheck
		testName = "StartSetupTransportManager"
		return
	}
}

// AddProcManagerAndRouter adds to environment ProcManager and router with RoutingTableManager
func AddProcManagerAndRouter() envStep {
	return func(env *testEnv) (testName string, err error) {
		testName = "ProcManagerAndRouter"

		env.procMgr = NewProcManager(10)
		logger := logging.MustGetLogger("testEnv")
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
func (env *testEnv) TearDown() error {
	err := env.R.Close()
	if err != nil {
		return err
	}
	return env.procMgr.Close()
}

// TearDown - closes open connections, stops running processes
func TearDown() envStep {
	return func(env *testEnv) (testName string, err error) {
		testName = "TearDown"
		err = env.TearDown()
		return
	}
}

func ChangeLogLevel(logLevel string) envStep {
	return func(env *testEnv) (testName string, err error) {
		testName = "ChangeLogLevel"
		lvl, err := logging.LevelFromString(logLevel) // nolint: errcheck
		logging.SetLevel(lvl)
		return
	}
}

func makeMockRouterEnv() (env *testEnv, err error) {
	env = &testEnv{}
	_, err = env.runSteps(
		ChangeLogLevel("error"),
		GenKeys(),
		AddTransportManagers(),
		AddProcManagerAndRouter(),
	)
	return
}

func Example_testEnv_runStepsAsExamples() {
	env := &testEnv{}
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

// runStepsAsTests - runs envSteps in Test-form
func (env *testEnv) runStepsAsTests(t *testing.T, steps ...envStep) (stepName string, err error) {
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

func Test_testEnv_runStepsAsTests(t *testing.T) {
	env := &testEnv{}
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

// runStepsAsExamples - runs envSteps in Example-form
func (env *testEnv) runStepsAsExamples(verbose bool, steps ...envStep) (stepName string, err error) {
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
