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
	app.Setup("therealssh-client", "1.0")
	defer app.Close()

	var rpcAddr = flag.String("rpc", ":2222", "Client RPC address to listen on")
	var debug = flag.Bool("debug", false, "enable debug messages")
	flag.Parse()

	ssh.Debug = *debug

	rpc, client, err := ssh.NewClient(*rpcAddr, app.Dial)
	if err != nil {
		log.Fatal("Client setup failure: ", err)
	}
	defer client.Close()

	if err := http.Serve(rpc, nil); err != nil {
		log.Fatal("Failed to start RPC interface: ", err)
	}
}
