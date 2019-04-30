package router

import (
	"errors"
	"fmt"
	"net"

	"github.com/skycoin/skycoin/src/util/logging"
	"github.com/skycoin/skywire/pkg/cipher"
	routeFinder "github.com/skycoin/skywire/pkg/route-finder/client"
	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/setup"
	"github.com/skycoin/skywire/pkg/transport"
)

func makeMockRouter() *router {
	logger := logging.MustGetLogger("router")

	pk, sk := cipher.GenerateKeyPair()
	conf := &Config{
		PubKey:     pk,
		SecKey:     sk,
		SetupNodes: []cipher.PubKey{},
	}

	// TODO(alexyu):  This mock must be simplified
	_, tpm, _ := transport.MockTransportManager()
	rtm := NewRoutingTableManager(
		logging.MustGetLogger("rt_manager"),
		routing.InMemoryRoutingTable(),
		DefaultRouteKeepalive,
		DefaultRouteCleanupDuration)

	r := router{
		log:  logger,
		conf: conf,
		tpm:  tpm,
		rtm:  rtm,
		rfc:  routeFinder.NewMock(),
	}
	return &r
}

type mockSh struct {
	r        *router
	pm       ProcManager
	connResp net.Conn
	connInit net.Conn
	sh       setupHandlers
	err      error
}

func makeMockSh(packetType setup.PacketType, packetBody []byte) (*mockSh, error) {
	connInit, connResp := net.Pipe()
	r := makeMockRouter()
	pm := NewProcManager(10) //IDK why it's 10
	sprotoInit := setup.NewSetupProtocol(connInit)

	errChan := make(chan error, 1)
	go func() {
		errChan <- sprotoInit.WritePacket(packetType, packetBody)
	}()

	sh, err := makeSetupHandlers(r, pm, connResp)
	if err != nil {
		return &mockSh{}, err
	}

	return &mockSh{r, pm, connResp, connInit, sh, <-errChan}, nil
}

func (shEnv *mockSh) TearDown() {
	shEnv.connResp.Close()
	shEnv.connInit.Close()
}

func Example_makeMockSh() {
	env, err := makeMockSh(setup.PacketAddRules, []byte("Hello"))
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
	defer env.TearDown()

	// time.Sleep(10 * time.Millisecond)

	fmt.Printf("sh.packetType: %v\n", env.sh.packetType)
	fmt.Printf("sh.packetBody: %v\n", string(env.sh.packetBody))

	//Output: sh.packetType: AddRules
	// sh.packetBody: "SGVsbG8="
}

func Example_setupHandlers_reject() {
	env, err := makeMockSh(setup.PacketAddRules, []byte("Hello"))
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
	defer env.TearDown()

	// Create reject func
	rejectFunc := env.sh.reject()
	fmt.Printf("rejectFunc signature: %T\n", rejectFunc)

	// Use reject func
	errChan := make(chan error, 1)
	go func() {
		errChan <- rejectFunc(errors.New("reject test"))
	}()

	// Receve reject message
	sprotoInit := setup.NewSetupProtocol(env.connInit)
	pt, data, err := sprotoInit.ReadPacket()
	fmt.Printf("%v %v %v", pt, string(data), err)

	// Output: rejectFunc signature: func(error) error
	// Unknown(254) "reject test" <nil>
}
