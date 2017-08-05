package main

import (
	"os"

	"os/signal"

	"github.com/skycoin/skywire/apptracker"
	_ "github.com/skycoin/viscript/signal"
)

func main() {
	args := os.Args
	if len(args) < 2 {
		panic("not sufficient number of args")
	}

	listenAddr := args[1]

	appTracker := apptracker.NewAppTracker(listenAddr)
	if appTracker == nil {
		panic("appTracker == nil")
	}
	osSignal := make(chan os.Signal)
	signal.Notify(osSignal, os.Interrupt, os.Kill)
	<-osSignal
}
