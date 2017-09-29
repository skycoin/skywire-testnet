package main

import (
	"flag"

	"os"
	"os/signal"

	ss "github.com/shadowsocks/shadowsocks-go/shadowsocks"
	log "github.com/sirupsen/logrus"
	"github.com/skycoin/net/skycoin-messenger/factory"
	"github.com/skycoin/skywire/app"
	"net"
	"strconv"
)

var (
	nodeAddress   string
	listenAddress string
)

func parseFlags() {
	flag.StringVar(&nodeAddress, "node-address", ":5001", "node address to connect")
	flag.StringVar(&listenAddress, "address", ":9443", "listen address")
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

	a := app.New(false, "socks", "")
	a.AppConnectionInitCallback = func(resp *factory.AppConnResp) {
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
	}
	a.Start(nodeAddress)

	select {
	case signal := <-osSignal:
		if signal == os.Interrupt {
			log.Debugln("exit by signal Interrupt")
		} else if signal == os.Kill {
			log.Debugln("exit by signal Kill")
		}
	}
}
