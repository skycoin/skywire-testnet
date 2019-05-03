package router

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/cipher"
	routeFinder "github.com/skycoin/skywire/pkg/route-finder/client"
	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/setup"
	"github.com/skycoin/skywire/pkg/transport"
)

type mockRouterEnv struct {
	r  *router
	pm ProcManager

	tpmLocal *transport.Manager
	tpmSetup *transport.Manager

	pkLocal   cipher.PubKey
	skLocal   cipher.SecKey
	pkSetup   cipher.PubKey
	pkRespond cipher.PubKey

	rt      routing.Table
	routeID routing.RouteID
}

type mockEnv struct {
	r        *router
	pm       ProcManager
	connResp net.Conn
	connInit net.Conn

	sh  setupHandlers
	err error

	*mockRouterEnv
}

func makeMockEnv() (*mockEnv, error) {

	env, err := makeMockRouterEnv()
	if err != nil {
		return &mockEnv{}, err
	}

	connInit, connResp := net.Pipe()
	pm := env.pm //IDK why it's 10
	sprotoInit := setup.NewSetupProtocol(connInit)

	errCh := make(chan error, 1)
	go func() {
		errCh <- sprotoInit.WritePacket(setup.PacketType(0), []byte{})
	}()

	sh, err := makeSetupHandlers(env.r, pm, connResp)
	if err != nil {
		return &mockEnv{}, err
	}

	return &mockEnv{env.r, pm, connResp, connInit, sh, <-errCh, env}, nil
}

func makeMockRouterEnv() (*mockRouterEnv, error) {
	dClient := transport.NewDiscoveryMock()
	rfc := routeFinder.NewMock()
	logStore := transport.InMemoryTransportLogStore()

	pkLocal, skLocal, _ := cipher.GenerateDeterministicKeyPair([]byte("local")) // nolint: errcheck
	pkSetup, skSetup, _ := cipher.GenerateDeterministicKeyPair([]byte("setup")) // nolint: errcheck
	pkRespond, _, _ := cipher.GenerateDeterministicKeyPair([]byte("respond"))   // nolint: errcheck

	fLocal, fSetup := transport.NewMockFactoryPair(pkLocal, pkSetup)
	fLocal.SetType("messaging")

	tpmLocal, err := transport.NewManager(&transport.ManagerConfig{PubKey: pkLocal, SecKey: skLocal, DiscoveryClient: dClient, LogStore: logStore}, fLocal)
	if err != nil {
		return &mockRouterEnv{}, err
	}

	tpmSetup, err := transport.NewManager(&transport.ManagerConfig{PubKey: pkSetup, SecKey: skSetup, DiscoveryClient: dClient, LogStore: logStore}, fSetup)
	if err != nil {
		return &mockRouterEnv{}, err
	}

	go tpmSetup.Serve(context.TODO()) // nolint: errcheck

	rt := routing.InMemoryRoutingTable()

	rule := routing.AppRule(time.Now().Add(time.Hour), 4, pkRespond, 6, 5)
	routeID, err := rt.AddRule(rule)
	if err != nil {
		return &mockRouterEnv{}, err
	}

	logger := logging.MustGetLogger("mockRouterEnv")

	conf := &Config{
		PubKey:     pkLocal,
		SecKey:     skLocal,
		SetupNodes: []cipher.PubKey{pkSetup},
	}

	r := &router{
		log:  logger,
		conf: conf,
		tpm:  tpmLocal,
		rtm:  NewRoutingTableManager(logger, rt, DefaultRouteKeepalive, DefaultRouteCleanupDuration),
		rfc:  rfc,
	}
	pm := NewProcManager(10)

	return &mockRouterEnv{
		r:  r,
		pm: pm,

		tpmSetup: tpmSetup,
		tpmLocal: tpmLocal,

		pkLocal:   pkLocal,
		skLocal:   skLocal,
		pkSetup:   pkSetup,
		pkRespond: pkRespond,

		routeID: routeID,
		rt:      rt,
	}, nil
}

func (rEnv *mockRouterEnv) TearDown() error {
	errCh := make(chan error, 4)
	errCh <- rEnv.tpmLocal.Close()
	errCh <- rEnv.tpmSetup.Close()
	errCh <- rEnv.r.Close()
	errCh <- rEnv.pm.Close()

	for err := range errCh {
		if err != nil {
			close(errCh)
			return err
		}
	}
	close(errCh)
	return nil
}

func (shEnv *mockEnv) TearDown() {
	shEnv.connResp.Close()
	shEnv.connInit.Close()
	err := shEnv.sh.r.Close()
	if err != nil {
		panic(err)
	}
	err = shEnv.sh.pm.Close()
	if err != nil {
		panic(err)
	}
}

func Example_makeMockEnv() {
	env, err := makeMockEnv()
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
	defer env.TearDown()

	fmt.Printf("sh.packetType: %v\n", env.sh.packetType)
	fmt.Printf("sh.packetBody: %v\n", string(env.sh.packetBody))

	//Output: sh.packetType: AddRules
	// sh.packetBody: ""
}
