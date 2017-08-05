package main

import (
	"os"

	"os/signal"

	"github.com/skycoin/skywire/app"
	"github.com/skycoin/skywire/messages"
	_ "github.com/skycoin/viscript/signal"
)

func main() {
	args := os.Args
	if len(args) < 3 {
		panic("not sufficient number of args")
	}
	id, nodeAddr := args[1], args[2]

	vpnServer, err := app.NewVPNServer(messages.MakeAppId(id), nodeAddr)
	if err != nil {
		panic(err)
	}
	if vpnServer == nil {
		panic("vpnServer == nil")
	}
	osSignal := make(chan os.Signal)
	signal.Notify(osSignal, os.Interrupt, os.Kill)
	<-osSignal
}
