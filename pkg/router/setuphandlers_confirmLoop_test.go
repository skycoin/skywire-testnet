// +build !no_ci

package router

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
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
	envSh *mockShEnv
	env   *mockRouterEnv

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
	envSh, err := makeSetupHandlersEnv()
	if err != nil {
		return loopEnv, err
	}

	loopEnv.envSh = envSh
	loopEnv.env = envSh.env

	return loopEnv, nil
}

func Example_makeConfirmLoopEnv() {
	loopEnv, err := makeConfirmLoopEnv()
	fmt.Printf("makeLoopEnv success: %v\n", err == nil)
	defer loopEnv.TearDown()

	// Output: makeLoopEnv success: true
}

func (loopEnv *confirmLoopEnv) TearDown() {
	loopEnv.envSh.TearDown()
}

// Subtests

type subTest func(*confirmLoopEnv) (string, error)

// Preparation
// Step 1: Rules and routes
func AddRules() subTest {
	return func(loopEnv *confirmLoopEnv) (subTestName string, err error) {

		subTestName = "1.1.AddRules/appRule"
		loopEnv.appRule = routing.AppRule(time.Now().Add(360*time.Second), 3, loopEnv.env.pkRemote, 3, 2)
		_, err = loopEnv.env.rtm.AddRule(loopEnv.appRule)
		if err != nil {
			return
		}
		// loopEnv.appRule = appRule

		subTestName = "1.2.AddRules/fwdRule"
		fwdRule := routing.ForwardRule(time.Now().Add(360*time.Second), 3, uuid.New())
		fwdRouteID, err := loopEnv.env.rtm.AddRule(fwdRule)
		if err != nil {
			return
		}

		subTestName = "1.AddRules"
		loopEnv.fwdRule = fwdRule
		loopEnv.fwdRouteID = fwdRouteID

		return
	}
}

// PrintRules - prints rules from Procmanager.RangeRules
func PrintRules() subTest {
	return func(loopEnv *confirmLoopEnv) (subTestName string, err error) {
		subTestName = "PrintRules"
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
func AddLoopData() subTest {
	return func(loopEnv *confirmLoopEnv) (subTestName string, err error) {
		envSh := loopEnv.envSh

		subTestName = "2.1.LoopData/noise.KKAndSecp256k1"
		ns, err := noise.KKAndSecp256k1(noise.Config{
			LocalPK:   envSh.env.pkLocal,
			LocalSK:   envSh.env.skLocal,
			RemotePK:  envSh.env.pkRemote,
			Initiator: true,
		})
		if err != nil {
			return
		}

		subTestName = "2.2.LoopData/ns.HandshakeMessage"
		nsRes, err := ns.HandshakeMessage()
		if err != nil {
			return
		}
		// LoopData
		loopData := setup.LoopData{
			RemotePK:     envSh.env.pkRemote,
			RemotePort:   3,
			LocalPort:    2,
			RouteID:      loopEnv.fwdRouteID,
			NoiseMessage: nsRes,
		}

		subTestName = "2.LoopData"
		loopEnv.loopData = loopData
		return
	}
}

// Step 3: Apps
func AddApps(workdir string) subTest {
	return func(loopEnv *confirmLoopEnv) (string, error) {
		appMeta := &app.Meta{AppName: "helloworld", Host: loopEnv.envSh.env.pkLocal}

		_, err := loopEnv.env.pm.RunProc(loopEnv.env.r, 2, appMeta, &app.ExecConfig{
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
func CheckRulesAndPorts() subTest {
	return func(loopEnv *confirmLoopEnv) (subTestName string, err error) {
		envSh := loopEnv.envSh
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("error in %v", subTestName)
			}
		}()

		// Entrails of confirmLoop
		loopMeta := makeLoopMeta(envSh.sh.r.conf.PubKey, loopEnv.loopData)
		loopEnv.loopMeta = loopMeta

		subTestName = "4.1.CheckRulesAndPorts/FindAppRule"
		appRtID, appRule, ok := envSh.sh.r.rtm.FindAppRule(loopMeta)
		if !ok {
			err = errors.New("AppRule not found")
			return
		}
		loopEnv.appRtID = appRtID
		loopEnv.appRule = appRule

		subTestName = "4.2.CheckRulesAndPorts/FindFwdRule"
		foundFwdRule, err := envSh.sh.r.rtm.FindFwdRule(loopEnv.loopData.RouteID)
		if err != nil {
			err = errors.New("FwdRule not found")
			return
		}

		subTestName = "4.3.CheckRulesAndPorts/ProcOfPort"
		proc, ok := envSh.sh.pm.ProcOfPort(loopMeta.Local.Port)
		if !ok {
			err = errors.New("ProcOfPort not found")
			return
		}
		loopEnv.proc = proc

		subTestName, _ = "4.4.CheckRulesAndPorts/foundFwdRule.RouteID()", foundFwdRule.RouteID()
		subTestName, _ = "4.5.CheckRulesAndPorts/foundFwdRule.RouteID()", foundFwdRule.TransportID()
		subTestName = "4.CheckRulesAndPorts"
		return
	}
}

// Step 5: ConfirmLoop and then SetRouteID & setRule
func ConfirmLoopAndFinish() subTest {
	return func(loopEnv *confirmLoopEnv) (subTestName string, err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("error in %v", subTestName)
			}
		}()

		subTestName = "5.1.ConfirmLoopAndFinish/ConfirmLoop"
		_, err = loopEnv.proc.ConfirmLoop(loopEnv.loopMeta, loopEnv.fwdRule.TransportID(), loopEnv.fwdRule.RouteID(), loopEnv.loopData.NoiseMessage)
		if err != nil {
			return
		}

		subTestName = "5.2.ConfirmLoopAndFinish/SetRouteID"
		loopEnv.appRule.SetRouteID(loopEnv.loopData.RouteID)

		subTestName = "5.3.ConfirmLoopAndFinish/SetRule"
		err = loopEnv.envSh.sh.r.rtm.SetRule(loopEnv.appRtID, loopEnv.appRule)

		return
	}
}

func GetFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

func (loopEnv *confirmLoopEnv) RunSubTests(t *testing.T, subtests ...subTest) {
	for _, st := range subtests {
		t.Run(GetFunctionName(st), func(t *testing.T) {
			subStep, err := st(loopEnv)
			require.NoError(t, err, subStep)
		})
	}
}

func (loopEnv *confirmLoopEnv) RunSubExamples(verbose bool, subtests ...subTest) (err error) {
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

	sh := loopEnv.envSh.sh
	res, err := sh.confirmLoop(loopEnv.loopData)
	fmt.Printf("confirmLoop(loopData): %v %v\n", res, err)

	// Output: Start loopEnv: success = true
	// confirmLoop(loopData): [] confirm: chacha20poly1305: message authentication failed
}
