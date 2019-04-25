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
	app.Setup("client-test","1.0")
	log.Println("client-test setup called")

	var hostPKS string
	hostPK := cipher.PubKey{}
	log.Println("client-test trying to read host pk env")
	for hostPKS == "" {
		time.Sleep(time.Millisecond*20)
		hostPKS = os.Getenv("test-node-pk")
	}
	log.Println("client-test readed host pk: ", hostPKS)
	err := hostPK.Set(hostPKS)
	if err != nil {
		panic(err)
	}

	var portS string
	log.Println("client-test trying to read port")
	for portS != "" {
		time.Sleep(time.Millisecond*20)
		portS = os.Getenv("server-test-port")
	}
	log.Println("client-test readed port: ", portS)
	port, err := strconv.ParseUint(portS, 10, 16)
	if err != nil {
		panic(err)
	}

	la := app.LoopAddr{
		PubKey: hostPK,
		Port: uint16(port),
	}

	log.Println("client-test: dialing server-test")
	conn, err := app.Dial(la)
	if err != nil {
		panic(err)
	}

	log.Println("connected with node over loop: ", la)
	for {
		_, err = conn.Write([]byte("foo"))
		if err != nil {
			panic(err)
		}
		log.Println("sent message")

		time.Sleep(4*time.Second)
	}
}
