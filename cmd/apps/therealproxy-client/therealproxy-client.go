/*
proxy client app for skywire node
*/
package main

import (
	"flag"
	"log"

	"github.com/skycoin/skywire/internal/therealproxy"
	"github.com/skycoin/skywire/pkg/app"
	"github.com/skycoin/skywire/pkg/cipher"
)

const socksPort = 3

func main() {
	app.Setup("therealproxy-client", "1.0")
	defer app.Close()

	var addr = flag.String("addr", ":1080", "Client address to listen on")
	var serverPK = flag.String("srv", "", "PubKey of the server to connect to")
	flag.Parse()

	if *serverPK == "" {
		log.Fatal("Invalid server PubKey")
	}

	pk := cipher.PubKey{}
	if err := pk.UnmarshalText([]byte(*serverPK)); err != nil {
		log.Fatal("Invalid server PubKey: ", err)
	}

	conn, err := app.Dial(app.LoopAddr{PubKey: pk, Port: uint16(socksPort)})
	if err != nil {
		log.Fatal("Failed to dial to a server: ", err)
	}

	client, err := therealproxy.NewClient(conn)
	if err != nil {
		log.Fatal("Failed to create a new client: ", err)
	}

	log.Fatal(client.ListenAndServe(*addr))
}
