package main

import (
	"os"
	"strconv"

	"github.com/skycoin/skywire/app"
	"github.com/skycoin/skywire/messages"
)

func main() {
	args := os.Args
	if len(args) < 5 {
		panic("not sufficient number of args")
	}
	id, nodeAddr, appIdStr, seqStr := args[1], args[2], args[3], args[4]

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

	vpnServer, err := app.NewVPNServer(messages.MakeAppId(id), nodeAddr)
	if err != nil {
		panic(err)
	}

	vpnServer.TalkToViscript(sequence, appId)
}
