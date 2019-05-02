package router

import (
	"context"
	"errors"
	"fmt"
	"io"
)

func Example_makeRouterHandlers() {

	env, err := makeMockEnv()
	if err != nil {
		fmt.Printf("%v\n", err)
	}

	rh := makeRouterHandlers(env.r, env.pm)

	fmt.Printf("isSetup: %T\n", rh.isSetup)
	fmt.Printf("serve: %T\n", rh.serve)

	// Output: isSetup: func(transport.Transport) bool
	// serve: func(transport.Transport, router.handlerFunc) error

}

func Example_routerHandlers_isSetup() {
	env, err := makeMockEnv()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	defer env.TearDown()

}

// Output:

func Example_routerHandlers_serve() {

	env, err := makeMockEnv()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	defer env.TearDown()
	rh := makeRouterHandlers(env.r, env.pm)

	tr, err := env.r.tpm.CreateTransport(context.TODO(), env.r.conf.SetupNodes[0], "messaging", false)
	if err != nil {
		fmt.Printf("transport: %s", err)
	}

	var handler handlerFunc
	handler = func(ProcManager, io.ReadWriter) error {
		return errors.New("handler error")
	}

	err = rh.serve(tr, handler)
	fmt.Printf("serve: %v\n", err)

	// Output: serve: handler error
}
