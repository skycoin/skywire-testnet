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
	discoveryAddresses node.Addresses
	connectManager     bool
	managerAddr        string
	address            string
	// use fixed seed if true
	seed bool
	// path for seed, public key and private key
	seedPath string
	webPort  string
)

func parseFlags() {
	flag.StringVar(&address, "address", ":5000", "address to listen on")
	flag.Var(&discoveryAddresses, "discovery-address", "addresses of discovery")
	flag.BoolVar(&connectManager, "connect-manager", true, "connect to manager if true")
	flag.StringVar(&managerAddr, "manager-address", ":5998", "address of node manager")
	flag.BoolVar(&seed, "seed", true, "use fixed seed to connect if true")
	flag.StringVar(&seedPath, "seed-path", filepath.Join(file.UserHome(), ".skywire", "node", "keys.json"), "path to save seed info")
	flag.StringVar(&webPort, "web-port", ":6001", "monitor web page port")
	flag.Parse()
}

func main() {
	parseFlags()

	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, os.Interrupt, os.Kill)

	var n *node.Node
	if !seed {
		n = node.New("", webPort)
	} else {
		if len(seedPath) < 1 {
			seedPath = filepath.Join(file.UserHome(), ".skywire", "node", "keys.json")
		}
		n = node.New(seedPath, webPort)
	}
	err := n.Start(discoveryAddresses, address)
	if err != nil {
		log.Error(err)
	}
	log.Debugf("listen on %s", address)
	var na *api.NodeApi
	if connectManager {
		err = n.ConnectManager(managerAddr)
		if err != nil {
			log.Error(err)
		}
		na = api.New(webPort, n, osSignal)
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
