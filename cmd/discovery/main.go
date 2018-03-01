package main

import (
	"flag"
	log "github.com/sirupsen/logrus"
	"github.com/skycoin/net/util"
	"github.com/skycoin/net/util/producer"
	"github.com/skycoin/skycoin/src/util/file"
	"github.com/skycoin/skywire/discovery"
	"os"
	"os/signal"
	"path/filepath"
)

var (
	address  string
	webDir   string
	webPort  string
	seedPath string

	ipDBPath string
	confPath string
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
	flag.StringVar(&confPath, "conf-path", filepath.Join(file.UserHome(), ".skywire", "discovery", "conf.json"), "config file path")
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
	err = producer.Init(confPath)
	if err != nil {
		log.Fatal(err)
	}
	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, os.Interrupt, os.Kill)

	d := discovery.New(seedPath, address, webPort, webDir)
	err = d.Start()
	if err != nil {
		log.Fatal(err)
	}
	log.Debugf("listen on %s", address)
	defer d.Close()

	select {
	case signal := <-osSignal:
		if signal == os.Interrupt {
			log.Debugln("exit by signal Interrupt")
		} else if signal == os.Kill {
			log.Debugln("exit by signal Kill")
		}
	}
}
