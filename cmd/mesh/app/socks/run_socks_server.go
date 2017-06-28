package main

import (
	"os"
	"strconv"

	"github.com/skycoin/skywire/src/app"
	"github.com/skycoin/skywire/src/messages"
)

func main() {
	args := os.Args
	if len(args) < 6 {
		panic("not sufficient number of args")
	}

	id, nodeAddr, proxyPort, appIdStr, seqStr :=
		args[1], args[2], args[3], args[4], args[5]

	seqInt, err := strconv.Atoi(seqStr)
	if err != nil {
		panic(err)
	}
	if seqInt < 0 {
		panic("negative sequence")
	}

	sequence := uint32(seqInt)

	appIdInt, err := strconv.Atoi(appIdStr)
	if err != nil {
		panic(err)
	}
	if appIdInt < 0 {
		panic("negative sequence")
	}

	appId := uint32(appIdInt)

	socksServer, err := app.NewSocksServer(messages.MakeAppId(id),
		nodeAddr, "0.0.0.0:"+proxyPort)
	if err != nil {
		panic(err)
	}

	socksServer.TalkToViscript(sequence, appId)
}
