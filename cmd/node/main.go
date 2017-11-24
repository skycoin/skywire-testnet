package main

import (
	"flag"
	log "github.com/sirupsen/logrus"
	"github.com/skycoin/skycoin/src/util/file"
	"github.com/skycoin/skywire/node"
	"github.com/skycoin/skywire/node/api"
	"os"
	"os/signal"
	"path/filepath"
)

var (
	config api.Config
)

func parseFlags() {
	flag.StringVar(&config.Address, "address", ":5000", "address to listen on")
	flag.Var(&config.DiscoveryAddresses, "discovery-address", "addresses of discovery")
	flag.BoolVar(&config.ConnectManager, "connect-manager", true, "connect to manager if true")
	flag.StringVar(&config.ManagerAddr, "manager-address", ":5998", "address of node manager")
	flag.StringVar(&config.ManagerWeb, "manager-web", ":8000", "address of node manager")
	flag.BoolVar(&config.Seed, "seed", true, "use fixed seed to connect if true")
	flag.StringVar(&config.SeedPath, "seed-path", filepath.Join(file.UserHome(), ".skywire", "node", "keys.json"), "path to save seed info")
	flag.StringVar(&config.WebPort, "web-port", ":6001", "monitor web page port")
	flag.Parse()
}

func main() {
	parseFlags()

	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, os.Interrupt, os.Kill)

	var n *node.Node
	if !config.Seed {
		n = node.New("", config.WebPort)
	} else {
		if len(config.SeedPath) < 1 {
			config.SeedPath = filepath.Join(file.UserHome(), ".skywire", "node", "keys.json")
		}
		n = node.New(config.SeedPath, config.WebPort)
	}
	err := n.Start(config.DiscoveryAddresses, config.Address)
	if err != nil {
		log.Error(err)
	}
	log.Debugf("listen on %s", config.Address)
	var na *api.NodeApi
	if config.ConnectManager {
		err = n.ConnectManager(config.ManagerAddr)
		if err != nil {
			log.Error(err)
		}
		na = api.New(config.WebPort, n, config, osSignal)
		na.StartSrv()
	}
	select {
	case signal := <-osSignal:
		if signal == os.Interrupt {
			log.Debugln("exit by signal Interrupt")
		} else if signal == os.Kill {
			log.Debugln("exit by signal Kill")
		}
		if na != nil {
			na.Close()
		}
	}
}
