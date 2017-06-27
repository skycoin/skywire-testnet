package main

import (
	"github.com/skycoin/skywire/src/nodemanager"
)

func main() {
	rpcInstance := nodemanager.NewRPC()
	rpcInstance.Serve()
}
