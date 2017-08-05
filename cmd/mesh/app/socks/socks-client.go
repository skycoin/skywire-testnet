package main

import (
	"os"

	"github.com/skycoin/skywire/app"
	"github.com/skycoin/skywire/messages"
	_ "github.com/skycoin/viscript/signal"
)

func main() {
	args := os.Args
	if len(args) < 4 {
		panic("not sufficient number of args")
	}
	id, nodeAddr, proxyPort := args[1], args[2], args[3]

	socksClient, err := app.NewSocksClient(messages.MakeAppId(id), nodeAddr, "0.0.0.0:"+proxyPort)
	if err != nil {
		panic(err)
	}

	socksClient.Listen()
}
