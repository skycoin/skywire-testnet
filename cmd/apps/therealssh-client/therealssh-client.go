/*
ssh client app for skywire node
*/
package main

import (
	"flag"
	"log"
	"net/http"

	ssh "github.com/skycoin/skywire/internal/therealssh"
	"github.com/skycoin/skywire/pkg/app"
)

func main() {
	var rpcAddr = flag.String("rpc", ":2222", "Client RPC address to listen on")
	var debug = flag.Bool("debug", false, "enable debug messages")
	flag.Parse()

	config := &app.Config{AppName: "therealssh-client", AppVersion: "1.0", ProtocolVersion: "0.0.1"}
	sshApp, err := app.Setup(config)
	if err != nil {
		log.Fatal("Setup failure: ", err)
	}
	defer sshApp.Close()

	ssh.Debug = *debug

	rpc, client, err := ssh.NewClient(*rpcAddr, sshApp)
	if err != nil {
		log.Fatal("Client setup failure: ", err)
	}
	defer client.Close()

	if err := http.Serve(rpc, nil); err != nil {
		log.Fatal("Failed to start RPC interface: ", err)
	}
}
