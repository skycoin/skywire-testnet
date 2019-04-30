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
	// RespFailure "reject test" <nil>
}

func Example_setupHandlers_respondWith() {
	env, err := makeMockSh(setup.RespSuccess, []byte(string(setup.RespSuccess)))
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
	defer env.TearDown()

	// Create respondWith func
	respondWithFunc := env.sh.respondWith()
	fmt.Printf("respondWithFunc signature: %T\n", respondWithFunc)

	// Use respondWith func
	errChan := make(chan error, 1)
	go func() {
		errChan <- respondWithFunc("Success test", nil)
	}()

	// Receve respondWith message
	sprotoInit := setup.NewSetupProtocol(env.connInit)
	pt, data, err := sprotoInit.ReadPacket()
	fmt.Printf("%v %v %v", pt, string(data), err)

	// Output: respondWithFunc signature: func(interface {}, error) error
	// RespSuccess "Success test" <nil>
}

func Example_setupHandlers_addRules() {
	env, err := makeMockSh(setup.PacketAddRules, []byte(string(setup.PacketAddRules)))
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
	defer env.TearDown()

	// Create addRulesFunc
	addRulesFunc := env.sh.addRules()
	fmt.Printf("addRulesFunc signature: %T\n", addRulesFunc)

	// Use addRulesFunc
	trID := uuid.New()
	expireAt := time.Now().Add(2 * time.Minute)
	rules := []routing.Rule{
		routing.ForwardRule(expireAt, 2, trID),
	}
	rID, err := addRulesFunc(rules)

	fmt.Printf("routeId, err : %v, %v\n", rID, err)

	// Output: addRulesFunc signature: func([]routing.Rule) ([]routing.RouteID, error)
	// routeId, err : [1], <nil>

}

func Example_setupHandlers_deleteRules() {
	env, err := makeMockSh(setup.PacketDeleteRules, []byte(string(setup.PacketDeleteRules)))
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
	defer env.TearDown()

	addRulesFunc := env.sh.addRules()

	// add rules
	trID := uuid.New()
	expireAt := time.Now().Add(2 * time.Minute)
	rules := []routing.Rule{
		routing.ForwardRule(expireAt, 2, trID),
	}
	routes, err := addRulesFunc(rules)
	if err != nil {
		fmt.Printf("error on addRules: %v\n", err)
	}

	// Create deleteRulesFunc
	deleteRulesFunc := env.sh.deleteRules()
	fmt.Printf("deleteRulesFunc signature: %T\n", deleteRulesFunc)

	//Use deleteRulesFunc
	deletedRoutes, err := deleteRulesFunc(routes)
	if err != nil {
		fmt.Printf("error in deleteRules: %v\n", err)
	}
	fmt.Printf("deletedRoutes, err: %v, %v\n", deletedRoutes, err)

	// Output: deleteRulesFunc signature: func([]routing.RouteID) ([]routing.RouteID, error)
	// deletedRoutes, err: [1], <nil>

}

func Example_setupHandlers_confirmLoop() {
	env, err := makeMockSh(setup.PacketConfirmLoop, []byte(string(setup.PacketConfirmLoop)))
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
	defer env.TearDown()

	// Create confirmLoopFunc
	confirmLoopFunc := env.sh.confirmLoop()
	fmt.Printf("confirmLoopFunc signature: %T\n", confirmLoopFunc)

	// Use confirmLoopFunc
	pk, _, _ := cipher.GenerateDeterministicKeyPair([]byte("loopData"))

	unknownLoopData := setup.LoopData{
		RemotePK:     pk,
		RemotePort:   0,
		LocalPort:    0,
		RouteID:      routing.RouteID(0),
		NoiseMessage: []byte{},
	}

	res, err := confirmLoopFunc(unknownLoopData)
	fmt.Printf("confirmLoop(unknownLoopData): %v %v\n", res, err)

	// Output: confirmLoopFunc signature: func(setup.LoopData) ([]uint8, error)
	// confirmLoop(unknownLoopData): [] unknown loop

}

func Example_setupHandlers_loopClosed() {
	env, err := makeMockSh(setup.PacketCloseLoop, []byte(string(setup.PacketCloseLoop)))
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
	defer env.TearDown()

	// Create loopClosedFunc
	loopClosedFunc := env.sh.loopClosed()
	fmt.Printf("loopClosed signature: %T\n", loopClosedFunc)

	// Use loopClosedFunc
	pk, _, _ := cipher.GenerateDeterministicKeyPair([]byte("loopData"))
	unknownLoopData := setup.LoopData{
		RemotePK:     pk,
		RemotePort:   0,
		LocalPort:    0,
		RouteID:      routing.RouteID(0),
		NoiseMessage: []byte{},
	}

	loopClosedErr := loopClosedFunc(unknownLoopData)
	fmt.Printf("loopClosed(unknownLoopData): %v\n", loopClosedErr)

	// Output: loopClosed signature: func(setup.LoopData) error
	// loopClosed(unknownLoopData): proc not found

}

func Example_setupHandlers_handle() {

}
