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

func makeConfirmLoopEnv() (*TEnv, error) {
	env := &TEnv{}
	_, err := env.runSteps(
		GenKeys(),
		AddTransportManagers(),
		AddProcManagerAndRouter(),
		AddSetupHandlersEnv(),
	)
	return env, err
}

func Example_makeConfirmLoopEnv() {
	env, err := makeConfirmLoopEnv()
	fmt.Printf("makeLoopEnv success: %v\n", err == nil)
	defer env.TearDown()

	// Output: makeLoopEnv success: true
}

// Subtests

// Preparation
// Step 1: Rules and routes
func AddRules() CfgStep {
	return func(env *TEnv) (CfgStepName string, err error) {

		CfgStepName = "1.1.AddRules/appRule"
		env.appRule = routing.AppRule(time.Now().Add(360*time.Second), 3, env.pkRemote, 3, 2)
		_, err = env.rtm.AddRule(env.appRule)
		if err != nil {
			return
		}
		// env.appRule = appRule

		CfgStepName = "1.2.AddRules/fwdRule"
		fwdRule := routing.ForwardRule(time.Now().Add(360*time.Second), 3, uuid.New())
		fwdRouteID, err := env.rtm.AddRule(fwdRule)
		if err != nil {
			return
		}

		CfgStepName = "1.AddRules"
		env.fwdRule = fwdRule
		env.fwdRouteID = fwdRouteID

		return
	}
}

// PrintRules - prints rules from Procmanager.RangeRules
func PrintRules() CfgStep {
	return func(env *TEnv) (CfgStepName string, err error) {
		CfgStepName = "PrintRules"
		err = env.rtm.RangeRules(
			func(routeID routing.RouteID, rule routing.Rule) (next bool) {
				fmt.Printf("%v %v\n", routeID, rule)
				next = true
				return
			},
		)
		return
	}
}

// Step 2: LoopData
func AddLoopData() CfgStep {
	return func(env *TEnv) (CfgStepName string, err error) {

		CfgStepName = "2.1.LoopData/noise.KKAndSecp256k1"
		ns, err := noise.KKAndSecp256k1(noise.Config{
			LocalPK:   env.pkLocal,
			LocalSK:   env.skLocal,
			RemotePK:  env.pkRemote,
			Initiator: true,
		})
		if err != nil {
			return
		}

		CfgStepName = "2.2.LoopData/ns.HandshakeMessage"
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

		CfgStepName = "2.LoopData"
		env.loopData = loopData
		return
	}
}

// Step 3: Apps
func AddApps(workdir string) CfgStep {
	return func(env *TEnv) (string, error) {
		appMeta := &app.Meta{AppName: "helloworld", Host: env.pkLocal}

		_, err := env.procMgr.RunProc(env.R, 2, appMeta, &app.ExecConfig{
			HostPK:  env.pkLocal,
			HostSK:  env.skLocal,
			WorkDir: filepath.Join(workdir),
			BinLoc:  filepath.Join(workdir, "helloworld"),
		})

		return "3.Apps", err
	}
}

// Stage 2: Entrails of confirmLoop
// Step 4: Check LoopData, routes
func CheckRulesAndPorts() CfgStep {
	return func(env *TEnv) (CfgStepName string, err error) {

		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("error in %v", CfgStepName)
			}
		}()

		// Entrails of confirmLoop
		loopMeta := makeLoopMeta(env.R.conf.PubKey, env.loopData)
		env.loopMeta = loopMeta

		CfgStepName = "4.1.CheckRulesAndPorts/FindAppRule"
		appRtID, appRule, ok := env.R.rtm.FindAppRule(loopMeta)
		if !ok {
			err = errors.New("AppRule not found")
			return
		}
		env.appRtID = appRtID
		env.appRule = appRule

		CfgStepName = "4.2.CheckRulesAndPorts/FindFwdRule"
		foundFwdRule, err := env.R.rtm.FindFwdRule(env.loopData.RouteID)
		if err != nil {
			err = errors.New("FwdRule not found")
			return
		}

		CfgStepName = "4.3.CheckRulesAndPorts/ProcOfPort"
		proc, ok := env.procMgr.ProcOfPort(loopMeta.Local.Port)
		if !ok {
			err = errors.New("ProcOfPort not found")
			return
		}
		env.proc = proc

		CfgStepName, _ = "4.4.CheckRulesAndPorts/foundFwdRule.RouteID()", foundFwdRule.RouteID()
		CfgStepName, _ = "4.5.CheckRulesAndPorts/foundFwdRule.RouteID()", foundFwdRule.TransportID()
		CfgStepName = "4.CheckRulesAndPorts"
		return
	}
}

// Step 5: ConfirmLoop and then SetRouteID & setRule
func ConfirmLoopAndFinish() CfgStep {
	return func(env *TEnv) (CfgStepName string, err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("error in %v", CfgStepName)
			}
		}()

		CfgStepName = "5.1.ConfirmLoopAndFinish/ConfirmLoop"
		_, err = env.proc.ConfirmLoop(env.loopMeta, env.fwdRule.TransportID(), env.fwdRule.RouteID(), env.loopData.NoiseMessage)
		if err != nil {
			return
		}

		CfgStepName = "5.2.ConfirmLoopAndFinish/SetRouteID"
		env.appRule.SetRouteID(env.loopData.RouteID)

		CfgStepName = "5.3.ConfirmLoopAndFinish/SetRule"
		err = env.R.rtm.SetRule(env.appRtID, env.appRule)

		return
	}
}

// printPorts - prints ports when passed into Procmanager.RangePorts
func printPorts(port uint16, proc *AppProc) (next bool) {
	fmt.Printf("%v %v\n", port, proc)
	next = true
	return
}

// printProcIDS - prints ports when passed into Procmanager.RangeProcIDs
func printProcIDs(pid ProcID, proc *AppProc) (next bool) {
	fmt.Printf("%v %v\n", pid, proc)
	next = true
	return
}

func Test_setupHandlers_confirmLoop(t *testing.T) {
	env, err := makeConfirmLoopEnv()
	defer env.TearDown()
	require.NoError(t, err)

	env.runStepsAsTests(t,
		AddRules(),
		AddLoopData(),
		AddApps("/tmp/apps"),
		CheckRulesAndPorts(),
		ConfirmLoopAndFinish(),
	)
}

func Example_setupHandlers_confirmLoopEntrails() {
	env, err := makeConfirmLoopEnv()
	fmt.Printf("Start env: success = %v\n", err == nil)
	defer env.TearDown()

	_, err = env.runStepsAsExamples(true,
		AddRules(),
		AddLoopData(),
		AddApps("/tmp/apps"),
		CheckRulesAndPorts(),
		ConfirmLoopAndFinish(),
	)

	fmt.Printf("Finish success: %v\n", err)
	fmt.Println(err)

	// Output: Start env: success = true
	// 1.AddRules success: true
	// 2.LoopData success: true
	// 3.Apps success: true
	// 4.CheckRulesAndPorts success: true
	// 5.1.ConfirmLoopAndFinish/ConfirmLoop success: false
	// Finish success: chacha20poly1305: message authentication failed
	// chacha20poly1305: message authentication failed
}

// All together
func Example_setupHandlers_confirmLoop() {

	env, err := makeConfirmLoopEnv()
	fmt.Printf("Create env: success = %v\n", err == nil)
	defer env.TearDown()

	_, err = env.runStepsAsExamples(false,
		AddRules(),
		AddLoopData(),
		AddApps("/tmp/apps"),
	)
	fmt.Printf("Startup env: success = %v\n", err == nil)

	sh := env.stpHandlers
	res, err := sh.confirmLoop(env.loopData)
	fmt.Printf("confirmLoop(loopData): %v %v\n", res, err)

	// Output: Start env: success = true
	// confirmLoop(loopData): [] confirm: chacha20poly1305: message authentication failed
}
