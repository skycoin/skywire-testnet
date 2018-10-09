package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/skycoin/skycoin/src/util/browser"
	"github.com/skycoin/skycoin/src/util/file"
	"github.com/skycoin/skywire/pkg/net/skycoin-messenger/websocket"
	"github.com/skycoin/skywire/pkg/net/skycoin-messenger/websocket/data"
)

var (
	webDir           string
	webSocketAddress string
	openBrowser      bool
	// dir path for seeds, public key and private key
	seedPath string
)

func parseFlags() {
	flag.StringVar(&webDir, "web-dir", "../web/dist", "directory of web files")
	flag.StringVar(&webSocketAddress, "websocket-address", "localhost:8082", "websocket address to listen on")
	flag.BoolVar(&openBrowser, "open-browser", false, "whether to open browser")
	flag.StringVar(&seedPath, "seed-path", filepath.Join(file.UserHome(), ".skyim", "account"), "dir path to save seeds info")
	flag.Parse()
}

func main() {
	parseFlags()

	if len(seedPath) < 1 {
		seedPath = filepath.Join(file.UserHome(), ".skyim", "account")
	}
	data.InitData(seedPath)

	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, os.Interrupt, os.Kill)

	log.Debug("listening web")
	http.Handle("/", http.FileServer(http.Dir(webDir)))
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		websocket.ServeWs(w, r)
	})
	ln, err := net.Listen("tcp", webSocketAddress)
	if err != nil {
		log.Error("net.Listen: ", err)
		os.Exit(1)
	}

	if openBrowser {
		go func() {
			browser.Open(fmt.Sprintf("http://%s", webSocketAddress))
		}()
	}
	go func() {
		err := http.Serve(ln, http.DefaultServeMux)
		if err != nil {
			log.Error("http.Serve: ", err)
			os.Exit(1)
		}
	}()

	select {
	case signal := <-osSignal:
		if signal == os.Interrupt {
			log.Debugln("exit by signal Interrupt")
		} else if signal == os.Kill {
			log.Debugln("exit by signal Kill")
		}
	}

}
