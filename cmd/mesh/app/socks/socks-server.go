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
	if len(args) < 4 {
		panic("not sufficient number of args")
	}

	id, nodeAddr, proxyPort :=
		args[1], args[2], args[3]

	socksServer, err := app.NewSocksServer(messages.MakeAppId(id),
		nodeAddr, "0.0.0.0:"+proxyPort)
	if err != nil {
		panic(err)
	}
	if socksServer == nil {
		panic("socksServer == nil")
	}

	osSignal := make(chan os.Signal)
	signal.Notify(osSignal, os.Interrupt, os.Kill)
	<-osSignal
}
