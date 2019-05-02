package router

import (
	"fmt"
)

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
