/*
proxy server app for skywire visor
*/
package main

import (
	"flag"

	"github.com/skycoin/skywire/internal/therealproxy"
	"github.com/skycoin/skywire/pkg/app"
)

func main() {
	log := app.NewLogger("socksproxy")
	therealproxy.Log = log.PackageLogger("therealproxy")

	var passcode = flag.String("passcode", "", "Authorize user against this passcode")
	flag.Parse()

	config := &app.Config{AppName: "socksproxy", AppVersion: "1.0", ProtocolVersion: "0.0.1"}
	socksApp, err := app.Setup(config)
	if err != nil {
		log.Fatal("Setup failure: ", err)
	}
	defer func() {
		if err := socksApp.Close(); err != nil {
			log.Println("Failed to close app:", err)
		}
	}()

	srv, err := therealproxy.NewServer(*passcode, log)
	if err != nil {
		log.Fatal("Failed to create a new server: ", err)
	}

	log.Fatal(srv.Serve(socksApp))
}
