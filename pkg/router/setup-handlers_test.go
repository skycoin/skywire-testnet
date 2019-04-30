package router

import (
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/google/uuid"
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
	env, err := makeMockSh(setup.RespFailure, []byte(string(setup.RespFailure)))
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
	defer env.TearDown()

	// Use reject func
	errChan := make(chan error, 1)
	go func() {
		errChan <- env.sh.reject(errors.New("reject test"))
	}()

	// Receve reject message
	sprotoInit := setup.NewSetupProtocol(env.connInit)
	pt, data, err := sprotoInit.ReadPacket()
	fmt.Printf("%v %v %v", pt, string(data), err)

	// Output: RespFailure "reject test" <nil>
}

func Example_setupHandlers_respondWith() {
	env, err := makeMockSh(setup.RespSuccess, []byte(string(setup.RespSuccess)))
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
	defer env.TearDown()

	// Use respondWith func
	errChan := make(chan error, 1)
	go func() {
		errChan <- env.sh.respondWith("Success test", nil)
	}()

	// Receve respondWith message
	sprotoInit := setup.NewSetupProtocol(env.connInit)
	pt, data, err := sprotoInit.ReadPacket()
	fmt.Printf("%v %v %v", pt, string(data), err)

	// Output: RespSuccess "Success test" <nil>
}

func Example_setupHandlers_addRules() {
	env, err := makeMockSh(setup.PacketAddRules, []byte(string(setup.PacketAddRules)))
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
	defer env.TearDown()

	// Use addRulesFunc
	trID := uuid.New()
	expireAt := time.Now().Add(2 * time.Minute)
	rules := []routing.Rule{
		routing.ForwardRule(expireAt, 2, trID),
	}
	rID, err := env.sh.addRules(rules)

	fmt.Printf("routeId, err: %v, %v\n", rID, err)

	// Output: routeId, err: [1], <nil>
}

func Example_setupHandlers_deleteRules() {
	env, err := makeMockSh(setup.PacketDeleteRules, []byte(string(setup.PacketDeleteRules)))
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
	defer env.TearDown()

	// add rules
	trID := uuid.New()
	expireAt := time.Now().Add(2 * time.Minute)
	rules := []routing.Rule{
		routing.ForwardRule(expireAt, 2, trID),
	}
	routes, err := env.sh.addRules(rules)
	if err != nil {
		fmt.Printf("error on addRules: %v\n", err)
	}

	//Use deleteRulesFunc
	// TODO(alexyu): test for unknown routes
	deletedRoutes, err := env.sh.deleteRules(routes)
	if err != nil {
		fmt.Printf("error in deleteRules: %v\n", err)
	}
	fmt.Printf("deletedRoutes, err: %v, %v\n", deletedRoutes, err)

	// Output: deletedRoutes, err: [1], <nil>
}

func Example_setupHandlers_confirmLoop() {
	env, err := makeMockSh(setup.PacketConfirmLoop, []byte(string(setup.PacketConfirmLoop)))
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
	defer env.TearDown()

	// Use confirmLoopFunc
	// TODO(alexyu): test for known loops
	pk, _, _ := cipher.GenerateDeterministicKeyPair([]byte("loopData"))

	unknownLoopData := setup.LoopData{
		RemotePK:     pk,
		RemotePort:   0,
		LocalPort:    0,
		RouteID:      routing.RouteID(0),
		NoiseMessage: []byte{},
	}

	res, err := env.sh.confirmLoop(unknownLoopData)
	fmt.Printf("confirmLoop(unknownLoopData): %v %v\n", res, err)

	// Output: confirmLoop(unknownLoopData): [] unknown loop
}

func Example_setupHandlers_loopClosed() {
	env, err := makeMockSh(setup.PacketCloseLoop, []byte(string(setup.PacketCloseLoop)))
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
	defer env.TearDown()

	// Use loopClosedFunc
	// TODO(alexyu): test with known loops
	pk, _, _ := cipher.GenerateDeterministicKeyPair([]byte("loopData"))
	unknownLoopData := setup.LoopData{
		RemotePK:     pk,
		RemotePort:   0,
		LocalPort:    0,
		RouteID:      routing.RouteID(0),
		NoiseMessage: []byte{},
	}

	loopClosedErr := env.sh.loopClosed(unknownLoopData)
	fmt.Printf("loopClosed(unknownLoopData): %v\n", loopClosedErr)

	// Output: loopClosed(unknownLoopData): proc not found
}

// WIP
type handleTestCase struct {
	packetType     setup.PacketType
	packetBodyFunc func() []byte
	runnerFunc     func() error
}

var handleTestCases = []handleTestCase{
	handleTestCase{
		packetType: setup.PacketAddRules,
		packetBodyFunc: func() []byte {
			return nil
		},
		runnerFunc: func() error {
			return nil
		},
	},
	handleTestCase{
		packetType: setup.PacketDeleteRules,
		packetBodyFunc: func() []byte {
			return nil
		},
		runnerFunc: func() error {
			return nil
		},
	},
	handleTestCase{
		packetType: setup.PacketConfirmLoop,
		packetBodyFunc: func() []byte {
			return nil
		},
		runnerFunc: func() error {
			return nil
		},
	},
	handleTestCase{
		packetType: setup.PacketLoopClosed,
		packetBodyFunc: func() []byte {
			return nil
		},
		runnerFunc: func() error {
			return nil
		},
	},
}

// WIP
func Example_handle() {
	for _, tc := range handleTestCases {
		env, err := makeMockSh(tc.packetType, tc.packetBodyFunc())
		if err != nil {
			fmt.Printf("error: %v\n", err)
		}
		env.TearDown()
	}
	fmt.Println("Finish")

	// Output: Finish
}
