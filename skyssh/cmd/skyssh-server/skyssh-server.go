/*
ssh server app for skywire node
*/
package main

import (
	"flag"
	"log"

	homedir "github.com/mitchellh/go-homedir"

	"github.com/skycoin/skywire/pkg/app"
	ssh "github.com/skycoin/skywire/skyssh/internal/therealssh"
)

func main() {
	var authFile = flag.String("auth", "~/.server/authorized_keys", "Auth file location. Should contain one PubKey per line.")
	var debug = flag.Bool("debug", false, "enable debug messages")

	flag.Parse()

	config := &app.Config{AppName: "skyssh-server", AppVersion: "1.0", ProtocolVersion: "0.0.1"}
	sshApp, err := app.Setup(config)
	if err != nil {
		log.Fatal("Setup failure: ", err)
	}
	defer sshApp.Close()

	path, err := homedir.Expand(*authFile)
	if err != nil {
		log.Fatal("Failed to resolve auth file path: ", err)
	}

	ssh.Debug = *debug

	auth, err := ssh.NewFileAuthorizer(path)
	if err != nil {
		log.Fatal("Failed to setup Authoriser: ", err)
	}

	server := ssh.NewServer(auth)
	defer server.Close()

	for {
		conn, err := sshApp.Accept()
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
