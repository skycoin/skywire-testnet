package main

import (
	"flag"
	"github.com/skycoin/skycoin/src/util/file"
	"os"
	"os/signal"
	log "github.com/sirupsen/logrus"
	"github.com/skycoin/net/skycoin-messenger/factory"
	"github.com/skycoin/net/skycoin-messenger/monitor"
	"path/filepath"
)

var (
	address  string
	webDir   string
	webPort  string
	seedPath string
)

func parseFlags() {
	var dir = "/src/github.com/skycoin/net/skycoin-messenger/monitor/web/dist-discovery"
	flag.StringVar(&address, "address", ":5999", "address to listen on")
	flag.StringVar(&webDir, "web-dir", filepath.Join(os.Getenv("GOPATH"), dir), "monitor web page")
	flag.StringVar(&webPort, "web-port", ":8000", "monitor web page port")
	flag.StringVar(&seedPath, "seed-path", filepath.Join(file.UserHome(), ".skywire", "discovery", "keys.json"), "path to save seed info")
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
	m := monitor.New(f, address, webPort, "", "")
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
