package router

import (
	"fmt"
	"io/ioutil"
	"net"
	"reflect"
	"runtime"
	"strings"
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

// TEnv - test environment. Encapsulates objects shared by `CfgStep`s during tests
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
	connResp    net.Conn
	connInit    net.Conn
	sprotoInit  *setup.Protocol
	// loops
	appRule    routing.Rule
	appRtID    routing.RouteID
	fwdRule    routing.Rule
	fwdRouteID routing.RouteID
	loopMeta   app.LoopMeta
	loopData   setup.LoopData
	// Apps
	proc *AppProc
}

// CfgStep defines steps in creating test environments
type CfgStep func(*TEnv) (testName string, err error)

func (env *TEnv) Run(steps ...CfgStep) (stepName string, err error) {
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

// TearDown - closes open connections, stops running processes
// TODO: close connections, check other resources
func (env *TEnv) TearDown() error {

	if env.tpmSetup != nil {
		if err := env.tpmSetup.Close(); err != nil {
			return err
		}
	}

	if err := env.R.Close(); err != nil {
		return err
	}
	if err := env.procMgr.Close(); err != nil {
		return err
	}

	// if env.connInit != nil {
	// 	_ = env.connInit.Close() // nolint: errcheck
	// }
	// if env.connResp != nil {
	// 	_ = env.connResp.Close() // nolint: errcheck
	// }
	return nil
}

// PrintTearDown - used in Examples on finish
func (env *TEnv) PrintTearDown() {
	fmt.Printf("env.TearDown() success: %v\n", env.TearDown() == nil)
}

// NoErrorTearDown - used in Tests on finish
func (env *TEnv) NoErrorTearDown(t *testing.T) {
	require.NoError(t, env.TearDown())
}

// DisableLogging sets output of logrus to /dev/null
func DisableLogging() CfgStep {
	return func(env *TEnv) (testName string, err error) {
		testName = "DisableLogging"
		logging.SetOutputTo(ioutil.Discard)
		return
	}
}

// ChangeLogLevel sets logging level
// Warning: it looks like this function has data races - don't use in CI-tests
func ChangeLogLevel(logLevel string) CfgStep {
	return func(env *TEnv) (testName string, err error) {
		testName = "ChangeLogLevel"
		lvl, err := logging.LevelFromString(logLevel) // nolint: errcheck
		logging.SetLevel(lvl)
		return
	}
}

func ExampleTEnv_RunAsExample() {
	env := &TEnv{}
	_, err := env.RunAsExample(true,
		DisableLogging(),
		GenerateDeterministicKeys(),
		AddTransportManagers(),
		AddProcManagerAndRouter(),
		TearDown(),
	)
	fmt.Printf("Success: %v\n", err == nil)

	// Output: DisableLogging success: true
	// GenerateDeterministicKeys success: true
	// AddTransportManagers success: true
	// ProcManagerAndRouter success: true
	// TearDown success: true
	// Success: true
}

// RunAsTest - runs CfgSteps in Test-form
func (env *TEnv) RunAsTest(t *testing.T, steps ...CfgStep) (stepName string, err error) {
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
	funcName := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
	parts := strings.Split(funcName, "/")
	return parts[len(parts)-1]
}

func ExampleGetFunctionName() {
	fmt.Println(GetFunctionName(GetFunctionName))
	// Output: router.GetFunctionName
}

func TestTEnv_RunAsTest(t *testing.T) {
	env := &TEnv{}
	_, err := env.RunAsTest(t,
		GenerateDeterministicKeys(),
		AddTransportManagers(),
		StartSetupTransportManager(),
		AddProcManagerAndRouter(),
		TearDown(),
	)
	fmt.Printf("Success: %v\n", err == nil)
}

// RunAsExample - runs CfgSteps in Example-form
func (env *TEnv) RunAsExample(verbose bool, steps ...CfgStep) (stepName string, err error) {
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
