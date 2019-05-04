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

type confirmLoopEnv struct {
	env *testEnv

	appRule    routing.Rule
	appRtID    routing.RouteID
	fwdRule    routing.Rule
	fwdRouteID routing.RouteID

	loopMeta app.LoopMeta
	loopData setup.LoopData

	proc *AppProc
}

func makeConfirmLoopEnv() (*confirmLoopEnv, error) {
	loopEnv := &confirmLoopEnv{}
	env := &testEnv{}
	_, err := env.runSteps(
		GenKeys(),
		AddTransportManagers(),
		AddProcManagerAndRouter(),
		AddSetupHandlersEnv(),
	)

	if err != nil {
		return loopEnv, err
	}

	loopEnv.env = env

	return loopEnv, nil
}

func Example_makeConfirmLoopEnv() {
	loopEnv, err := makeConfirmLoopEnv()
	fmt.Printf("makeLoopEnv success: %v\n", err == nil)
	defer loopEnv.TearDown()

	// Output: makeLoopEnv success: true
}

func (loopEnv *confirmLoopEnv) TearDown() {
	loopEnv.env.TearDown()
}

// Subtests

type loopSubTest func(*confirmLoopEnv) (string, error)

// Preparation
// Step 1: Rules and routes
func AddRules() loopSubTest {
	return func(loopEnv *confirmLoopEnv) (loopSubTestName string, err error) {

		loopSubTestName = "1.1.AddRules/appRule"
		loopEnv.appRule = routing.AppRule(time.Now().Add(360*time.Second), 3, loopEnv.env.pkRemote, 3, 2)
		_, err = loopEnv.env.rtm.AddRule(loopEnv.appRule)
		if err != nil {
			return
		}
		// loopEnv.appRule = appRule

		loopSubTestName = "1.2.AddRules/fwdRule"
		fwdRule := routing.ForwardRule(time.Now().Add(360*time.Second), 3, uuid.New())
		fwdRouteID, err := loopEnv.env.rtm.AddRule(fwdRule)
		if err != nil {
			return
		}

		loopSubTestName = "1.AddRules"
		loopEnv.fwdRule = fwdRule
		loopEnv.fwdRouteID = fwdRouteID

		return
	}
}

// PrintRules - prints rules from Procmanager.RangeRules
func PrintRules() loopSubTest {
	return func(loopEnv *confirmLoopEnv) (loopSubTestName string, err error) {
		loopSubTestName = "PrintRules"
		err = loopEnv.env.rtm.RangeRules(
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
func AddLoopData() loopSubTest {
	return func(loopEnv *confirmLoopEnv) (loopSubTestName string, err error) {

		loopSubTestName = "2.1.LoopData/noise.KKAndSecp256k1"
		ns, err := noise.KKAndSecp256k1(noise.Config{
			LocalPK:   loopEnv.env.pkLocal,
			LocalSK:   loopEnv.env.skLocal,
			RemotePK:  loopEnv.env.pkRemote,
			Initiator: true,
		})
		if err != nil {
			return
		}

		loopSubTestName = "2.2.LoopData/ns.HandshakeMessage"
		nsRes, err := ns.HandshakeMessage()
		if err != nil {
			return
		}
		// LoopData
		loopData := setup.LoopData{
			RemotePK:     loopEnv.env.pkRemote,
			RemotePort:   3,
			LocalPort:    2,
			RouteID:      loopEnv.fwdRouteID,
			NoiseMessage: nsRes,
		}

		loopSubTestName = "2.LoopData"
		loopEnv.loopData = loopData
		return
	}
}

// Step 3: Apps
func AddApps(workdir string) loopSubTest {
	return func(loopEnv *confirmLoopEnv) (string, error) {
		appMeta := &app.Meta{AppName: "helloworld", Host: loopEnv.env.pkLocal}

		_, err := loopEnv.env.procMgr.RunProc(loopEnv.env.R, 2, appMeta, &app.ExecConfig{
			HostPK:  loopEnv.env.pkLocal,
			HostSK:  loopEnv.env.skLocal,
			WorkDir: filepath.Join(workdir),
			BinLoc:  filepath.Join(workdir, "helloworld"),
		})

		return "3.Apps", err
	}
}

// Stage 2: Entrails of confirmLoop
// Step 4: Check LoopData, routes
func CheckRulesAndPorts() loopSubTest {
	return func(loopEnv *confirmLoopEnv) (loopSubTestName string, err error) {
		envSh := loopEnv.env.SH
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("error in %v", loopSubTestName)
			}
		}()

		// Entrails of confirmLoop
		loopMeta := makeLoopMeta(loopEnv.env.R.conf.PubKey, loopEnv.loopData)
		loopEnv.loopMeta = loopMeta

		loopSubTestName = "4.1.CheckRulesAndPorts/FindAppRule"
		appRtID, appRule, ok := loopEnv.env.R.rtm.FindAppRule(loopMeta)
		if !ok {
			err = errors.New("AppRule not found")
			return
		}
		loopEnv.appRtID = appRtID
		loopEnv.appRule = appRule

		loopSubTestName = "4.2.CheckRulesAndPorts/FindFwdRule"
		foundFwdRule, err := loopEnv.env.R.rtm.FindFwdRule(loopEnv.loopData.RouteID)
		if err != nil {
			err = errors.New("FwdRule not found")
			return
		}

		loopSubTestName = "4.3.CheckRulesAndPorts/ProcOfPort"
		proc, ok := envSh.sh.pm.ProcOfPort(loopMeta.Local.Port)
		if !ok {
			err = errors.New("ProcOfPort not found")
			return
		}
		loopEnv.proc = proc

		loopSubTestName, _ = "4.4.CheckRulesAndPorts/foundFwdRule.RouteID()", foundFwdRule.RouteID()
		loopSubTestName, _ = "4.5.CheckRulesAndPorts/foundFwdRule.RouteID()", foundFwdRule.TransportID()
		loopSubTestName = "4.CheckRulesAndPorts"
		return
	}
}

// Step 5: ConfirmLoop and then SetRouteID & setRule
func ConfirmLoopAndFinish() loopSubTest {
	return func(loopEnv *confirmLoopEnv) (loopSubTestName string, err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("error in %v", loopSubTestName)
			}
		}()

		loopSubTestName = "5.1.ConfirmLoopAndFinish/ConfirmLoop"
		_, err = loopEnv.proc.ConfirmLoop(loopEnv.loopMeta, loopEnv.fwdRule.TransportID(), loopEnv.fwdRule.RouteID(), loopEnv.loopData.NoiseMessage)
		if err != nil {
			return
		}

		loopSubTestName = "5.2.ConfirmLoopAndFinish/SetRouteID"
		loopEnv.appRule.SetRouteID(loopEnv.loopData.RouteID)

		loopSubTestName = "5.3.ConfirmLoopAndFinish/SetRule"
		err = loopEnv.env.R.rtm.SetRule(loopEnv.appRtID, loopEnv.appRule)

		return
	}
}

func (loopEnv *confirmLoopEnv) RunSubTests(t *testing.T, subtests ...loopSubTest) {
	for _, st := range subtests {
		t.Run(GetFunctionName(st), func(t *testing.T) {
			subStep, err := st(loopEnv)
			require.NoError(t, err, subStep)
		})
	}
}

func (loopEnv *confirmLoopEnv) RunSubExamples(verbose bool, subtests ...loopSubTest) (err error) {
	for _, st := range subtests {
		stepName, err := st(loopEnv)
		if verbose {
			fmt.Printf("%v success: %v\n", stepName, err == nil)
		}
		if err != nil {
			return err
		}
	}
	return
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
	loopEnv, err := makeConfirmLoopEnv()
	defer loopEnv.TearDown()
	require.NoError(t, err)

	loopEnv.RunSubTests(t,
		AddRules(),
		AddLoopData(),
		AddApps("/tmp/apps"),
		CheckRulesAndPorts(),
		ConfirmLoopAndFinish(),
	)
}

func Example_setupHandlers_confirmLoopEntrails() {
	loopEnv, err := makeConfirmLoopEnv()
	fmt.Printf("Start loopEnv: success = %v\n", err == nil)
	defer loopEnv.TearDown()

	err = loopEnv.RunSubExamples(true,
		AddRules(),
		AddLoopData(),
		AddApps("/tmp/apps"),
		CheckRulesAndPorts(),
		ConfirmLoopAndFinish(),
	)

	fmt.Printf("Finish success: %v\n", err)
	fmt.Println(err)

	// Output: Start loopEnv: success = true
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

	loopEnv, err := makeConfirmLoopEnv()
	fmt.Printf("Create loopEnv: success = %v\n", err == nil)
	defer loopEnv.TearDown()

	err = loopEnv.RunSubExamples(false,
		AddRules(),
		AddLoopData(),
		AddApps("/tmp/apps"),
	)
	fmt.Printf("Startup loopEnv: success = %v\n", err == nil)

	sh := loopEnv.env.SH.stpHandlers
	res, err := sh.confirmLoop(loopEnv.loopData)
	fmt.Printf("confirmLoop(loopData): %v %v\n", res, err)

	// Output: Start loopEnv: success = true
	// confirmLoop(loopData): [] confirm: chacha20poly1305: message authentication failed
}
