/*
proxy server app for skywire node
*/
package main

import (
	"flag"
	"log"

	"github.com/skycoin/skywire/internal/therealproxy"
	"github.com/skycoin/skywire/pkg/app"
)

func main() {
	app.Setup("therealproxy", "1.0")
	defer app.Close()

	var passcode = flag.String("passcode", "", "Authorise user against this passcode")
	flag.Parse()

	srv, err := therealproxy.NewServer(*passcode)
	if err != nil {
		log.Fatal("Failed to create a new server: ", err)
	}

	log.Fatal(srv.Serve(new(app.Listener)))
}
