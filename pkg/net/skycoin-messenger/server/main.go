package main

import (
	"flag"
	"os"
	"os/signal"

	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/skycoin/skycoin/src/util/file"
	"github.com/skycoin/skywire/pkg/net/skycoin-messenger/factory"
)

var (
	address  string
	seedPath string
)

func parseFlags() {
	flag.StringVar(&address, "address", ":8080", "address to listen on")
	flag.StringVar(&seedPath, "seed-path", filepath.Join(file.UserHome(), ".skyim", "server", "keys.json"), "dir path to save seeds info")
	flag.Parse()
}

func main() {
	parseFlags()

	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, os.Interrupt, os.Kill)

	f := factory.NewMessengerFactory()
	f.SetDefaultSeedConfigPath(seedPath)
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
