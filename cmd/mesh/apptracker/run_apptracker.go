package main

import (
	"os"
	"strconv"

	"github.com/skycoin/skywire/src/apptracker"
)

func main() {
	args := os.Args
	if len(args) < 4 {
		panic("not sufficient number of args")
	}

	listenAddr := args[1]

	seqStr := args[2]
	seqInt, err := strconv.Atoi(seqStr)
	if err != nil {
		panic(err)
	}

	if seqInt < 0 {
		panic("negative sequence")
	}
	sequence := uint32(seqInt)

	appIdStr := args[3]
	appIdInt, err := strconv.Atoi(appIdStr)
	if err != nil {
		panic(err)
	}

	if appIdInt < 0 {
		panic("negative sequence")
	}
	appId := uint32(appIdInt)

	appTracker := apptracker.NewAppTracker(listenAddr)
	if err != nil {
		panic(err)
	}

	appTracker.TalkToViscript(sequence, appId)
}
