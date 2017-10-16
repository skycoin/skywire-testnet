package main

import (
	"flag"

	"os"
	"os/signal"

	log "github.com/sirupsen/logrus"
	"github.com/skycoin/net/skycoin-messenger/factory"
	"path/filepath"
	"github.com/skycoin/net/skycoin-messenger/monitor"
)

var (
	address string
	webDir  string
	webPort string
)

func parseFlags() {
	var dir = "/src/github.com/skycoin/net/skycoin-messenger/monitor/web/dist"
	flag.StringVar(&webDir, "webDir", filepath.Join(os.Getenv("GOPATH"), dir), "monitor web page")
	flag.StringVar(&webPort, "webPort", ":4998", "monitor web page port")
	flag.StringVar(&address, "address", ":8000", "address to listen on")
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
	m := monitor.New(f, webPort)
	m.Start(webDir)
	select {
	case signal := <-osSignal:
		if signal == os.Interrupt {
			log.Debugln("exit by signal Interrupt")
		} else if signal == os.Kill {
			log.Debugln("exit by signal Kill")
		}
		m.Close()
	}
}
