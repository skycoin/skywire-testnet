// +build !no_ci

package router

import (
	"context"
	"fmt"
	"time"

	"github.com/skycoin/skywire/pkg/app"
)

func routerTestEnv() (env *TEnv, err error) {
	env = &TEnv{}
	_, err = env.runSteps(
		ChangeLogLevel("error"),
		GenKeys(),
		AddTransportManagers(),
		AddProcManagerAndRouter(),
		AddSetupHandlersEnv(),
	)
	return
}

func Example_router_handleTransport() {

	env, err := routerTestEnv()
	fmt.Printf("routerTestEnv success: %v\n", err == nil)

	go func() {
		err = env.R.handleTransport(env.procMgr, env.connInit)
		fmt.Printf("handleTransport success: %v\n", err == nil)
	}()

	time.Sleep(time.Second)

	// Output: makeMockRouterEnv success: true
}

func Example_router_CloseLoop() {
	env, err := routerTestEnv()
	fmt.Printf("routerTestEnv success: %v\n", err == nil)

	r := env.R

	lm := app.LoopMeta{
		Local:  app.LoopAddr{PubKey: env.pkLocal, Port: 0},
		Remote: app.LoopAddr{PubKey: env.pkRemote, Port: 0},
	}

	go func() {
		err = r.CloseLoop(lm)
		fmt.Printf("CloseLoop success: %v\n", err)
	}()

	time.Sleep(time.Second)
	// Output: makeMockRouterEnv success: true

}

func Example_router_Serve() {
	env, err := makeMockRouterEnv()
	fmt.Printf("makeMockRouterEnv success: %v\n", err == nil)
	env.runSteps(StartSetupTransportManager())

	go func() {
		err = env.R.Serve(context.TODO(), env.procMgr)
		fmt.Printf("router.Serve success: %v\n", err)
	}()

	time.Sleep(time.Second)

	// Output: makeMockRouterEnv success: true
}
