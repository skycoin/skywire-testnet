// +build !no_ci

package router

import (
	"context"
	"fmt"
	"time"

	"github.com/skycoin/skywire/pkg/app"
)

/* AlexYu:

This is collection of broken tests.
Commented lines under the code of each Example are additional test-comments

*/

// setup.CreateLoop success: true

func Example_router_handleTransport() {

	shEnv, err := makeSetupHandlersEnv()
	fmt.Printf("makeMockRouterEnv success: %v\n", err == nil)

	go func() {
		err = shEnv.env.r.handleTransport(shEnv.env.pm, shEnv.connInit)
		fmt.Printf("handleTransport success: %v\n", err == nil)
	}()

	time.Sleep(time.Second)

	// Output: makeMockRouterEnv success: true

}

func Example_router_CloseLoop() {
	env, err := makeMockRouterEnv()
	fmt.Printf("makeMockRouterEnv success: %v\n", err == nil)
	// env, _ := makeMockEnv()
	r := env.r

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
	env.StartSetupTpm()
	defer env.TearDown()

	go func() {
		err = env.r.Serve(context.TODO(), env.pm)
		fmt.Printf("router.Serve success: %v\n", err)
	}()

	time.Sleep(time.Second)

	// Output: makeMockRouterEnv success: true

}
