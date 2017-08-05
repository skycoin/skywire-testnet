package main

import (
	"os"
	"strconv"

	"os/signal"

	"github.com/skycoin/skywire/messages"
	"github.com/skycoin/skywire/node"
	_ "github.com/skycoin/viscript/signal"
)

func main() {
	args := os.Args
	if len(args) < 5 {
		panic("not sufficient number of args")
	}
	nodeAddr, nmAddr, connect, appTalkPortStr := args[1], args[2], args[3], args[4]

	appTalkPort, err := strconv.Atoi(appTalkPortStr)
	if err != nil {
		panic(err)
	}
	if appTalkPort < 0 || appTalkPort > 65535 {
		panic("incorrect app talk port")
	}

	need_connect := connect == "true"

	var n messages.NodeInterface

	nodeConfig := &node.NodeConfig{
		nodeAddr,
		[]string{nmAddr},
		appTalkPort,
	}

	if need_connect {
		n, err = node.CreateAndConnectNode(nodeConfig)
	} else {
		n, err = node.CreateNode(nodeConfig)
	}
	if err != nil {
		panic(err)
	}
	if n == nil {
		panic("n == nil")
	}
	osSignal := make(chan os.Signal)
	signal.Notify(osSignal, os.Interrupt, os.Kill)
	<-osSignal
}
