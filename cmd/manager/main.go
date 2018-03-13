package main

import (
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/skycoin/net/skycoin-messenger/factory"
	"github.com/skycoin/net/skycoin-messenger/monitor"
	"github.com/skycoin/skycoin/src/util/file"
	"github.com/skycoin/skywire/manager"
	"os"
	"os/signal"
	"path/filepath"
)

var (
	address  string
	webDir   string
	webPort  string
	seedPath string

	version bool
)

func parseFlags() {
	var dir = "/src/github.com/skycoin/net/skycoin-messenger/monitor/web/dist-manager"
	flag.StringVar(&webDir, "web-dir", filepath.Join(os.Getenv("GOPATH"), dir), "monitor web page")
	flag.StringVar(&webPort, "web-port", ":8000", "monitor web page port")
	flag.StringVar(&address, "address", ":5998", "address to listen on")
	flag.StringVar(&seedPath, "seed-path", filepath.Join(file.UserHome(), ".skywire", "discovery", "keys.json"), "path to save seed info")
	flag.BoolVar(&version, "v", false, "print current version")
	flag.Parse()
}

func main() {
	parseFlags()
	if version {
		fmt.Println(manager.Version)
		return
	}

	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, os.Interrupt, os.Kill)

	f := factory.NewMessengerFactory()
	defer f.Close()
	f.SetDefaultSeedConfigPath(seedPath)
	f.SetLoggerLevel(factory.DebugLevel)
	f.SetAppVersion(manager.Version)
	err := f.Listen(address)
	log.Debugf("listen on %s", address)
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
	m := monitor.New(f, address, webPort, manager.Tag, manager.Version)
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
