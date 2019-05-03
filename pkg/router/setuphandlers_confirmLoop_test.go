// +build !no_ci

package router

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/skycoin/skywire/internal/noise"
	"github.com/skycoin/skywire/pkg/app"
	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/setup"
)

func Example_setupHandlers_confirmLoopEntrails() {
	envSh, err := makeSetupHandlersEnv()
	fmt.Printf("makeSetupHandlersEnv success: %v\n", err == nil)
	defer envSh.TearDown()
	env := envSh.env

	// Initialiazation

	fmt.Printf("\n* Initialiazation *\n\n")

	// Rules
	fmt.Printf("\nRules:\n")

	appRule := routing.AppRule(time.Now().Add(360*time.Second), 3, env.pkRemote, 3, 2)
	routeID, err := env.rtm.AddRule(appRule)
	fmt.Printf("AddRule %v success:  %v\n", routeID, err == nil)

	fwdRule := routing.ForwardRule(time.Now().Add(360*time.Second), 3, uuid.New())
	fwdRouteID, err := env.rtm.AddRule(fwdRule)
	fmt.Printf("AddRule %v  success: %v\n", fwdRouteID, err == nil)

	// Noise
	fmt.Printf("\nNoise:\n")

	ns, err := noise.KKAndSecp256k1(noise.Config{
		LocalPK:   envSh.env.pkLocal,
		LocalSK:   envSh.env.skLocal,
		RemotePK:  envSh.env.pkRemote,
		Initiator: false,
	})
	fmt.Printf("noise.KKAndSecp256k1 success: %v\n", err == nil)

	nsProcessErr := ns.ProcessMessage([]byte("Blood for the Blood God!"))
	fmt.Printf("ns.ProcessMessage : %v\n", nsProcessErr)

	nsRes, nsHandshakeErr := ns.HandshakeMessage()
	fmt.Printf("ns.Handshake %v bytes: %v\n", len(nsRes), nsHandshakeErr)

	// LoopData
	fmt.Printf("\nLoopData:\n")
	loopData := setup.LoopData{
		RemotePK:     envSh.env.pkRemote,
		RemotePort:   3,
		LocalPort:    2,
		RouteID:      fwdRouteID,
		NoiseMessage: nsRes,
	}

	// Apps

	fmt.Printf("\nApps:\n")
	appMeta := &app.Meta{AppName: "helloworld", Host: envSh.env.pkLocal}

	appProc, err := env.pm.RunProc(env.r, 2, appMeta, &app.ExecConfig{
		HostPK:  env.pkLocal,
		HostSK:  env.skLocal,
		WorkDir: filepath.Join("/tmp/apps"),
		BinLoc:  filepath.Join("/tmp`/apps/helloworld"),
	})
	fmt.Printf("RunProc appProc %v:  %v\n", appProc, err)

	// Entrails of confirmLoop
	fmt.Printf("\n* Entrails of confirmLoop *\n\n")

	loopMeta := makeLoopMeta(envSh.sh.r.conf.PubKey, loopData)
	fmt.Printf("loopMeta: %v\n", loopMeta)

	fmt.Printf("\n** Check rules, ports **\n\n")

	fmt.Println("rtm.RangeRules:")
	envSh.sh.r.rtm.RangeRules(printRules)

	appRtID, appRule, ok := envSh.sh.r.rtm.FindAppRule(loopMeta)
	fmt.Printf("FindAppRule: %v %v %v\n", appRtID, appRule, ok)

	foundFwdRule, err := envSh.sh.r.rtm.FindFwdRule(loopData.RouteID)
	fmt.Printf("FindFwdRule %v: %v\n", foundFwdRule, err)

	proc, ok := envSh.sh.pm.ProcOfPort(loopMeta.Local.Port)
	fmt.Printf("ProcOfPort: %v\n", ok)

	fmt.Println("Ports:")
	envSh.sh.pm.RangePorts(printPorts)

	fmt.Println("ProcIDs:")
	envSh.sh.pm.RangeProcIDs(printProcIDs)

	fmt.Printf("fwdRule.RouteID(): %v\n", foundFwdRule.RouteID())

	fmt.Printf("fwdRule.TransportID(): %v\n", foundFwdRule.TransportID())

	return
	msg, err := proc.ConfirmLoop(loopMeta, fwdRule.TransportID(), fwdRule.RouteID(), loopData.NoiseMessage)
	fmt.Printf("ConfirmLoop: %v %v\n", msg, err)

	envSh.sh.r.log.Infof("Setting reverse route ID %d for rule with ID %d", loopData.RouteID, appRtID)
	appRule.SetRouteID(loopData.RouteID)

	err = envSh.sh.r.rtm.SetRule(appRtID, appRule)
	fmt.Printf("SetRule: %v\n", err)

	fmt.Printf("Confirmed loop with %v\nmsg: %v\n", loopMeta.Remote, msg)

	// Output: makeSetupHandlersEnv success: true

}

func Example_setupHandlers_confirmLoop() {
	// env, err := makeMockRouterEnv()
	// fmt.Printf("makeMockRouterEnv success: %v\n", err == nil)
	// // defer env.TearDown()

	envSh, err := makeSetupHandlersEnv()
	fmt.Printf("makeSetupHandlersEnv success: %v\n", err == nil)
	defer envSh.TearDown()

	loopData := setup.LoopData{
		RemotePK:     envSh.env.pkRemote,
		RemotePort:   3,
		LocalPort:    2,
		RouteID:      routing.RouteID(1),
		NoiseMessage: []byte{},
	}

	res, err := envSh.sh.confirmLoop(loopData)
	fmt.Printf("confirmLoop(loopData): %v %v\n", res, err)

	// Output: makeMockRouterEnv success: true
	// makeSetupHandlersEnv success: true
	// confirmLoop(loopData): [] unknown loop
}
