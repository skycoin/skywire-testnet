/*
ssh server app for skywire node
*/
package main

import (
	"flag"
	"log"

	homedir "github.com/mitchellh/go-homedir"

	ssh "github.com/skycoin/skywire/internal/therealssh"
	"github.com/skycoin/skywire/pkg/app"
)

func main() {
	app.Setup("therealssh", "1.0")
	defer app.Close()

	var authFile = flag.String("auth", "~/.therealssh/authorized_keys", "Auth file location. Should contain one PubKey per line.")
	var debug = flag.Bool("debug", false, "enable debug messages")

	flag.Parse()

	path, err := homedir.Expand(*authFile)
	if err != nil {
		log.Fatal("Failed to resolve auth file path: ", err)
	}

	ssh.Debug = *debug

	auth, err := ssh.NewFileAuthorizer(path)
	if err != nil {
		log.Fatal("Failed to setup Authorizer: ", err)
	}

	server := ssh.NewServer(auth)
	defer server.Close()

	for {
		conn, err := app.Accept()
		if err != nil {
			log.Fatal("failed to receive packet: ", err)
		}

		go func() {
			if err := server.Serve(conn); err != nil {
				log.Println("Failed to serve conn:", err)
			}
		}()
	}
}
