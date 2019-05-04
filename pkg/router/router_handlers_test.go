package router

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/skycoin/skywire/pkg/transport"
)

func rhTestEnvironment() (*testEnv, error) {
	env := &testEnv{}
	_, err := env.runSteps(
		GenKeys(),
		AddTransportManagers(),
		StartSetupTransportManager(),
		AddProcManagerAndRouter(),
	)
	return env, err
}

func Example_makeRouterHandlers() {
	env, err := rhTestEnvironment()
	fmt.Printf("testEnv creation success: %v\n", err == nil)
	defer env.TearDown()

	rh := makeRouterHandlers(env.R, env.procMgr)
	fmt.Printf("isSetup: %T\n", rh.isSetup)
	fmt.Printf("serve: %T\n", rh.serve)

	// Output: testEnv creation success: true
	// isSetup: func(transport.Transport) bool
	// serve: func(transport.Transport, router.handlerFunc) error
}

func Example_routerHandlers_isSetup() {
	env, err := rhTestEnvironment()
	fmt.Printf("testEnv creation success: %v\n", err == nil)
	defer env.TearDown()

	trFalse := transport.NewMockTransport(nil, env.pkLocal, env.pkRemote)
	trTrue := transport.NewMockTransport(nil, env.pkLocal, env.pkSetup)
	trNotOk := transport.NewMockTransport(nil, env.pkSetup, env.pkRemote)

	rh := makeRouterHandlers(env.R, env.procMgr)
	fmt.Printf("rh.isSetup(trTrue): %v\n", rh.isSetup(trTrue))
	fmt.Printf("rh.isSetup(trFalse): %v\n", rh.isSetup(trFalse))
	fmt.Printf("rh.isSetup(trNotOk): %v\n", rh.isSetup(trNotOk))

	// Output: testEnv creation success: true
	// rh.isSetup(trTrue): true
	// rh.isSetup(trFalse): false
	// rh.isSetup(trNotOk): false
}

func Example_routerHandlers_serve() {
	env := &testEnv{}
	env.runSteps(
		GenKeys(),
		AddTransportManagers(),
		StartSetupTransportManager(),
		AddProcManagerAndRouter(),
	)

	rh := makeRouterHandlers(env.R, env.procMgr)

	tr, err := env.R.tpm.CreateTransport(
		context.TODO(),
		env.R.conf.SetupNodes[0],
		"messaging",
		false)
	if err != nil {
		fmt.Printf("transport: %s", err)
	}

	handler := func(ProcManager, io.ReadWriter) error {
		err := errors.New("handler test message")
		return err
	}

	err = rh.serve(tr, handler)
	fmt.Printf("serve: %v\n", err)
	// Output: serve: handler test message
}
