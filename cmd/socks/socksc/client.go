package main

import (
	"flag"
	"fmt"
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

const (
	Version = "1.0.0"
)

var (
	nodeAddress   string
	listenAddress string
	// use fixed seed if true
	seed bool
	// path for seed, public key and private key
	seedPath string
	// connect to node
	nodeKey string
	// connect to app
	appKey string

	discoveryKey string

	version bool
)

func parseFlags() {
	flag.StringVar(&nodeAddress, "node-address", ":5001", "node address to connect")
	flag.StringVar(&listenAddress, "address", ":9443", "listen address")
	flag.BoolVar(&seed, "seed", true, "use fixed seed to connect if true")
	flag.StringVar(&seedPath, "seed-path", filepath.Join(file.UserHome(), ".skywire", "sc", "keys.json"), "path to save seed info")
	flag.StringVar(&nodeKey, "node-key", "", "connect to node key")
	flag.StringVar(&appKey, "app-key", "", "connect to app key")
	flag.StringVar(&discoveryKey, "discovery-key", "", "connect to discovery key")
	flag.BoolVar(&version, "v", false, "print current version")
	flag.Parse()
}

func main() {
	parseFlags()
	if version {
		fmt.Println(Version)
		return
	}

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

	a := app.NewClient(app.Client, "socksc", Version)
	a.AppConnectionInitCallback = func(resp *factory.AppConnResp) *factory.AppFeedback {
		if resp.Failed {
			return &factory.AppFeedback{
				Port:   port,
				Failed: resp.Failed,
				Msg:    resp.Msg,
			}
		}
		config := &ss.Config{
			Password:   "123456",
			LocalPort:  port,
			ServerPort: resp.Port,
			Server:     resp.Host,
		}
		log.Debugf("%#v", resp)
		ss.SetDebug(true)
		go appmain(listenAddress, config)
		return &factory.AppFeedback{
			Port:   port,
			Failed: resp.Failed,
			Msg:    resp.Msg,
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

	err = a.ConnectTo(nodeKey, appKey, discoveryKey)
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
