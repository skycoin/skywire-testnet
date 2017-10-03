package main

import (
	"flag"

	"os"
	"os/signal"

	log "github.com/sirupsen/logrus"
	"github.com/skycoin/skywire/app"
)

var (
	nodeAddress string
)

func parseFlags() {
	flag.StringVar(&nodeAddress, "node-address", ":5000", "node address to connect")
	flag.Parse()
}

func main() {
	parseFlags()

	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, os.Interrupt, os.Kill)

	a := app.New(true, "skywire_ssh", ":22")
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
