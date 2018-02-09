package main

import (
	"flag"
	log "github.com/sirupsen/logrus"
	"github.com/skycoin/net/skycoin-messenger/factory"
	"github.com/skycoin/net/skycoin-messenger/monitor"
	"github.com/skycoin/skycoin/src/util/file"
	"os"
	"os/signal"
	"path/filepath"
	"github.com/skycoin/net/util"
)

var (
	address  string
	webDir   string
	webPort  string
	seedPath string

	ipDBPath string
)

func parseFlags() {
	var dir = "/src/github.com/skycoin/net/skycoin-messenger/monitor/web/dist-discovery"
	flag.StringVar(&address, "address", ":5999", "address to listen on")
	flag.StringVar(&webDir, "web-dir", filepath.Join(os.Getenv("GOPATH"), dir), "monitor web page")
	flag.StringVar(&webPort, "web-port", ":8000", "monitor web page port")
	flag.StringVar(&seedPath, "seed-path", filepath.Join(file.UserHome(), ".skywire", "discovery", "keys.json"), "path to save seed info")
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	flag.StringVar(&ipDBPath, "ipdb-path", filepath.Join(dir, "ip.db"), "ip db file path")
	flag.Parse()
}

func main() {
	parseFlags()

	var err error

	err = util.IPLocator.Init(ipDBPath)
	if err != nil {
		log.Fatal(err)
	}
	defer util.IPLocator.Close()

	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, os.Interrupt, os.Kill)

	f := factory.NewMessengerFactory()
	defer f.Close()
	f.SetDefaultSeedConfigPath(seedPath)
	f.SetLoggerLevel(factory.DebugLevel)
	err = f.Listen(address)
	log.Debugf("listen on %s", address)
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
	m := monitor.New(f, address, webPort, "", "")
	m.Start(webDir)
	defer m.Close()
	select {
	case signal := <-osSignal:
		if signal == os.Interrupt {
			log.Debugln("exit by signal Interrupt")
		} else if signal == os.Kill {
			log.Debugln("exit by signal Kill")
		}
	}
}
