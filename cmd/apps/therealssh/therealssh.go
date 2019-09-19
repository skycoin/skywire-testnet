/*
ssh server app for skywire visor
*/
package main

import (
	"flag"

	"github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	"github.com/SkycoinProject/skycoin/src/util/logging"

	"github.com/SkycoinProject/skywire-mainnet/pkg/app"
	ssh "github.com/SkycoinProject/skywire-mainnet/pkg/therealssh"
)

var log *logging.MasterLogger

func main() {
	log = app.NewLogger("SSH")
	ssh.Log = log.PackageLogger("therealssh")

	var authFile = flag.String("auth", "~/.therealssh/authorized_keys", "Auth file location. Should contain one PubKey per line.")
	var debug = flag.Bool("debug", false, "enable debug messages")

	flag.Parse()

	config := &app.Config{AppName: "SSH", AppVersion: "1.0", ProtocolVersion: "0.0.1"}
	sshApp, err := app.Setup(config)
	if err != nil {
		log.Fatal("Setup failure: ", err)
	}
	defer func() {
		if err := sshApp.Close(); err != nil {
			log.Println("Failed to close app:", err)
		}
	}()

	path, err := homedir.Expand(*authFile)
	if err != nil {
		log.Fatal("Failed to resolve auth file path: ", err)
	}

	ssh.Debug = *debug
	if !ssh.Debug {
		logging.SetLevel(logrus.InfoLevel)
	}

	auth, err := ssh.NewFileAuthorizer(path)
	if err != nil {
		log.Fatal("Failed to setup Authorizer: ", err)
	}

	server := ssh.NewServer(auth, log)
	defer func() {
		if err := server.Close(); err != nil {
			log.Println("Failed to close server:", err)
		}
	}()

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
