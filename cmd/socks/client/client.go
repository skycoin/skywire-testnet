package main

import (
	"flag"

	"os"
	"os/signal"

	ss "github.com/shadowsocks/shadowsocks-go/shadowsocks"
	log "github.com/sirupsen/logrus"
	"github.com/skycoin/net/skycoin-messenger/factory"
	"github.com/skycoin/skywire/app"
)

var (
	nodeAddress  string
	port int
)

func parseFlags() {
	flag.StringVar(&nodeAddress, "node-address", ":5001", "node address to connect")
	flag.IntVar(&port, "p", 9443, "local port")
	flag.Parse()
}

func main() {
	parseFlags()

	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, os.Interrupt, os.Kill)

	a := app.New(false, "socks", "")
	a.AppConnectionInitCallback = func(resp *factory.AppConnResp) {
		config := &ss.Config{
			Password:"123456",
			LocalPort:port,
			ServerPort:resp.Port,
			Server:resp.Host,
		}
		log.Debugf("%#v", config)
		ss.SetDebug(true)
		appmain(config)
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
