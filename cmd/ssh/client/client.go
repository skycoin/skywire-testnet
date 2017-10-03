package main

import (
	"flag"

	"os"
	"os/signal"

	"net"
	"strconv"

	log "github.com/sirupsen/logrus"
	"github.com/skycoin/net/skycoin-messenger/factory"
	"github.com/skycoin/skywire/app"
)

var (
	nodeAddress string
)

func parseFlags() {
	flag.StringVar(&nodeAddress, "node-address", ":5001", "node address to connect")
	flag.Parse()
}

func main() {
	parseFlags()

	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, os.Interrupt, os.Kill)

	a := app.New(false, "skywire_ssh", "")
	a.AppConnectionInitCallback = func(resp *factory.AppConnResp) {
		log.Infof("please ssh to %s", net.JoinHostPort(resp.Host, strconv.Itoa(resp.Port)))
	}
	a.Start(nodeAddress)

	select {
	case signal := <-osSignal:
		if signal == os.Interrupt {
			log.Debugln("exit by signal Interrupt")
		} else if signal == os.Kill {
			log.Debugln("exit by signal Kill")
		}
	}
}
