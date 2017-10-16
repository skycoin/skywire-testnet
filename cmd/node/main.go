package main

import (
	"flag"
	"os"
	"os/signal"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/skycoin/skycoin/src/util/file"
	"github.com/skycoin/skywire/node"
)

var (
	discoveryAddresses node.Addresses
	managerAddr        string
	address            string
	// use fixed seed if true
	seed bool
	// path for seed, public key and private key
	seedPath string
)

func parseFlags() {
	flag.StringVar(&address, "address", ":5000", "address to listen on")
	flag.Var(&discoveryAddresses, "discovery-address", "addresses of discovery")
	flag.StringVar(&managerAddr, "manager-address",":5998", "address of node manager")
	flag.BoolVar(&seed, "seed", true, "use fixed seed to connect if true")
	flag.StringVar(&seedPath, "seedPath", filepath.Join(file.UserHome(), ".skywire", "node", "keys.json"), "path to save seed info")
	flag.Parse()
}

func main() {
	parseFlags()

	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, os.Interrupt, os.Kill)

	var n *node.Node
	if !seed {
		n = node.New("")
	} else {
		if len(seedPath) < 1 {
			seedPath = filepath.Join(file.UserHome(), ".skywire", "node", "keys.json")
		}
		n = node.New(seedPath)
	}
	err := n.Start(discoveryAddresses, address)
	if err != nil {
		log.Fatal(err)
	}
	log.Debugf("listen on %s", address)
	err = n.ConnectManager(managerAddr)
	if err != nil {
		log.Fatal(err)
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