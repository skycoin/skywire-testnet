/*
proxy client app for skywire visor
*/
package main

import (
	"flag"
	"log"
	"net"
	"time"

	"github.com/skycoin/skywire/internal/netutil"

	"github.com/skycoin/skywire/internal/therealproxy"
	"github.com/skycoin/skywire/pkg/app"
	"github.com/skycoin/skywire/pkg/cipher"
)

const socksPort = 3

var r = netutil.NewRetrier(time.Second, 0, 1)

func main() {
	var addr = flag.String("addr", ":1080", "Client address to listen on")
	var serverPK = flag.String("srv", "", "PubKey of the server to connect to")
	flag.Parse()

	config := &app.Config{AppName: "socksproxy-client", AppVersion: "1.0", ProtocolVersion: "0.0.1"}
	socksApp, err := app.Setup(config)
	if err != nil {
		log.Fatal("Setup failure: ", err)
	}
	defer socksApp.Close()

	if *serverPK == "" {
		log.Fatal("Invalid server PubKey")
	}

	pk := cipher.PubKey{}
	if err := pk.UnmarshalText([]byte(*serverPK)); err != nil {
		log.Fatal("Invalid server PubKey: ", err)
	}

	var conn net.Conn
	err = r.Do(func() error {
		conn, err = socksApp.Dial(&app.Addr{PubKey: pk, Port: uint16(socksPort)})
		return err
	})
	if err != nil {
		log.Fatal("Failed to dial to a server: ", err)
	}

	log.Printf("Connected to %v\n", pk)

	client, err := therealproxy.NewClient(conn)
	if err != nil {
		log.Fatal("Failed to create a new client: ", err)
	}

	log.Printf("Serving  %v\n", addr)

	log.Fatal(client.ListenAndServe(*addr))
}
