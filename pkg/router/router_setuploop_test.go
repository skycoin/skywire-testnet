// +build !no_ci

package router

import (
	"context"
	"fmt"
	"time"

	"github.com/skycoin/skywire/internal/noise"
	"github.com/skycoin/skywire/pkg/app"
	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/setup"
)

func Example_router_FindRouteAndSetupLoopEntrails() {

	env := &TEnv{}
	_, err := env.runSteps(
		ChangeLogLevel("error"),
		GenKeys(),
		AddTransportManagers(),
		AddProcManagerAndRouter(),
		StartSetupTransportManager(),
	)
	fmt.Printf("TEnv started: %v\n", err == nil)

	// prepare noise
	ns, err := noise.KKAndSecp256k1(noise.Config{
		LocalPK:   env.pkLocal,
		LocalSK:   env.skLocal,
		RemotePK:  env.pkRemote,
		Initiator: true,
	})
	if err != nil {
		fmt.Printf("noise.KKAndSecp256k1 error: %v\n", err)
	}
	nsMsg, err := ns.HandshakeMessage()
	if err != nil {
		fmt.Printf("ns.HandshakeMessage error: %v\n", err)
	}

	lm := app.LoopMeta{
		Local:  app.LoopAddr{PubKey: env.pkLocal, Port: 0},
		Remote: app.LoopAddr{PubKey: env.pkRemote, Port: 0},
	}

	// Internals of FindRoutesAndSetupLoop
	fwdRt, rvsRt, err := env.R.fetchBestRoutes(lm.Local.PubKey, lm.Remote.PubKey)
	fmt.Printf("fetchBestRoutes %T %T success: %v\n", fwdRt, rvsRt, err == nil)

	loop := routing.Loop{
		LocalPort:    lm.Local.Port,
		RemotePort:   lm.Remote.Port,
		NoiseMessage: nsMsg,
		Expiry:       time.Now().Add(RouteTTL),
		Forward:      fwdRt,
		Reverse:      rvsRt,
	}

	sProto, tp, err := env.R.setupProto(context.Background())
	fmt.Printf("setupProto %T %T success:  %v\n", sProto, tp, err == nil)

	// defer func() { _ = tp.Close() }()

	// Hangs here
	go func() {
		err = setup.CreateLoop(sProto, &loop)
		fmt.Printf("setup.CreateLoop success: %v\n", err)
	}()

	go func() {
		err = env.R.FindRoutesAndSetupLoop(lm, nsMsg)
		fmt.Printf("FindRoutesAndSetupLoop error: %v\n", err)
	}()

	time.Sleep(time.Second)

	// Output: TEnv started: true
	// fetchBestRoutes routing.Route routing.Route success: true
	// setupProto *setup.Protocol *transport.ManagedTransport success:  true

}
