package main

import (
	"flag"
	"os"
	"os/signal"

	log "github.com/sirupsen/logrus"
	"github.com/skycoin/net/skycoin-messenger/factory"
)

var (
	address string
)

func parseFlags() {
	flag.StringVar(&address, "address", ":8080", "address to listen on")
	flag.Parse()
}

func main() {
	parseFlags()

	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, os.Interrupt, os.Kill)

	f := factory.NewMessengerFactory()
	f.SetLoggerLevel(factory.DebugLevel)
	err := f.Listen(address)
	log.Debugf("listen on %s", address)
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	select {
	case signal := <-osSignal:
		if signal == os.Interrupt {
			log.Debugln("exit by signal Interrupt")
		} else if signal == os.Kill {
			log.Debugln("exit by signal Kill")
		}
	}

}
