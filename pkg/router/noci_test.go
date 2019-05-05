// +build !no_ci

package router

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/pkg/app"
	"github.com/skycoin/skywire/pkg/setup"
)

func routerTestEnv() (env *TEnv, err error) {
	env = &TEnv{}
	_, err = env.Run(
		ChangeLogLevel("error"),
		GenerateDeterministicKeys(),
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
	env, err := routerTestEnv()
	fmt.Printf("routerTestEnv success: %v\n", err == nil)
	_, err = env.Run(StartSetupTransportManager())
	fmt.Printf("StartSetupTransportManager success: %v\n", err == nil)

	go func() {
		err = env.R.Serve(context.TODO(), env.procMgr)
		fmt.Printf("router.Serve error: %v\n", err)
	}()

	time.Sleep(time.Second)

	// Output: routerTestEnv success: true
	// StartSetupTransportManager success: true
}

func TestRouterAncientTest(t *testing.T) {

	env := &TEnv{}
	_, err := env.RunAsTest(t,
		// ChangeLogLevel("error"),
		GenerateDeterministicKeys(),
		AddTransportManagers(),
		AddProcManagerAndRouter(),
	)
	require.NoError(t, err)

	errCh := make(chan error)
	go func() {
		acceptCh, _ := env.tpmSetup.Observe()
		tr := <-acceptCh

		proto := setup.NewSetupProtocol(tr)
		p, data, err := proto.ReadPacket()
		if err != nil {
			errCh <- err
			return
		}

		if p != setup.PacketCloseLoop {
			errCh <- errors.New("unknown command")
			return
		}

		ld := &setup.LoopData{}
		if err := json.Unmarshal(data, ld); err != nil {
			errCh <- err
			return
		}

		if ld.LocalPort != 5 || ld.RemotePort != 6 || ld.RemotePK != env.pkRemote {
			errCh <- errors.New("invalid payload")
			return
		}

		errCh <- proto.WritePacket(setup.RespSuccess, []byte{})
	}()

	// rw, rwIn := net.Pipe()
	// // go r.ServeApp(rwIn, 5) // nolint: errcheck

	go env.R.Serve(context.TODO(), env.procMgr) // nolint: errcheck

	// proto := appnet.NewProtocol(rw)
	// go proto.Serve(nil) // nolint: errcheck

	time.Sleep(100 * time.Millisecond)

	// raddr := &app.LoopAddr{PubKey: pk3, Port: 6}
	// require.NoError(t, r.pm.SetLoop(5, raddr, &loop{}))

	// require.NoError(t, rw.Close())

	// time.Sleep(100 * time.Millisecond)

	// require.NoError(t, <-errCh)
	// _, err = r.pm.Get(5)
	// require.Error(t, err)

	// rule, err = rt.Rule(routeID)
	// require.NoError(t, err)
	// require.Nil(t, rule)
	env.NoErrorTearDown(t)
}
