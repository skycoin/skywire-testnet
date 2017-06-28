package main

import (
	"os"
	"strconv"

	"github.com/skycoin/skywire/src/messages"
	"github.com/skycoin/skywire/src/node"
)

func main() {
	args := os.Args
	if len(args) < 7 {
		panic("not sufficient number of args")
	}
	nodeAddr, nmAddr, connect, appTalkPortStr, hostname, appIdStr, seqStr := args[1], args[2], args[3], args[4], args[5], args[6], args[7]

	seqInt, err := strconv.Atoi(seqStr)
	if err != nil {
		panic(err)
	}
	if seqInt < 0 {
		panic("negative sequence")
	}
	sequence := uint32(seqInt)

	appTalkPort, err := strconv.Atoi(appTalkPortStr)
	if err != nil {
		panic(err)
	}
	if appTalkPort < 0 || appTalkPort > 65535 {
		panic("incorrect app talk port")
	}

	appIdInt, err := strconv.Atoi(appIdStr)
	if err != nil {
		panic(err)
	}
	if appIdInt < 0 {
		panic("negative sequence")
	}
	appId := uint32(appIdInt)

	need_connect := connect == "true"

	var n messages.NodeInterface

	nodeConfig := &node.NodeConfig{
		nodeAddr,
		[]string{nmAddr},
		appTalkPort,
		hostname,
	}

	if need_connect {
		n, err = node.CreateAndConnectNode(nodeConfig)
	} else {
		n, err = node.CreateNode(nodeConfig)
	}
	if err != nil {
		panic(err)
	}

	n.TalkToViscript(sequence, appId)
}
