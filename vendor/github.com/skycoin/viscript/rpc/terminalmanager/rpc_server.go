package terminalmanager

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
)

const DefaultPort = "7777"

type RPC struct {
}

func NewRPC() *RPC {
	newRPC := &RPC{}
	return newRPC
}

func (r *RPC) Serve() {
	port := os.Getenv("TERMINAL_RPC_PORT")
	if port == "" {
		log.Println("No TERMINAL_RPC_PORT environmental variable is found, assigning default port value:", DefaultPort)
		port = DefaultPort
	}

	terminalManager := newTerminalManager()
	receiver := new(RPCReceiver)
	receiver.TerminalManager = terminalManager
	err := rpc.Register(receiver)
	if err != nil {
		panic(err)
	}

	rpc.HandleHTTP()
	l, err := net.Listen("tcp", ":"+port)
	if err != nil {
		panic(err)
	}

	log.Println("Serving RPC on port", port, "\n\n")
	err = http.Serve(l, nil)
	if err != nil {
		panic(err)
	}
}
