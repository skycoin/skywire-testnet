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

	err := os.Setenv("TEST_NODE_PK", app.Info().Host.String())
	if err != nil {
		panic(err)
	}
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
			log.Fatal(fmt.Errorf("server conn.Read err: %s", err))
		}
		log.Println("succesfully readed buf => ", string(buf))
	}
}
