package main

import (
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/skycoin/skycoin/src/util/file"
	"github.com/skycoin/skywire/app"
	"os"
	"os/signal"
	"path/filepath"
)

var (
	nodeAddress string
	// use fixed seed if true
	seed bool
	// path for seed, public key and private key
	seedPath string
	// allow node public keys to connect
	nodeKeys NodeKeys
)

type NodeKeys []string

func (keys *NodeKeys) String() string {
	return fmt.Sprintf("%v", []string(*keys))
}

func (keys *NodeKeys) Set(key string) error {
	*keys = append(*keys, key)
	return nil
}

func parseFlags() {
	flag.StringVar(&nodeAddress, "node-address", ":5000", "node address to connect")
	flag.BoolVar(&seed, "seed", true, "use fixed seed to connect if true")
	flag.StringVar(&seedPath, "seed-path", filepath.Join(file.UserHome(), ".skywire", "sshs", "keys.json"), "path to save seed info")
	flag.Var(&nodeKeys, "node-key", "allow node public keys to connect")
	flag.Parse()
}

func main() {
	parseFlags()

	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, os.Interrupt, os.Kill)

	a := app.New(app.Private, "sshs", ":22")
	a.SetAllowNodes(nodeKeys)
	if !seed {
		seedPath = ""
	} else {
		if len(seedPath) < 1 {
			seedPath = filepath.Join(file.UserHome(), ".skywire", "sshs", "keys.json")
		}
	}
	err := a.Start(nodeAddress, seedPath)
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
