package utils

import (
	"net/http"
	log "github.com/sirupsen/logrus"
	"github.com/skycoin/skywire/node"
	"os"
	"github.com/skycoin/skycoin/src/cipher"
)

type Utils struct {
	address string
	node *node.Node
	osSignal chan os.Signal
}

var srv *http.Server

func New(addr string,node *node.Node,signal chan os.Signal) *Utils {
	return &Utils{address: addr,node: node,osSignal:signal}
}

func (u *Utils) Close() error {
	return srv.Shutdown(nil)
}
func (u *Utils) StartSrv() {
	srv = &http.Server{Addr: u.address}
	http.HandleFunc("/node/getTransports", func(w http.ResponseWriter, r *http.Request) {
		k := r.FormValue("key")
		key,err := cipher.PubKeyFromHex(k)
		if err != nil {
			return
		}
		u.node.Test(key)
		w.Write([]byte("successful"))
	})
	http.HandleFunc("/node/close", func(w http.ResponseWriter, r *http.Request) {
		u.osSignal <- os.Interrupt
		w.Write([]byte("successful"))
	})
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Printf("http server: ListenAndServe() error: %s", err)
		}
	}()
	log.Debugf("http server listen on %s", u.address)
}

