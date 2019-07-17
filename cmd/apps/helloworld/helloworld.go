/*
simple client server app for skywire visor testing
*/
package main

import (
	"log"
	"os"

	"github.com/skycoin/dmsg/cipher"

	"github.com/skycoin/skywire/pkg/app"
	"github.com/skycoin/skywire/pkg/routing"
)

func main() {
	helloworldApp, err := app.Setup(&app.Config{AppName: "helloworld", AppVersion: "1.0", ProtocolVersion: "0.0.1"})
	if err != nil {
		log.Fatal("Setup failure:", err)
	}
	defer func() {
		if err := helloworldApp.Close(); err != nil {
			log.Println("Failed to close app: ", err)
		}
	}()

	if len(os.Args) == 1 {
		log.Println("listening for incoming connections")
		for {
			conn, err := helloworldApp.Accept()
			if err != nil {
				log.Fatal("Failed to accept conn: ", err)
			}

			log.Println("got new connection from:", conn.RemoteAddr())
			go func() {
				buf := make([]byte, 4)
				if _, err := conn.Read(buf); err != nil {
					log.Println("Failed to read remote data:", err)
				}

				log.Printf("Message from %s: %s", conn.RemoteAddr().String(), string(buf))
				if _, err := conn.Write([]byte("pong")); err != nil {
					log.Println("Failed to write to a remote node:", err)
				}
			}()
		}
	}

	remotePK := cipher.PubKey{}
	if err := remotePK.UnmarshalText([]byte(os.Args[1])); err != nil {
		log.Fatal("Failed to construct PubKey: ", err, os.Args[1])
	}

	conn, err := helloworldApp.Dial(routing.Addr{PubKey: remotePK, Port: 10})
	if err != nil {
		log.Fatal("Failed to open remote conn: ", err)
	}

	if _, err := conn.Write([]byte("ping")); err != nil {
		log.Fatal("Failed to write to a remote node: ", err)
	}

	buf := make([]byte, 4)
	if _, err = conn.Read(buf); err != nil {
		log.Fatal("Failed to read remote data: ", err)
	}

	log.Printf("Message from %s: %s", conn.RemoteAddr().String(), string(buf))
}
