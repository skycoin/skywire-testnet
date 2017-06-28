package main

import (
	"os"
	"strconv"

	"github.com/skycoin/skywire/app"
	"github.com/skycoin/skywire/messages"
)

func main() {
	args := os.Args
	if len(args) < 6 {
		panic("not sufficient number of args")
	}
	id, nodeAddr, proxyPort, appIdStr, seqStr := args[1], args[2], args[3], args[4], args[5]

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

	vpnClient, err := app.NewVPNClient(messages.MakeAppId(id), nodeAddr, "0.0.0.0:"+proxyPort)
	if err != nil {
		panic(err)
	}

	go vpnClient.Listen()
	vpnClient.TalkToViscript(sequence, appId)
}
