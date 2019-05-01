/*
simple client server app for skywire node testing
*/
package main

import (
	"log"
	"os"

	"github.com/skycoin/skywire/pkg/app"
	"github.com/skycoin/skywire/pkg/cipher"
)

func main() {
	app.Setup("helloworld", "1.0")
	defer app.Close()

	if len(os.Args) == 1 {
		log.Println("listening for incoming connections")
		for {
			conn, err := app.Accept()
			if err != nil {
				log.Fatal("Failed to accept conn: ", err)
			}

			log.Println("got new connection from:", conn.RemoteAddr())
			go func() {
				buf := make([]byte, 4)
				if _, err := conn.Read(buf); err != nil {
					log.Println("Failed to read remote data: ", err)
				}

				log.Printf("Message from %s: %s", conn.RemoteAddr().String(), string(buf))
				if _, err := conn.Write([]byte("pong")); err != nil {
					log.Println("Failed to write to a remote node: ", err)
				}
			}()
		}
	}

	remotePK := cipher.PubKey{}
	if err := remotePK.UnmarshalText([]byte(os.Args[1])); err != nil {
		log.Fatal("Failed to construct PubKey: ", err, os.Args[1])
	}

	conn, err := app.Dial(app.LoopAddr{PubKey: remotePK, Port: 10})
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
