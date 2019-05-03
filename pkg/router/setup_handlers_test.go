package router

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/google/uuid"

	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/setup"
)

type mockShEnv struct {
	connResp net.Conn
	connInit net.Conn

	sprotoInit *setup.Protocol

	sh  setupHandlers
	err error

	env *mockRouterEnv
}

func makeSetupHandlersEnv() (*mockShEnv, error) {
	env, err := makeMockRouterEnv()
	if err != nil {
		return &mockShEnv{}, err
	}

	connInit, connResp := net.Pipe()
	sprotoInit := setup.NewSetupProtocol(connInit)

	errCh := make(chan error, 1)
	go func() {
		errCh <- sprotoInit.WritePacket(setup.PacketType(42), []byte("Ultimate Answer"))
	}()

	sh, err := makeSetupHandlers(env.r, env.pm, connResp)
	if err != nil {
		return &mockShEnv{}, err
	}

	return &mockShEnv{connResp, connInit, sprotoInit, sh, <-errCh, env}, nil
}

func (shEnv *mockShEnv) TearDown() {
	shEnv.connResp.Close()
	shEnv.connInit.Close()
	// shEnv.env.TearDown()
}

func Example_makeSetupHandlersEnv() {

	envSh, err := makeSetupHandlersEnv()

	fmt.Printf("makeSetupHandlersEnv success: %v\n", err == nil)
	fmt.Printf("envSh.packetType: %v\n", envSh.sh.packetType)
	fmt.Printf("envSh.packetBody: %v\n", string(envSh.sh.packetBody))
	defer envSh.TearDown()

	go func() {
		if _, err = envSh.connInit.Write([]byte("Hello")); err != nil {
			fmt.Println(err)
		}
	}()

	var buf []byte
	n, err := envSh.connResp.Read(buf)
	fmt.Printf("envSh.connResp.Read: %v, %v, %v\n", string(buf), n, err == nil)

	//Output: makeSetupHandlersEnv success: true
	// envSh.packetType: Unknown(42)
	// envSh.packetBody: "VWx0aW1hdGUgQW5zd2Vy"
	// envSh.connResp.Read: , 0, true

}

func Example_setupHandlers_reject() {

	envSh, err := makeSetupHandlersEnv()
	fmt.Printf("makeSetupHandlersEnv success: %v\n", err == nil)

	// Use reject func
	errChan := make(chan error, 1)
	go func() {
		errChan <- envSh.sh.reject(errors.New("reject test"))
	}()

	// Receve reject message
	pt, data, err := envSh.sprotoInit.ReadPacket()
	fmt.Printf("%v %v %v", pt, string(data), err)

	// Output: makeSetupHandlersEnv success: true
	// RespFailure "reject test" <nil>
}

func Example_setupHandlers_respondWith() {

	envSh, err := makeSetupHandlersEnv()
	fmt.Printf("makeSetupHandlersEnv success: %v\n", err == nil)

	// Use respondWith func
	errChan := make(chan error, 1)
	go func() {
		errChan <- envSh.sh.respondWith("Success test", nil)
	}()

	// Receve respondWith message

	pt, data, err := envSh.sprotoInit.ReadPacket()
	fmt.Printf("%v %v %v", pt, string(data), err)

	// Output: makeSetupHandlersEnv success: true
	// RespSuccess "Success test" <nil>
}

func Example_setupHandlers_addRules() {

	envSh, err := makeSetupHandlersEnv()
	fmt.Printf("makeSetupHandlersEnv success: %v\n", err == nil)

	// Add ForwardRule
	trID := uuid.New()
	expireAt := time.Now().Add(2 * time.Minute)
	rules := []routing.Rule{
		routing.ForwardRule(expireAt, 2, trID),
		routing.AppRule(time.Now(), 3, envSh.env.pkRemote, 3, 2),
	}
	rID, err := envSh.sh.addRules(rules)

	fmt.Printf("routeId, err: %v, %v\n", rID, err)

	// Output: makeSetupHandlersEnv success: true
	// routeId, err: [1 2], <nil>
}

func Example_setupHandlers_deleteRules() {

	envSh, err := makeSetupHandlersEnv()
	fmt.Printf("makeSetupHandlersEnv success: %v\n", err == nil)

	// add rules
	trID := uuid.New()
	expireAt := time.Now().Add(2 * time.Minute)
	rules := []routing.Rule{
		routing.ForwardRule(expireAt, 2, trID),
	}
	routes, err := envSh.sh.addRules(rules)
	if err != nil {
		fmt.Printf("error on addRules: %v\n", err)
	}

	// delete rules
	// TODO(alexyu): test for unknown routes
	deletedRoutes, err := envSh.sh.deleteRules(routes)
	if err != nil {
		fmt.Printf("error in deleteRules: %v\n", err)
	}
	fmt.Printf("deletedRoutes, err: %v, %v\n", deletedRoutes, err)

	// Output: makeSetupHandlersEnv success: true
	// deletedRoutes, err: [1], <nil>
}

func Example_setupHandlers_confirmLoop() {
	env, err := makeMockRouterEnv()
	fmt.Printf("makeMockRouterEnv success: %v\n", err == nil)
	// defer env.TearDown()

	envSh, err := makeSetupHandlersEnv()
	fmt.Printf("makeSetupHandlersEnv success: %v\n", err == nil)

	unknownLoopData := setup.LoopData{
		RemotePK:     env.pkRemote,
		RemotePort:   3,
		LocalPort:    2,
		RouteID:      routing.RouteID(1),
		NoiseMessage: []byte{},
	}

	res, err := envSh.sh.confirmLoop(unknownLoopData)
	fmt.Printf("confirmLoop(unknownLoopData): %v %v\n", res, err)

	// Output: makeMockRouterEnv success: true
	// makeSetupHandlersEnv success: true
	// confirmLoop(unknownLoopData): [] unknown loop
}

func Example_setupHandlers_loopClosed() {
	env, err := makeMockRouterEnv()
	fmt.Printf("makeMockRouterEnv success: %v\n", err == nil)
	// defer env.TearDown()

	envSh, err := makeSetupHandlersEnv()
	fmt.Printf("makeSetupHandlersEnv success: %v\n", err == nil)

	unknownLoopData := setup.LoopData{
		RemotePK:     env.pkSetup,
		RemotePort:   0,
		LocalPort:    0,
		RouteID:      routing.RouteID(0),
		NoiseMessage: []byte{},
	}

	loopClosedErr := envSh.sh.loopClosed(unknownLoopData)
	fmt.Printf("loopClosed(unknownLoopData): %v\n", loopClosedErr)

	// Output: makeMockRouterEnv success: true
	// makeSetupHandlersEnv success: true
	// loopClosed(unknownLoopData): proc not found
}

// WIP
type handleTestCase struct {
	packetType setup.PacketType
	bodyFunc   func(*mockShEnv) (*mockShEnv, error)
}

var handleTestCases = []handleTestCase{
	handleTestCase{
		packetType: setup.PacketAddRules,
		bodyFunc: func(env *mockShEnv) (*mockShEnv, error) {

			trID := uuid.New()
			expireAt := time.Now().Add(2 * time.Minute)
			rules := []routing.Rule{
				routing.ForwardRule(expireAt, 2, trID),
			}
			body, err := json.Marshal(rules)
			env.sh.packetBody = body

			return env, err
		},
	},
	handleTestCase{
		packetType: setup.PacketAddRules,
		bodyFunc: func(env *mockShEnv) (*mockShEnv, error) {
			env.sh.packetBody = []byte("invalid packet body")
			return env, nil
		},
	},
	handleTestCase{
		packetType: setup.PacketDeleteRules,
		bodyFunc: func(env *mockShEnv) (*mockShEnv, error) {
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

			body, err := json.Marshal(routes)

			env.sh.packetBody = body
			return env, err
		},
	},
	handleTestCase{
		packetType: setup.PacketDeleteRules,
		bodyFunc: func(env *mockShEnv) (*mockShEnv, error) {
			env.sh.packetBody = []byte("invalid packet body")
			return env, nil
		},
	},
	handleTestCase{
		packetType: setup.PacketConfirmLoop,
		bodyFunc: func(env *mockShEnv) (*mockShEnv, error) {

			loopData := setup.LoopData{
				RemotePK:     env.env.pkLocal,
				RemotePort:   0,
				LocalPort:    0,
				RouteID:      routing.RouteID(0),
				NoiseMessage: []byte{},
			}
			body, err := json.Marshal(loopData)

			env.sh.packetBody = body
			return env, err
		},
	},
	handleTestCase{
		packetType: setup.PacketConfirmLoop,
		bodyFunc: func(env *mockShEnv) (*mockShEnv, error) {
			env.sh.packetBody = []byte("invalid packet body")
			return env, nil
		},
	},
	handleTestCase{
		packetType: setup.PacketLoopClosed,
		bodyFunc: func(env *mockShEnv) (*mockShEnv, error) {

			unknownLoopData := setup.LoopData{
				RemotePK:     env.env.pkRemote,
				RemotePort:   0,
				LocalPort:    0,
				RouteID:      routing.RouteID(0),
				NoiseMessage: []byte{},
			}
			body, err := json.Marshal(unknownLoopData)

			env.sh.packetBody = body
			return env, err

		},
	},
	handleTestCase{
		packetType: setup.PacketLoopClosed,
		bodyFunc: func(env *mockShEnv) (*mockShEnv, error) {
			env.sh.packetBody = []byte("invalid packet body")
			return env, nil
		},
	},
	handleTestCase{
		packetType: setup.PacketType(42),
		bodyFunc: func(env *mockShEnv) (*mockShEnv, error) {
			env.sh.packetBody = []byte("invalid packet body")
			return env, nil
		},
	},
}

func Example_handle() {
	fmt.Println("Start")
	for _, tc := range handleTestCases {

		envSh, err := makeSetupHandlersEnv()
		if err != nil {
			fmt.Println(err)
		}
		defer envSh.TearDown()

		envSh, err = tc.bodyFunc(envSh)
		if err != nil {
			fmt.Printf("%v\n", err)
		}

		errCh := make(chan error, 1)
		go func() {
			envSh.sh.packetType = tc.packetType
			errCh <- envSh.sh.handle()
		}()

		pt, data, err := envSh.sprotoInit.ReadPacket()
		fmt.Printf("response: %v %v %v\n", pt, string(data), err)

	}
	fmt.Println("Finish")

	// Output: Start
	// response: RespSuccess [1] <nil>
	// response: RespFailure "invalid character 'i' looking for beginning of value" <nil>
	// response: RespSuccess [1] <nil>
	// response: RespFailure "invalid character 'i' looking for beginning of value" <nil>
	// response: RespFailure "unknown loop" <nil>
	// response: RespFailure "invalid character 'i' looking for beginning of value" <nil>
	// response: RespFailure "proc not found" <nil>
	// response: RespFailure "invalid character 'i' looking for beginning of value" <nil>
	// response: RespFailure "unknown foundation packet" <nil>
	// Finish

}
