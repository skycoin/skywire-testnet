package main

import (
	"os"

	"os/signal"

	network "github.com/skycoin/skywire/nodemanager"
	_ "github.com/skycoin/viscript/signal"
)

func main() {
	args := os.Args
	if len(args) < 4 {
		panic("not sufficient number of args")
	}

	domainName, ctrlAddr, appTrackerAddr := args[1], args[2], args[3]

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
	if nm == nil {
		panic("nm == nil")
	}

	osSignal := make(chan os.Signal)
	signal.Notify(osSignal, os.Interrupt, os.Kill)
	<-osSignal
}
