// +build !no_ci

package router

import (
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/internal/noise"
	"github.com/skycoin/skywire/pkg/app"
	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/setup"
)

func ExamplePrintRules() {
	env := &TEnv{}
	_, err := env.RunAsExample(true,
		GenerateDeterministicKeys(),
		AddTransportManagers(),
		AddProcManagerAndRouter(),
		AddSetupHandlersEnv(),
		AddRules(),
		PrintRules(),
	)
	fmt.Printf("env.Run() success: %v\n", err == nil)
	env.PrintTearDown()

	// 	Output: GenerateDeterministicKeys success: true
	// AddTransportManagers success: true
	// ProcManagerAndRouter success: true
	// AddSetupHandersEnv success: true
	// 1.AddRules success: true
	// env.rtm.RangeRules:
	//   1 App: <resp-rid: 3><remote-pk: 027d96fa1bd3108a2a86c7e3ba79bd7b5559e34c8d481a1012b47caa7a5c99870e><remote-port: 3><local-port: 2>
	//   2 Forward: <next-rid: 3><next-tid: 33ecb29a-2c91-4de3-b46f-8a8f9227f3cd>
	// PrintRules success: true
	// env.Run() success: true
	// env.TearDown() success: true
}

// Subtests

// Preparation
// Step 1: Rules and routes
func AddRules() CfgStep {
	return func(env *TEnv) (cfgStepName string, err error) {

		cfgStepName = "1.1.AddRules/appRule"
		env.appRule = routing.AppRule(time.Now().Add(360*time.Second), 3, env.pkRemote, 3, 2)
		_, err = env.rtm.AddRule(env.appRule)
		if err != nil {
			return
		}
		// env.appRule = appRule

		cfgStepName = "1.2.AddRules/fwdRule"
		fwdRule := routing.ForwardRule(time.Now().Add(360*time.Second), 3, uuid.New())
		fwdRouteID, err := env.rtm.AddRule(fwdRule)
		if err != nil {
			return
		}

		cfgStepName = "1.AddRules"
		env.fwdRule = fwdRule
		env.fwdRouteID = fwdRouteID

		return
	}
}

// PrintRules - prints rules from Procmanager.RangeRules
func PrintRules() CfgStep {
	return func(env *TEnv) (cfgStepName string, err error) {
		cfgStepName = "PrintRules"
		fmt.Println("env.rtm.RangeRules:")
		err = env.rtm.RangeRules(
			func(routeID routing.RouteID, rule routing.Rule) (next bool) {
				fmt.Printf("  %v %v\n", routeID, rule)
				next = true
				return
			},
		)
		return
	}
}

// AddLoopData creates setup.LoopData for initiating ConfirmLoop
func AddLoopData() CfgStep {
	return func(env *TEnv) (cfgStepName string, err error) {

		cfgStepName = "LoopData/noise.KKAndSecp256k1"
		ns, err := noise.KKAndSecp256k1(noise.Config{
			LocalPK:   env.pkLocal,
			LocalSK:   env.skLocal,
			RemotePK:  env.pkRemote,
			Initiator: true,
		})
		if err != nil {
			return
		}

		cfgStepName = "LoopData/ns.HandshakeMessage"
		nsRes, err := ns.HandshakeMessage()
		if err != nil {
			return
		}
		// LoopData
		loopData := setup.LoopData{
			RemotePK:     env.pkRemote,
			RemotePort:   3,
			LocalPort:    2,
			RouteID:      env.fwdRouteID,
			NoiseMessage: nsRes,
		}

		cfgStepName = "LoopData"
		env.loopData = loopData
		return
	}
}

func AddAppAndRunProc(workdir, appname string) CfgStep {
	return func(env *TEnv) (testName string, err error) {
		testName = "AddAppAndRunProc"
		appMeta := &app.Meta{AppName: appname, Host: env.pkLocal}

		_, err = env.procMgr.RunProc(env.R, 2, appMeta, &app.ExecConfig{
			HostPK:  env.pkLocal,
			HostSK:  env.skLocal,
			WorkDir: filepath.Join(workdir),
			BinLoc:  filepath.Join(workdir, appname),
		})

		return
	}
}

// CheckRulesAndPorts: stage 2 of confirmLoop
func CheckRulesAndPorts() CfgStep {
	return func(env *TEnv) (cfgStepName string, err error) {
		// Entrails of confirmLoop
		loopMeta := makeLoopMeta(env.R.conf.PubKey, env.loopData)
		env.loopMeta = loopMeta

		cfgStepName = "CheckRulesAndPorts/FindAppRule"
		appRtID, appRule, ok := env.R.rtm.FindAppRule(loopMeta)
		if !ok {
			err = errors.New("AppRule not found")
			return
		}
		env.appRtID = appRtID
		env.appRule = appRule

		cfgStepName = "CheckRulesAndPorts/FindFwdRule"
		foundFwdRule, err := env.R.rtm.FindFwdRule(env.loopData.RouteID)
		if err != nil {
			err = errors.New("FwdRule not found")
			return
		}

		cfgStepName = "CheckRulesAndPorts/ProcOfPort"
		proc, ok := env.procMgr.ProcOfPort(loopMeta.Local.Port)
		if !ok {
			err = errors.New("ProcOfPort not found")
			return
		}
		env.proc = proc

		// Those checks could be a source of panics. That's why they checked here
		cfgStepName, _ = "CheckRulesAndPorts/foundFwdRule.RouteID()", foundFwdRule.RouteID()     // nolint: ineffassign
		cfgStepName, _ = "CheckRulesAndPorts/foundFwdRule.RouteID()", foundFwdRule.TransportID() // nolint: ineffassign
		cfgStepName = "CheckRulesAndPorts"
		return
	}
}

// ConfirmLoopAndFinish: final stage of confirmLoop. ConfirmLoop and then SetRouteID & setRule
func ConfirmLoopAndFinish() CfgStep {
	return func(env *TEnv) (cfgStepName string, err error) {
		cfgStepName = "ConfirmLoopAndFinish/ConfirmLoop"
		_, err = env.proc.ConfirmLoop(env.loopMeta, env.fwdRule.TransportID(), env.fwdRule.RouteID(), env.loopData.NoiseMessage)
		if err != nil {
			return
		}

		cfgStepName = "ConfirmLoopAndFinish/SetRouteID"
		env.appRule.SetRouteID(env.loopData.RouteID)

		cfgStepName = "ConfirmLoopAndFinish/SetRule"
		err = env.R.rtm.SetRule(env.appRtID, env.appRule)
		return
	}
}

// printPorts - prints ports when passed into Procmanager.RangePorts
func printPorts(port uint16, proc *AppProc) (next bool) { // nolint: deadcode, unused
	fmt.Printf("%v %v\n", port, proc)
	next = true
	return
}

// printProcIDS - prints ports when passed into Procmanager.RangeProcIDs
func printProcIDs(pid ProcID, proc *AppProc) (next bool) { // nolint: deadcode, unused
	fmt.Printf("%v %v\n", pid, proc)
	next = true
	return
}

func Test_setupHandlers_confirmLoop(t *testing.T) {
	env := &TEnv{}

	_, err := env.RunAsTest(t,
		GenerateDeterministicKeys(),
		AddTransportManagers(),
		AddProcManagerAndRouter(),
		AddSetupHandlersEnv(),
		AddRules(),
		AddLoopData(),
		AddAppAndRunProc("/bin", "sh"),
		CheckRulesAndPorts(),
		ConfirmLoopAndFinish(),
	)
	require.NoError(t, err)
	require.NoError(t, env.TearDown())
}

func Example_setupHandlers_confirmLoopEntrails() {
	env := &TEnv{}
	_, err := env.RunAsExample(true,
		GenerateKeys(),
		AddTransportManagers(),
		AddProcManagerAndRouter(),
		AddSetupHandlersEnv(),
		// Preparation for confirmLoop
		AddRules(),
		AddLoopData(),
		AddAppAndRunProc("/bin", "sh"), // This step is unstable
		//confirmLoop entrails
		CheckRulesAndPorts(),
		ConfirmLoopAndFinish(),
	)

	fmt.Printf("env.Run success: %v\n", err == nil)
	fmt.Println(err)

	fmt.Printf("env.TearDown() success: %v\n", env.TearDown() == nil)

	// Output: GenerateKeys success: true
	// AddTransportManagers success: true
	// ProcManagerAndRouter success: true
	// AddSetupHandersEnv success: true
	// 1.AddRules success: true
	// LoopData success: true
	// AddAppAndRunProc success: true
	// CheckRulesAndPorts success: true
	// ConfirmLoopAndFinish/ConfirmLoop success: false
	// env.Run success: false
	// chacha20poly1305: message authentication failed
}

func Example_setupHandlers_confirmLoop() {
	env := &TEnv{}
	_, err := env.RunAsExample(true,
		GenerateDeterministicKeys(),
		AddTransportManagers(),
		AddProcManagerAndRouter(),
		AddSetupHandlersEnv(),
		StartSetupTransportManager(),
		// Preparation for confirmLoop
		AddRules(),
		AddLoopData(),
		AddAppAndRunProc("/bin", "sh"),
	)
	fmt.Printf("env.Run success: %v\n", err == nil)

	fmt.Printf("Startup env: success = %v\n", err == nil)

	sh := env.stpHandlers
	res, err := sh.confirmLoop(env.loopData)
	fmt.Printf("confirmLoop(loopData): %v %v\n", res, err)

	fmt.Printf("env.TearDown() success: %v\n", env.TearDown() == nil)

	// Output: Start env: success = true
	// confirmLoop(loopData): [] confirm: chacha20poly1305: message authentication failed
}
