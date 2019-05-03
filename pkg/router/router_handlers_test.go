package router

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/skycoin/skywire/pkg/transport"
)

func Example_makeRouterHandlers() {

	env, err := makeMockRouterEnv()
	fmt.Printf("makeMockRouterEnv success: %v\n", err == nil)
	// defer env.TearDown()

	rh := makeRouterHandlers(env.r, env.pm)

	fmt.Printf("isSetup: %T\n", rh.isSetup)
	fmt.Printf("serve: %T\n", rh.serve)

	// Output: makeMockRouterEnv success: true
	// isSetup: func(transport.Transport) bool
	// serve: func(transport.Transport, router.handlerFunc) error

}

func Example_routerHandlers_isSetup() {
	env, err := makeMockRouterEnv()
	fmt.Printf("makeMockRouterEnv success: %v\n", err == nil)
	// defer env.TearDown()

	trFalse := transport.NewMockTransport(nil, env.pkLocal, env.pkRemote)
	trTrue := transport.NewMockTransport(nil, env.pkLocal, env.pkSetup)
	trNotOk := transport.NewMockTransport(nil, env.pkSetup, env.pkRemote)

	rh := makeRouterHandlers(env.r, env.pm)
	fmt.Printf("rh.isSetup(trTrue): %v\n", rh.isSetup(trTrue))
	fmt.Printf("rh.isSetup(trFalse): %v\n", rh.isSetup(trFalse))
	fmt.Printf("rh.isSetup(trNotOk): %v\n", rh.isSetup(trNotOk))

	// Output: makeMockRouterEnv success: true
	// rh.isSetup(trTrue): true
	// rh.isSetup(trFalse): false
	// rh.isSetup(trNotOk): false
}

func Example_routerHandlers_serve() {

	env, err := makeMockRouterEnv()
	fmt.Printf("makeMockRouterEnv success: %v\n", err == nil)
	env.StartSetupTpm()
	// defer env.TearDown()

	rh := makeRouterHandlers(env.r, env.pm)

	tr, err := env.r.tpm.CreateTransport(context.TODO(), env.r.conf.SetupNodes[0], "messaging", false)
	if err != nil {
		fmt.Printf("transport: %s", err)
	}

	handler := func(ProcManager, io.ReadWriter) error {
		err := errors.New("handler error")
		return err
	}

	err = rh.serve(tr, handler)
	fmt.Printf("serve: %v\n", err)
	// Output: makeMockRouterEnv success: true
	// serve: handler error

}
