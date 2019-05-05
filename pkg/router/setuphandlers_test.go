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

func AddSetupHandlersEnv() CfgStep {
	return func(env *TEnv) (stepname string, err error) {
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
		env.stpHandlers = stpHandlers
		env.connResp = connResp
		env.connInit = connInit
		env.sprotoInit = sprotoInit
		return
	}
}

func Example_makeSetupHandlersEnv() {

	env := &TEnv{}
	_, err := env.Run(
		GenerateDeterministicKeys(),
		AddTransportManagers(),
		AddProcManagerAndRouter(),
		AddSetupHandlersEnv(),
	)

	fmt.Printf("env.Run() success: %v\n", err == nil)
	fmt.Printf("packetType: %v\n", env.stpHandlers.packetType)
	fmt.Printf("packetBody: %v\n", string(env.stpHandlers.packetBody))
	defer env.PrintTearDown()

	go func() {
		if _, err = env.connInit.Write([]byte("Hello")); err != nil {
			fmt.Println(err)
		}
	}()

	var buf []byte
	n, err := env.connResp.Read(buf)
	fmt.Printf("envSh.connResp.Read: %v, %v, %v\n", string(buf), n, err == nil)

	// Output: env.Run() success: true
	// packetType: Unknown(42)
	// packetBody: "VWx0aW1hdGUgQW5zd2Vy"
	// envSh.connResp.Read: , 0, true
	// env.TearDown() success: true

}

func Example_setupHandlers_reject() {

	env := &TEnv{}
	_, err := env.Run(
		GenerateDeterministicKeys(),
		AddTransportManagers(),
		AddProcManagerAndRouter(),
		AddSetupHandlersEnv(),
	)

	fmt.Printf("env.Run() success: %v\n", err == nil)

	// Use reject func
	errChan := make(chan error, 1)
	go func() {
		errChan <- env.stpHandlers.reject(errors.New("reject test"))
	}()

	// Receve reject message
	pt, data, err := env.sprotoInit.ReadPacket()
	fmt.Printf("%v %v %v", pt, string(data), err)

	// Output: env.Run() success: true
	// RespFailure "reject test" <nil>

}

func Example_setupHandlers_respondWith() {

	env := &TEnv{}
	_, err := env.Run(
		GenerateDeterministicKeys(),
		AddTransportManagers(),
		AddProcManagerAndRouter(),
		AddSetupHandlersEnv(),
	)

	fmt.Printf("env.Run() success: %v\n", err == nil)

	// Use respondWith func
	errChan := make(chan error, 1)
	go func() {
		errChan <- env.stpHandlers.respondWith("Success test", nil)
	}()

	// Receve respondWith message
	pt, data, err := env.sprotoInit.ReadPacket()
	fmt.Printf("%v %v %v", pt, string(data), err)

	// Output: env.Run() success: true
	// RespSuccess "Success test" <nil>

}

func Example_setupHandlers_addRules() {

	env := &TEnv{}
	_, err := env.Run(
		GenerateDeterministicKeys(),
		AddTransportManagers(),
		AddProcManagerAndRouter(),
		AddSetupHandlersEnv(),
	)
	fmt.Printf("env.Run() success: %v\n", err == nil)

	// Add ForwardRule
	trID := uuid.New()
	expireAt := time.Now().Add(2 * time.Minute)
	rules := []routing.Rule{
		routing.ForwardRule(expireAt, 2, trID),
		routing.AppRule(time.Now(), 3, env.pkRemote, 3, 2),
	}
	rID, err := env.stpHandlers.addRules(rules)

	fmt.Printf("routeId, err: %v, %v\n", rID, err)

	// Output: env.Run() success: true
	// routeId, err: [1 2], <nil>

}

func Example_setupHandlers_deleteRules() {

	env := &TEnv{}
	_, err := env.Run(
		GenerateDeterministicKeys(),
		AddTransportManagers(),
		AddProcManagerAndRouter(),
		AddSetupHandlersEnv(),
	)
	fmt.Printf("env.Run() success: %v\n", err == nil)

	// add rules
	trID := uuid.New()
	expireAt := time.Now().Add(2 * time.Minute)
	rules := []routing.Rule{
		routing.ForwardRule(expireAt, 2, trID),
	}
	routes, err := env.stpHandlers.addRules(rules)
	if err != nil {
		fmt.Printf("error on addRules: %v\n", err)
	}

	deletedRoutes, err := env.stpHandlers.deleteRules(routes)
	if err != nil {
		fmt.Printf("error in deleteRules: %v\n", err)
	}
	fmt.Printf("deletedRoutes, err: %v, %v\n", deletedRoutes, err)

	// Output: env.Run() success: true
	// deletedRoutes, err: [1], <nil>
}

func Example_setupHandlers_loopClosed() {
	env := &TEnv{}
	_, err := env.Run(
		GenerateDeterministicKeys(),
		AddTransportManagers(),
		AddProcManagerAndRouter(),
		AddSetupHandlersEnv(),
	)
	fmt.Printf("env.Run() success: %v\n", err == nil)

	unknownLoopData := setup.LoopData{
		RemotePK:     env.pkSetup,
		RemotePort:   0,
		LocalPort:    0,
		RouteID:      routing.RouteID(0),
		NoiseMessage: []byte{},
	}

	loopClosedErr := env.stpHandlers.loopClosed(unknownLoopData)
	fmt.Printf("loopClosed(unknownLoopData): %v\n", loopClosedErr)

	// Output: env.Run() success: true
	// loopClosed(unknownLoopData): proc not found
}

type handleTestCase struct {
	packetType setup.PacketType
	bodyFunc   CfgStep
}

var handleTestCases = []handleTestCase{
	handleTestCase{
		packetType: setup.PacketAddRules,
		bodyFunc: func(env *TEnv) (testName string, err error) {
			testName = fmt.Sprintf("%v/1", setup.PacketAddRules)
			trID := uuid.New()
			expireAt := time.Now().Add(2 * time.Minute)
			rules := []routing.Rule{
				routing.ForwardRule(expireAt, 2, trID),
			}
			body, err := json.Marshal(rules)
			env.stpHandlers.packetBody = body
			return
		},
	},
	handleTestCase{
		packetType: setup.PacketAddRules,
		bodyFunc: func(env *TEnv) (testName string, err error) {
			testName = fmt.Sprintf("%v/2", setup.PacketAddRules)
			env.stpHandlers.packetBody = []byte("invalid packet body")
			return
		},
	},
	handleTestCase{
		packetType: setup.PacketDeleteRules,
		bodyFunc: func(env *TEnv) (testName string, err error) {
			testName = fmt.Sprintf("%v/1", setup.PacketDeleteRules)
			// add rules
			trID := uuid.New()
			expireAt := time.Now().Add(2 * time.Minute)
			rules := []routing.Rule{
				routing.ForwardRule(expireAt, 2, trID),
			}
			routes, err := env.stpHandlers.addRules(rules)
			if err != nil {
				fmt.Printf("error on addRules: %v\n", err)
			}

			body, err := json.Marshal(routes)
			env.stpHandlers.packetBody = body
			return
		},
	},
	handleTestCase{
		packetType: setup.PacketDeleteRules,
		bodyFunc: func(env *TEnv) (testName string, err error) {
			testName = fmt.Sprintf("%v/2", setup.PacketDeleteRules)
			env.stpHandlers.packetBody = []byte("invalid packet body")
			return
		},
	},
	handleTestCase{
		packetType: setup.PacketConfirmLoop,
		bodyFunc: func(env *TEnv) (testName string, err error) {
			testName = fmt.Sprintf("%v/1", setup.PacketConfirmLoop)
			loopData := setup.LoopData{
				RemotePK:     env.pkLocal,
				RemotePort:   0,
				LocalPort:    0,
				RouteID:      routing.RouteID(0),
				NoiseMessage: []byte{},
			}
			body, err := json.Marshal(loopData)

			env.stpHandlers.packetBody = body
			return
		},
	},
	handleTestCase{
		packetType: setup.PacketConfirmLoop,
		bodyFunc: func(env *TEnv) (testName string, err error) {
			testName = fmt.Sprintf("%v/2", setup.PacketConfirmLoop)
			env.stpHandlers.packetBody = []byte("invalid packet body")
			return
		},
	},
	handleTestCase{
		packetType: setup.PacketLoopClosed,
		bodyFunc: func(env *TEnv) (testName string, err error) {
			testName = fmt.Sprintf("%v/1", setup.PacketLoopClosed)

			unknownLoopData := setup.LoopData{
				RemotePK:     env.pkRemote,
				RemotePort:   0,
				LocalPort:    0,
				RouteID:      routing.RouteID(0),
				NoiseMessage: []byte{},
			}
			body, err := json.Marshal(unknownLoopData)

			env.stpHandlers.packetBody = body
			return

		},
	},
	handleTestCase{
		packetType: setup.PacketLoopClosed,
		bodyFunc: func(env *TEnv) (testName string, err error) {
			testName = fmt.Sprintf("%v/2", setup.PacketLoopClosed)
			env.stpHandlers.packetBody = []byte("invalid packet body")
			return
		},
	},
	handleTestCase{
		packetType: setup.PacketType(42),
		bodyFunc: func(env *TEnv) (testName string, err error) {
			testName = string(setup.PacketType(42))
			env.stpHandlers.packetBody = []byte("invalid packet body")
			return
		},
	},
}

func Example_handle() {
	fmt.Println("Start")
	for _, tc := range handleTestCases {

		env := &TEnv{}
		_, err := env.Run(
			GenerateDeterministicKeys(),
			AddTransportManagers(),
			AddProcManagerAndRouter(),
			AddSetupHandlersEnv(),
		)
		if err != nil {
			fmt.Println(err)
		}

		testName, err := tc.bodyFunc(env)
		if err != nil {
			fmt.Printf("%v\n", err)
		}

		errCh := make(chan error, 1)
		go func() {
			env.stpHandlers.packetType = tc.packetType
			errCh <- env.stpHandlers.handle()
		}()

		pt, data, err := env.sprotoInit.ReadPacket()
		fmt.Printf("Test %v response: %v %v %v\n", testName, pt, string(data), err)

		if err := env.TearDown(); err != nil {
			fmt.Printf("env.TearDown() error: %v\n", err)
		}
	}

	fmt.Println("Finish")

	// Output: Start
	// Test AddRules/1 response: RespSuccess [1] <nil>
	// Test AddRules/2 response: RespFailure "invalid character 'i' looking for beginning of value" <nil>
	// Test DeleteRules/1 response: RespSuccess [1] <nil>
	// Test DeleteRules/2 response: RespFailure "invalid character 'i' looking for beginning of value" <nil>
	// Test ConfirmLoop/1 response: RespFailure "unknown loop" <nil>
	// Test ConfirmLoop/2 response: RespFailure "invalid character 'i' looking for beginning of value" <nil>
	// Test LoopClosed/1 response: RespFailure "proc not found" <nil>
	// Test LoopClosed/2 response: RespFailure "invalid character 'i' looking for beginning of value" <nil>
	// Test * response: RespFailure "unknown foundation packet" <nil>
	// Finish

}
