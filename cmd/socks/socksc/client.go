package main

import (
	"flag"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"

	ss "github.com/shadowsocks/shadowsocks-go/shadowsocks"
	log "github.com/sirupsen/logrus"
	"github.com/skycoin/net/skycoin-messenger/factory"
	"github.com/skycoin/skycoin/src/util/file"
	"github.com/skycoin/skywire/app"
)

var (
	nodeAddress   string
	listenAddress string
	// use fixed seed if true
	seed bool
	// path for seed, public key and private key
	seedPath string
)

func parseFlags() {
	flag.StringVar(&nodeAddress, "node-address", ":5001", "node address to connect")
	flag.StringVar(&listenAddress, "address", ":9443", "listen address")
	flag.BoolVar(&seed, "seed", true, "use fixed seed to connect if true")
	flag.StringVar(&seedPath, "seed-path", filepath.Join(file.UserHome(), ".skywire", "sc", "keys.json"), "path to save seed info")
	flag.Parse()
}

func main() {
	parseFlags()

	_, p, err := net.SplitHostPort(listenAddress)
	if err != nil {
		log.Error("invalid listen address")
		os.Exit(-1)
	}
	port, err := strconv.Atoi(p)
	if err != nil {
		log.Error("invalid listen address")
		os.Exit(-1)
	}

	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, os.Interrupt, os.Kill)

	a := app.New(app.Client, "socks", "")
	a.AppConnectionInitCallback = func(resp *factory.AppConnResp) *factory.AppFeedback {
		config := &ss.Config{
			Password:   "123456",
			LocalPort:  port,
			ServerPort: resp.Port,
			Server:     resp.Host,
		}
		log.Debugf("%#v", config)
		ss.SetDebug(true)
		appmain(listenAddress, config)
		log.Debugln("appmain")
		return &factory.AppFeedback{
			Port: port,
		}
	}

	if !seed {
		seedPath = ""
	} else {
		if len(seedPath) < 1 {
			seedPath = filepath.Join(file.UserHome(), ".skywire", "sc", "keys.json")
		}
	}
	err = a.Start(nodeAddress, seedPath)
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
