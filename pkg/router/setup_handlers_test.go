package router

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/skycoin/skywire/pkg/cipher"
	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/setup"
)

func Example_setupHandlers_reject() {
	env, err := makeMockEnv()
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
	env, err := makeMockEnv()
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
	env, err := makeMockEnv()
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

	// Output: routeId, err: [2], <nil>
}

func Example_setupHandlers_deleteRules() {
	env, err := makeMockEnv()
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

	// Output: deletedRoutes, err: [2], <nil>
}

func Example_setupHandlers_confirmLoop() {
	env, err := makeMockEnv()
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
	defer env.TearDown()

	// Use confirmLoopFunc
	// TODO(alexyu): test for known loops
	pk, _, _ := cipher.GenerateDeterministicKeyPair([]byte("loopData")) //nolint: errcheck

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
	env, err := makeMockEnv()
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
	defer env.TearDown()

	// Use loopClosedFunc
	// TODO(alexyu): test with known loops
	pk, _, _ := cipher.GenerateDeterministicKeyPair([]byte("loopData")) //nolint: errcheck
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
	packetType setup.PacketType
	bodyFunc   func(*mockEnv) (*mockEnv, error)
}

var handleTestCases = []handleTestCase{
	handleTestCase{
		packetType: setup.PacketAddRules,
		bodyFunc: func(env *mockEnv) (*mockEnv, error) {

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
		bodyFunc: func(env *mockEnv) (*mockEnv, error) {
			env.sh.packetBody = []byte("invalid packet body")
			return env, nil
		},
	},
	handleTestCase{
		packetType: setup.PacketDeleteRules,
		bodyFunc: func(env *mockEnv) (*mockEnv, error) {
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
		bodyFunc: func(env *mockEnv) (*mockEnv, error) {
			env.sh.packetBody = []byte("invalid packet body")
			return env, nil
		},
	},
	handleTestCase{
		packetType: setup.PacketConfirmLoop,
		bodyFunc: func(env *mockEnv) (*mockEnv, error) {
			pk, _, _ := cipher.GenerateDeterministicKeyPair([]byte("loopData")) // nolint: errcheck

			// TODO(alexyu): test with known loop
			unknownLoopData := setup.LoopData{
				RemotePK:     pk,
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
		packetType: setup.PacketConfirmLoop,
		bodyFunc: func(env *mockEnv) (*mockEnv, error) {
			env.sh.packetBody = []byte("invalid packet body")
			return env, nil
		},
	},
	handleTestCase{
		packetType: setup.PacketLoopClosed,
		bodyFunc: func(env *mockEnv) (*mockEnv, error) {
			pk, _, _ := cipher.GenerateDeterministicKeyPair([]byte("loopData")) // nolint: errcheck

			// TODO(alexyu): test with known loop
			unknownLoopData := setup.LoopData{
				RemotePK:     pk,
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
		bodyFunc: func(env *mockEnv) (*mockEnv, error) {
			env.sh.packetBody = []byte("invalid packet body")
			return env, nil
		},
	},
	handleTestCase{
		packetType: setup.PacketType(42),
		bodyFunc: func(env *mockEnv) (*mockEnv, error) {
			env.sh.packetBody = []byte("invalid packet body")
			return env, nil
		},
	},
}

func Example_handle() {
	fmt.Println("Start")
	for _, tc := range handleTestCases {
		env, err := makeMockEnv()
		if err != nil {
			fmt.Printf("makeMockEnv: %v\n", err)
		}

		env, err = tc.bodyFunc(env)
		if err != nil {
			fmt.Printf("%v\n", err)
		}

		errCh := make(chan error, 1)
		go func() {
			env.sh.packetType = tc.packetType
			errCh <- env.sh.handle()
		}()

		sprotoInit := setup.NewSetupProtocol(env.connInit)
		pt, data, err := sprotoInit.ReadPacket()
		fmt.Printf("handle %v  success: %v\n", tc.packetType, <-errCh == nil)
		fmt.Printf("response: %v %v %v\n", pt, string(data), err)

		env.TearDown()
	}
	fmt.Println("Finish")

	// Output: Start
	// handle AddRules  success: true
	// response: RespSuccess [2] <nil>
	// handle AddRules  success: true
	// response: RespFailure "invalid character 'i' looking for beginning of value" <nil>
	// handle DeleteRules  success: true
	// response: RespSuccess [2] <nil>
	// handle DeleteRules  success: true
	// response: RespFailure "invalid character 'i' looking for beginning of value" <nil>
	// handle ConfirmLoop  success: true
	// response: RespFailure "unknown loop" <nil>
	// handle ConfirmLoop  success: true
	// response: RespFailure "invalid character 'i' looking for beginning of value" <nil>
	// handle LoopClosed  success: true
	// response: RespFailure "proc not found" <nil>
	// handle LoopClosed  success: true
	// response: RespFailure "invalid character 'i' looking for beginning of value" <nil>
	// handle Unknown(42)  success: true
	// response: RespFailure "unknown foundation packet" <nil>
	// Finish

}
