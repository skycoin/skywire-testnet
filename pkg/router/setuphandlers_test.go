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
	// env *testEnv
	stpHandlers setupHandlers

	connResp   net.Conn
	connInit   net.Conn
	sprotoInit *setup.Protocol
}

func AddSetupHandlersEnv() envStep {
	return func(env *testEnv) (stepname string, err error) {
		stepname = "AddSetupHandersEnv"

		connInit, connResp := net.Pipe()
		sprotoInit := setup.NewSetupProtocol(connInit)

		errCh := make(chan error, 1)
		go func() {
			errCh <- sprotoInit.WritePacket(setup.PacketType(42), []byte("Ultimate Answer"))
		}()

		stpHandlers, err := makeSetupHandlers(env.R, env.procMgr, connResp)
		if err != nil {
			return
		}

		env.SH = &mockShEnv{
			stpHandlers: stpHandlers,
			connResp:    connResp,
			connInit:    connInit,
			sprotoInit:  sprotoInit,
		}
		return
	}
}

func setupHandlersTestEnv() (env *testEnv, err error) {
	env = &testEnv{}
	_, err = env.runSteps(
		GenKeys(),
		AddTransportManagers(),
		AddProcManagerAndRouter(),
		AddSetupHandlersEnv(),
	)
	return
}

// func makeSetupHandlersEnv() (*mockShEnv, error) {

// 	env, err := makeMockRouterEnv()
// 	if err != nil {
// 		return &mockShEnv{}, err
// 	}

// 	connInit, connResp := net.Pipe()
// 	sprotoInit := setup.NewSetupProtocol(connInit)

// 	errCh := make(chan error, 1)
// 	go func() {
// 		errCh <- sprotoInit.WritePacket(setup.PacketType(42), []byte("Ultimate Answer"))
// 	}()

// 	sh, err := makeSetupHandlers(env.R, env.procMgr, connResp)
// 	if err != nil {
// 		return &mockShEnv{}, err
// 	}

// 	return &mockShEnv{
// 		env:      env,
// 		connResp: connResp,
// 		connInit: connInit,
// 		sh:       sh,
// 	}, <-errCh
// }

func (SH *mockShEnv) TearDown() {
	SH.connResp.Close()
	SH.connInit.Close()
	// SH.env.TearDown()
}

func Example_makeSetupHandlersEnv() {

	env, err := setupHandlersTestEnv()

	fmt.Printf("makeSetupHandlersEnv success: %v\n", err == nil)
	fmt.Printf("envSh.packetType: %v\n", env.SH.stpHandlers.packetType)
	fmt.Printf("envSh.packetBody: %v\n", string(env.SH.stpHandlers.packetBody))
	defer env.TearDown()

	go func() {
		if _, err = env.SH.connInit.Write([]byte("Hello")); err != nil {
			fmt.Println(err)
		}
	}()

	var buf []byte
	n, err := env.SH.connResp.Read(buf)
	fmt.Printf("envSh.connResp.Read: %v, %v, %v\n", string(buf), n, err == nil)

	//Output: makeSetupHandlersEnv success: true
	// envSh.packetType: Unknown(42)
	// envSh.packetBody: "VWx0aW1hdGUgQW5zd2Vy"
	// envSh.connResp.Read: , 0, true

}

func Example_setupHandlers_reject() {

	env, err := setupHandlersTestEnv()
	fmt.Printf("setupHandlersTestEnv success: %v\n", err == nil)

	// Use reject func
	errChan := make(chan error, 1)
	go func() {
		errChan <- env.SH.stpHandlers.reject(errors.New("reject test"))
	}()

	// Receve reject message
	pt, data, err := env.SH.sprotoInit.ReadPacket()
	fmt.Printf("%v %v %v", pt, string(data), err)

	// Output: makeSetupHandlersEnv success: true
	// RespFailure "reject test" <nil>
}

func Example_setupHandlers_respondWith() {

	env, err := setupHandlersTestEnv()
	fmt.Printf("makeSetupHandlersEnv success: %v\n", err == nil)

	// Use respondWith func
	errChan := make(chan error, 1)
	go func() {
		errChan <- env.SH.stpHandlers.respondWith("Success test", nil)
	}()

	// Receve respondWith message
	pt, data, err := env.SH.sprotoInit.ReadPacket()
	fmt.Printf("%v %v %v", pt, string(data), err)

	// Output: makeSetupHandlersEnv success: true
	// RespSuccess "Success test" <nil>
}

func Example_setupHandlers_addRules() {

	env := &testEnv{}
	_, err := env.runSteps(
		GenKeys(),
		AddTransportManagers(),
		AddProcManagerAndRouter(),
		AddSetupHandlersEnv(),
	)
	fmt.Printf("testEnv success: %v\n", err == nil)

	// Add ForwardRule
	trID := uuid.New()
	expireAt := time.Now().Add(2 * time.Minute)
	rules := []routing.Rule{
		routing.ForwardRule(expireAt, 2, trID),
		routing.AppRule(time.Now(), 3, env.pkRemote, 3, 2),
	}
	rID, err := env.SH.stpHandlers.addRules(rules)

	fmt.Printf("routeId, err: %v, %v\n", rID, err)

	// Output: makeSetupHandlersEnv success: true
	// routeId, err: [1 2], <nil>
}

func Example_setupHandlers_deleteRules() {

	env, err := setupHandlersTestEnv()
	fmt.Printf("success: %v\n", err == nil)

	// add rules
	trID := uuid.New()
	expireAt := time.Now().Add(2 * time.Minute)
	rules := []routing.Rule{
		routing.ForwardRule(expireAt, 2, trID),
	}
	routes, err := env.SH.stpHandlers.addRules(rules)
	if err != nil {
		fmt.Printf("error on addRules: %v\n", err)
	}

	deletedRoutes, err := env.SH.stpHandlers.deleteRules(routes)
	if err != nil {
		fmt.Printf("error in deleteRules: %v\n", err)
	}
	fmt.Printf("deletedRoutes, err: %v, %v\n", deletedRoutes, err)

	// Output: makeSetupHandlersEnv success: true
	// deletedRoutes, err: [1], <nil>
}

func Example_setupHandlers_loopClosed() {
	env, err := setupHandlersTestEnv()
	fmt.Printf("setupHandlersTestEnv success: %v\n", err == nil)

	unknownLoopData := setup.LoopData{
		RemotePK:     env.pkSetup,
		RemotePort:   0,
		LocalPort:    0,
		RouteID:      routing.RouteID(0),
		NoiseMessage: []byte{},
	}

	loopClosedErr := env.SH.stpHandlers.loopClosed(unknownLoopData)
	fmt.Printf("loopClosed(unknownLoopData): %v\n", loopClosedErr)

	// Output: makeMockRouterEnv success: true
	// makeSetupHandlersEnv success: true
	// loopClosed(unknownLoopData): proc not found
}

// WIP
type handleTestCase struct {
	packetType setup.PacketType
	bodyFunc   func(*testEnv) (*testEnv, error)
}

var handleTestCases = []handleTestCase{
	handleTestCase{
		packetType: setup.PacketAddRules,
		bodyFunc: func(env *testEnv) (*testEnv, error) {

			trID := uuid.New()
			expireAt := time.Now().Add(2 * time.Minute)
			rules := []routing.Rule{
				routing.ForwardRule(expireAt, 2, trID),
			}
			body, err := json.Marshal(rules)
			env.SH.stpHandlers.packetBody = body

			return env, err
		},
	},
	handleTestCase{
		packetType: setup.PacketAddRules,
		bodyFunc: func(env *testEnv) (*testEnv, error) {
			env.SH.stpHandlers.packetBody = []byte("invalid packet body")
			return env, nil
		},
	},
	handleTestCase{
		packetType: setup.PacketDeleteRules,
		bodyFunc: func(env *testEnv) (*testEnv, error) {
			// add rules
			trID := uuid.New()
			expireAt := time.Now().Add(2 * time.Minute)
			rules := []routing.Rule{
				routing.ForwardRule(expireAt, 2, trID),
			}
			routes, err := env.SH.stpHandlers.addRules(rules)
			if err != nil {
				fmt.Printf("error on addRules: %v\n", err)
			}

			body, err := json.Marshal(routes)

			env.SH.stpHandlers.packetBody = body
			return env, err
		},
	},
	handleTestCase{
		packetType: setup.PacketDeleteRules,
		bodyFunc: func(env *testEnv) (*testEnv, error) {
			env.SH.stpHandlers.packetBody = []byte("invalid packet body")
			return env, nil
		},
	},
	handleTestCase{
		packetType: setup.PacketConfirmLoop,
		bodyFunc: func(env *testEnv) (*testEnv, error) {

			loopData := setup.LoopData{
				RemotePK:     env.pkLocal,
				RemotePort:   0,
				LocalPort:    0,
				RouteID:      routing.RouteID(0),
				NoiseMessage: []byte{},
			}
			body, err := json.Marshal(loopData)

			env.SH.stpHandlers.packetBody = body
			return env, err
		},
	},
	handleTestCase{
		packetType: setup.PacketConfirmLoop,
		bodyFunc: func(env *testEnv) (*testEnv, error) {
			env.SH.stpHandlers.packetBody = []byte("invalid packet body")
			return env, nil
		},
	},
	handleTestCase{
		packetType: setup.PacketLoopClosed,
		bodyFunc: func(env *testEnv) (*testEnv, error) {

			unknownLoopData := setup.LoopData{
				RemotePK:     env.pkRemote,
				RemotePort:   0,
				LocalPort:    0,
				RouteID:      routing.RouteID(0),
				NoiseMessage: []byte{},
			}
			body, err := json.Marshal(unknownLoopData)

			env.SH.stpHandlers.packetBody = body
			return env, err

		},
	},
	handleTestCase{
		packetType: setup.PacketLoopClosed,
		bodyFunc: func(env *testEnv) (*testEnv, error) {
			env.SH.stpHandlers.packetBody = []byte("invalid packet body")
			return env, nil
		},
	},
	handleTestCase{
		packetType: setup.PacketType(42),
		bodyFunc: func(env *testEnv) (*testEnv, error) {
			env.SH.stpHandlers.packetBody = []byte("invalid packet body")
			return env, nil
		},
	},
}

func Example_handle() {
	fmt.Println("Start")
	for _, tc := range handleTestCases {

		env := &testEnv{}
		_, err := env.runSteps(
			GenKeys(),
			AddTransportManagers(),
			AddProcManagerAndRouter(),
			AddSetupHandlersEnv(),
		)
		// envSh, err := makeSetupHandlersEnv()
		if err != nil {
			fmt.Println(err)
		}
		defer env.TearDown()

		env, err = tc.bodyFunc(env)
		if err != nil {
			fmt.Printf("%v\n", err)
		}

		errCh := make(chan error, 1)
		go func() {
			env.SH.stpHandlers.packetType = tc.packetType
			errCh <- env.SH.stpHandlers.handle()
		}()

		pt, data, err := env.SH.sprotoInit.ReadPacket()
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
