package main

import (
	"github.com/skycoin/skywire/pkg/app"
	"github.com/skycoin/skywire/pkg/cipher"
	"log"
	"os"
	"strconv"
	"time"
)

func main() {
	log.Println("client-test started")

	hostPKS := os.Args[1]
	log.Println("client-test readed host pk: ", hostPKS)
	portS := os.Args[2]
	log.Println("client-test readed port: ", portS)

	app.Setup("client-test","1.0")

	var hostPK cipher.PubKey
	err := hostPK.Set(hostPKS)
	if err != nil {
		log.Fatal(err)
	}

	portUint64, err := strconv.ParseUint(portS, 10, 16)
	if err != nil {
		log.Fatal(err)
	}

	la := app.LoopAddr{
		PubKey: hostPK,
		Port: uint16(portUint64),
	}

	log.Println("client-test: dialing server-test")
	time.Sleep(time.Millisecond*100) // wait server to start
	conn, err := app.Dial(la)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("connected with node over loop: ", la)
	_, err = conn.Write([]byte("foo"))
	if err != nil {
		log.Fatal(err)
	}
	log.Println("sent message")
}
