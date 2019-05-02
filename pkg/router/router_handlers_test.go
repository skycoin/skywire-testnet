package router

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/skycoin/skywire/pkg/transport"
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

	pkLocal := env.r.conf.PubKey
	pkSetup := env.r.conf.SetupNodes[0]
	pkRespond := env.mockRouterEnv.pkRespond

	trFalse := transport.NewMockTransport(nil, pkLocal, pkRespond)
	trTrue := transport.NewMockTransport(nil, pkLocal, pkSetup)
	trNotOk := transport.NewMockTransport(nil, pkSetup, pkRespond)

	rh := makeRouterHandlers(env.r, env.pm)
	fmt.Printf("rh.isSetup(trTrue): %v\n", rh.isSetup(trTrue))
	fmt.Printf("rh.isSetup(trFalse): %v\n", rh.isSetup(trFalse))
	fmt.Printf("rh.isSetup(trNotOk): %v\n", rh.isSetup(trNotOk))

	// Output: rh.isSetup(trTrue): true
	// rh.isSetup(trFalse): false
	// rh.isSetup(trNotOk): false
}

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

	handler := func(ProcManager, io.ReadWriter) error {
		err := errors.New("handler error")
		return err
	}

	err = rh.serve(tr, handler)
	fmt.Printf("serve: %v\n", err)
	// Output: serve: handler error
}
