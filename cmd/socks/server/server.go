package main

import (
	"flag"

	"os"
	"os/signal"

	"strconv"

	ss "github.com/shadowsocks/shadowsocks-go/shadowsocks"
	log "github.com/sirupsen/logrus"
	"github.com/skycoin/skywire/app"
)

var (
	nodeAddress string
	serverPort  int
)

func parseFlags() {
	flag.StringVar(&nodeAddress, "node-address", ":5000", "node address to connect")
	flag.IntVar(&serverPort, "p", 28443, "server port")
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
	a := app.New(true, "socks", ":"+strconv.Itoa(serverPort))
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
