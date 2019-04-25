package main

import (
	"fmt"
	"github.com/skycoin/skywire/pkg/app"
	"log"
	"os"
)

func main() {
	log.Println("server-test started")
	app.Setup("server-test", "1.0")
	defer app.Close()


	os.Setenv("test-node-pk", app.Info().Host.String())

	port := conn.LocalAddr().(*app.LoopAddr).Port
	os.Setenv("server-test-port", fmt.Sprint(port))
	log.Println("listening in: ", app.Info().Host, " : ", port)

	log.Println("server-test: os env are set")
	log.Println("server-test: calling app.Accept")
	conn, err := app.Accept()
	if err != nil {
		panic(fmt.Errorf("server app.Accept err: %s", err))
	}
	log.Println("server-test: accepted connection")
	for {
		buf := make([]byte, 4)
		_, err = conn.Read(buf)
		if err != nil {
			panic(fmt.Errorf("server conn.Read err: %s", err))
		}
		log.Println("succesfully readed buf => ", string(buf))
	}
}
