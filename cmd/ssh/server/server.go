package main

import (
	"flag"
	"os"
	"os/signal"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/skycoin/skycoin/src/util/file"
	"github.com/skycoin/skywire/app"
)

var (
	nodeAddress string
	// use fixed seed if true
	seed bool
	// path for seed, public key and private key
	seedPath string
)

func parseFlags() {
	flag.StringVar(&nodeAddress, "node-address", ":5000", "node address to connect")
	flag.BoolVar(&seed, "seed", true, "use fixed seed to connect if true")
	flag.StringVar(&seedPath, "seedPath", "", "path to save seed info(default:$HOME/.skywire/sshd/keys.json)")
	flag.Parse()
}

func main() {
	parseFlags()

	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, os.Interrupt, os.Kill)

	a := app.New(true, "skywire_ssh", ":22")
	if !seed {
		seedPath = ""
	} else {
		if len(seedPath) < 1 {
			seedPath = filepath.Join(file.UserHome(), ".skywire", "sshd", "keys.json")
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
