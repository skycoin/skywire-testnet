package main

import (
	"flag"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"

	ss "github.com/shadowsocks/shadowsocks-go/shadowsocks"
	log "github.com/sirupsen/logrus"
	"github.com/skycoin/skycoin/src/util/file"
	"github.com/skycoin/skywire/app"
)

var (
	nodeAddress string
	serverPort  int
	// use fixed seed if true
	seed bool
	// path for seed, public key and private key
	seedPath string
	// allow node public keys to connect
	nodeKeys app.NodeKeys
)

func parseFlags() {
	flag.StringVar(&nodeAddress, "node-address", ":5000", "node address to connect")
	flag.IntVar(&serverPort, "p", 28443, "server port")
	flag.BoolVar(&seed, "seed", true, "use fixed seed to connect if true")
	flag.StringVar(&seedPath, "seed-path", filepath.Join(file.UserHome(), ".skywire", "ss", "keys.json"), "path to save seed info")
	flag.Var(&nodeKeys, "node-key", "allow node public keys to connect")
	flag.Parse()
}

func main() {
	parseFlags()

	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, os.Interrupt, os.Kill)

	config = &ss.Config{
		PortPassword: map[string]string{strconv.Itoa(serverPort): "123456"},
	}
	ss.SetDebug(true)
	appmain()
	a := app.New(app.Public, "sockss", ":"+strconv.Itoa(serverPort))
	a.SetAllowNodes(nodeKeys)

	if !seed {
		seedPath = ""
	} else {
		if len(seedPath) < 1 {
			seedPath = filepath.Join(file.UserHome(), ".skywire", "ss", "keys.json")
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
