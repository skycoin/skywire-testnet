package api

import (
	"encoding/json"
	"github.com/rs/cors"
	log "github.com/sirupsen/logrus"
	"github.com/skycoin/skywire/node"
	"net/http"
	"os"
)

type NodeApi struct {
	address  string
	node     *node.Node
	osSignal chan os.Signal
	srv      *http.Server
}

func New(addr string, node *node.Node, signal chan os.Signal) *NodeApi {
	return &NodeApi{address: addr, node: node, osSignal: signal, srv: &http.Server{Addr: addr}}
}

func (na *NodeApi) Close() error {
	return na.srv.Shutdown(nil)
}
func (na *NodeApi) StartSrv() {
	mux := http.NewServeMux()
	mux.HandleFunc("/node/getTransports", func(w http.ResponseWriter, r *http.Request) {
		js, err := json.Marshal(na.node.GetTransport())
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
	})
	mux.HandleFunc("/node/shutDown", func(w http.ResponseWriter, r *http.Request) {
		na.osSignal <- os.Interrupt
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("true"))
	})
	mux.HandleFunc("/node/restart", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("true"))
	})
	na.srv.Handler = cors.Default().Handler(mux)
	go func() {
		if err := na.srv.ListenAndServe(); err != nil {
			log.Printf("http server: ListenAndServe() error: %s", err)
		}
	}()
	log.Debugf("http server listen on %s", na.address)
}
