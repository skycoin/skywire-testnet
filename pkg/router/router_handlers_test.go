package router

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/skycoin/skywire/pkg/transport"
)

func Example_makeRouterHandlers() {
	env := &TEnv{}
	_, err := env.Run(
		GenerateDeterministicKeys(),
		AddTransportManagers(),
		StartSetupTransportManager(),
		AddProcManagerAndRouter(),
	)
	fmt.Printf("env.Run success: %v\n", err == nil)

	rh := makeRouterHandlers(env.R, env.procMgr)
	fmt.Printf("isSetup signature: %T\n", rh.isSetup)
	fmt.Printf("serve signature: %T\n", rh.serve)

	env.PrintTearDown()

	// Output: env.Run success: true
	// isSetup signature: func(transport.Transport) bool
	// serve signature: func(transport.Transport, router.handlerFunc) error
	// env.TearDown() success: true
}

func Example_routerHandlers_isSetup() {
	env := &TEnv{}
	_, err := env.Run(
		GenerateDeterministicKeys(),
		AddTransportManagers(),
		StartSetupTransportManager(),
		AddProcManagerAndRouter(),
	)
	fmt.Printf("env.Run success: %v\n", err == nil)

	trFalse := transport.NewMockTransport(nil, env.pkLocal, env.pkRemote)
	trTrue := transport.NewMockTransport(nil, env.pkLocal, env.pkSetup)
	trNotOk := transport.NewMockTransport(nil, env.pkSetup, env.pkRemote)

	rh := makeRouterHandlers(env.R, env.procMgr)
	fmt.Printf("rh.isSetup(trTrue): %v\n", rh.isSetup(trTrue))
	fmt.Printf("rh.isSetup(trFalse): %v\n", rh.isSetup(trFalse))
	fmt.Printf("rh.isSetup(trNotOk): %v\n", rh.isSetup(trNotOk))

	env.PrintTearDown()

	// Output: env.Run success: true
	// rh.isSetup(trTrue): true
	// rh.isSetup(trFalse): false
	// rh.isSetup(trNotOk): false
	// env.TearDown() success: true
}

func Example_routerHandlers_serve() {
	env := &TEnv{}
	_, err := env.Run(
		GenerateDeterministicKeys(),
		AddTransportManagers(),
		StartSetupTransportManager(),
		AddProcManagerAndRouter(),
	)
	fmt.Printf("env.Run success: %v\n", err == nil)

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

	fmt.Printf("serve: %v\n", rh.serve(tr, handler))
	env.PrintTearDown()

	// Output: env.Run success: true
	// serve: handler test message
	// env.TearDown() success: true
}
