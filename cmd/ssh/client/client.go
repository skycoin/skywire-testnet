package main

import (
	"flag"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"

	log "github.com/sirupsen/logrus"
	"github.com/skycoin/net/skycoin-messenger/factory"
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
	flag.StringVar(&nodeAddress, "node-address", ":5001", "node address to connect")
	flag.BoolVar(&seed, "seed", true, "use fixed seed to connect if true")
	flag.StringVar(&seedPath, "seedPath", "", "path to save seed info(default:$HOME/.skywire/ssh/keys.json)")
	flag.Parse()
}

func main() {
	parseFlags()

	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, os.Interrupt, os.Kill)

	a := app.New(false, "skywire_ssh", "")
	a.AppConnectionInitCallback = func(resp *factory.AppConnResp) {
		log.Infof("please ssh to %s", net.JoinHostPort(resp.Host, strconv.Itoa(resp.Port)))
	}
	if !seed {
		seedPath = ""
	} else {
		if len(seedPath) < 1 {
			seedPath = filepath.Join(file.UserHome(), ".skywire", "ssh", "keys.json")
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
