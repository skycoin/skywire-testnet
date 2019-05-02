package router

import (
	"fmt"
	"net"

	"github.com/skycoin/skycoin/src/util/logging"
	"github.com/skycoin/skywire/pkg/cipher"
	routeFinder "github.com/skycoin/skywire/pkg/route-finder/client"
	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/setup"
	"github.com/skycoin/skywire/pkg/transport"
)

type mockEnv struct {
	r        *router
	pm       ProcManager
	connResp net.Conn
	connInit net.Conn
	sh       setupHandlers
	err      error
}

func makeMockEnv() (*mockEnv, error) {
	connInit, connResp := net.Pipe()
	r := makeMockRouter()
	pm := NewProcManager(10) //IDK why it's 10
	sprotoInit := setup.NewSetupProtocol(connInit)

	errCh := make(chan error, 1)
	go func() {
		errCh <- sprotoInit.WritePacket(setup.PacketType(0), []byte{})
	}()

	sh, err := makeSetupHandlers(r, pm, connResp)
	if err != nil {
		return &mockEnv{}, err
	}

	return &mockEnv{r, pm, connResp, connInit, sh, <-errCh}, nil
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

func makeMockRouter() *router {
	logger := logging.MustGetLogger("router")

	pk, sk := cipher.GenerateKeyPair()

	// TODO(alexyu): SetupNodes

	conf := &Config{
		PubKey:     pk,
		SecKey:     sk,
		SetupNodes: []cipher.PubKey{},
	}

	// TODO(alexyu):  This mock must be simplified
	_, tpm, _ := transport.MockTransportManager() //nolint: errcheck
	rtm := NewRoutingTableManager(
		logging.MustGetLogger("rt_manager"),
		routing.InMemoryRoutingTable(),
		DefaultRouteKeepalive,
		DefaultRouteCleanupDuration)

	return &router{
		log:  logger,
		conf: conf,
		tpm:  tpm,
		rtm:  rtm,
		rfc:  routeFinder.NewMock(),
	}
}
