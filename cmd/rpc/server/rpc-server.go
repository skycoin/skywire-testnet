package main

import (
	"github.com/skycoin/skywire/nodemanager"
)

func main() {
	rpcInstance := nodemanager.NewRPC()
	rpcInstance.Serve()
}
