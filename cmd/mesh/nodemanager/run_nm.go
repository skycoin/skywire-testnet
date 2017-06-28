package main

import (
	"os"
	"strconv"

	network "github.com/skycoin/skywire/nodemanager"
)

func main() {
	args := os.Args
	if len(args) < 4 {
		panic("not sufficient number of args")
	}

	domainName, ctrlAddr, appTrackerAddr := args[1], args[2], args[3]

	seqStr := args[4]
	seqInt, err := strconv.Atoi(seqStr)
	if err != nil {
		panic(err)
	}

	if seqInt < 0 {
		panic("negative sequence")
	}
	sequence := uint32(seqInt)

	appIdStr := args[5]
	appIdInt, err := strconv.Atoi(appIdStr)
	if err != nil {
		panic(err)
	}

	if appIdInt < 0 {
		panic("negative sequence")
	}
	appId := uint32(appIdInt)

	nmConfig := &network.NodeManagerConfig{
		Domain:         domainName,
		CtrlAddr:       ctrlAddr,
		AppTrackerAddr: appTrackerAddr,
		// in the future when RouteManager and Logistics Server will be separated from nodemanager, add these addresses support to this script
	}

	nm, err := network.NewNetwork(nmConfig)
	if err != nil {
		panic(err)
	}

	nm.TalkToViscript(sequence, appId)
}
