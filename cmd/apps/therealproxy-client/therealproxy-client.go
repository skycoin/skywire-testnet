/*
proxy client app for skywire visor
*/
package main

import (
	"flag"
	"net"
	"time"

	"github.com/SkycoinProject/dmsg/cipher"

	"github.com/SkycoinProject/skywire-mainnet/internal/netutil"
	"github.com/SkycoinProject/skywire-mainnet/internal/therealproxy"
	"github.com/SkycoinProject/skywire-mainnet/pkg/app"
	"github.com/SkycoinProject/skywire-mainnet/pkg/routing"
)

const socksPort = 3

var r = netutil.NewRetrier(time.Second, 0, 1)

func main() {
	log := app.NewLogger("socksproxy-client")
	therealproxy.Log = log.PackageLogger("therealproxy")

	var addr = flag.String("addr", ":1080", "Client address to listen on")
	var serverPK = flag.String("srv", "", "PubKey of the server to connect to")
	flag.Parse()

	config := &app.Config{AppName: "socksproxy-client", AppVersion: "1.0", ProtocolVersion: "0.0.1"}
	socksApp, err := app.Setup(config)
	if err != nil {
		log.Fatal("Setup failure: ", err)
	}
	defer func() {
		if err := socksApp.Close(); err != nil {
			log.Println("Failed to close app:", err)
		}
	}()

	if *serverPK == "" {
		log.Fatal("Invalid server PubKey")
	}

	pk := cipher.PubKey{}
	if err := pk.UnmarshalText([]byte(*serverPK)); err != nil {
		log.Fatal("Invalid server PubKey: ", err)
	}

	var conn net.Conn
	err = r.Do(func() error {
		conn, err = socksApp.Dial(routing.Addr{PubKey: pk, Port: routing.Port(socksPort)})
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
