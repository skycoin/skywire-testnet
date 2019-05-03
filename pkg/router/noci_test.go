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

func Example_router_FindRouteAndSetupLoop() {

	env, err := makeMockRouterEnv()
	fmt.Printf("makeMockRouterEnv success: %v\n", err == nil)

	// prepare noise
	ns, err := noise.KKAndSecp256k1(noise.Config{
		LocalPK:   env.pkLocal,
		LocalSK:   env.skLocal,
		RemotePK:  env.pkRespond,
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
		Remote: app.LoopAddr{PubKey: env.pkRespond, Port: 0},
	}

	// Internals of FindRoutesAndSetupLoop
	fwdRt, rvsRt, err := env.r.fetchBestRoutes(lm.Local.PubKey, lm.Remote.PubKey)
	fmt.Printf("fetchBestRoutes %T %T success: %v\n", fwdRt, rvsRt, err == nil)

	loop := routing.Loop{
		LocalPort:    lm.Local.Port,
		RemotePort:   lm.Remote.Port,
		NoiseMessage: nsMsg,
		Expiry:       time.Now().Add(RouteTTL),
		Forward:      fwdRt,
		Reverse:      rvsRt,
	}

	sProto, tp, err := env.r.setupProto(context.Background())
	fmt.Printf("setupProto %T %T success:  %v\n", sProto, tp, err == nil)

	defer func() { _ = tp.Close() }()

	// Hangs here
	go func() {
		err = setup.CreateLoop(sProto, &loop)
		fmt.Printf("setup.CreateLoop success: %v\n", err)
	}()

	// go func() {
	// 	err = r.FindRoutesAndSetupLoop(lm, msg)
	// 	fmt.Printf("FindRoutesAndSetupLoop error: %v\n", err)
	// }()

	time.Sleep(time.Second)

	// Output: makeMockRouterEnv success: true
	// fetchBestRoutes routing.Route routing.Route success: true
	// setupProto *setup.Protocol *transport.ManagedTransport success:  true
	// setup.CreateLoop success: true
}

func Example_router_handleTransport() {
	env, err := makeMockEnv()
	fmt.Printf("makeMockEnv success: %v\n", err == nil)

	go func() {
		err = env.r.handleTransport(env.pm, env.connInit)
		fmt.Printf("handleTransport success: %v\n", err == nil)
	}()

	time.Sleep(time.Second)

	// Output: makeMockEnv success: true
	// handleTransport success: true
}

func Example_router_CloseLoop() {
	env, err := makeMockRouterEnv()
	fmt.Printf("makeMockRouterEnv success: %v\n", err == nil)
	// env, _ := makeMockEnv()
	r := env.r

	lm := app.LoopMeta{
		Local:  app.LoopAddr{PubKey: env.pkLocal, Port: 0},
		Remote: app.LoopAddr{PubKey: env.pkRespond, Port: 0},
	}

	go func() {
		err = r.CloseLoop(lm)
		fmt.Printf("CloseLoop success: %v\n", err)
	}()

	time.Sleep(time.Second)
	// Output: makeMockRouterEnv success: true
	// CloseLoop success: true
}

func Example_router_Serve() {
	env, err := makeMockEnv()
	if err != nil {
		fmt.Printf("makeMockEnv: %v\n", err)
	}

	// errCh := make(chan error, 1)
	go func() {
		err = env.r.Serve(context.TODO(), env.pm)
		fmt.Printf("router.Serve success: %v\n", err)
	}()

	time.Sleep(time.Second)

	// Output: router.Serve success: true
}
