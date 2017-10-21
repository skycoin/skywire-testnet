package api

import (
	"context"
	"encoding/json"
	"github.com/rs/cors"
	log "github.com/sirupsen/logrus"
	"github.com/skycoin/skywire/node"
	"net/http"
	"os"
	"os/exec"
	"sync"
)

type NodeApi struct {
	address  string
	node     *node.Node
	osSignal chan os.Signal
	srv      *http.Server

	sshsCxt      context.Context
	sshsCancel   context.CancelFunc
	sockssCxt    context.Context
	sockssCancel context.CancelFunc
	sync.RWMutex
}

func New(addr string, node *node.Node, signal chan os.Signal) *NodeApi {
	return &NodeApi{address: addr, node: node, osSignal: signal, srv: &http.Server{Addr: addr}}
}

func (na *NodeApi) Close() error {
	return na.srv.Shutdown(nil)
}
func (na *NodeApi) StartSrv() {
	mux := http.NewServeMux()
	mux.HandleFunc("/node/getTransports", wrap(na.getTransports))
	mux.HandleFunc("/node/reboot", wrap(na.runReboot))
	mux.HandleFunc("/node/run/sshs", wrap(na.runSshs))
	mux.HandleFunc("/node/run/sockss", wrap(na.runSockss))
	na.srv.Handler = cors.Default().Handler(mux)
	go func() {
		if err := na.srv.ListenAndServe(); err != nil {
			log.Printf("http server: ListenAndServe() error: %s", err)
		}
	}()
	log.Debugf("http server listen on %s", na.address)
}
func (na *NodeApi) getTransports(w http.ResponseWriter, r *http.Request) (result []byte, err error) {
	result, err = json.Marshal(na.node.GetTransport())
	if err != nil {
		return
	}
	return
}

func wrap(fn func(w http.ResponseWriter, r *http.Request) (result []byte, err error)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := fn(w, r)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(result)
	}
}

func (na *NodeApi) runReboot(w http.ResponseWriter, r *http.Request) (result []byte, err error) {
	cmd := exec.Command("reboot")
	err = cmd.Start()
	if err != nil {
		return
	}
	err = cmd.Wait()
	if err != nil {
		return
	}
	result = []byte("true")
	return
}

func (na *NodeApi) runSshs(w http.ResponseWriter, r *http.Request) (result []byte, err error) {
	na.Lock()
	if na.sshsCancel != nil {
		na.sshsCancel()
	}
	na.sshsCxt, na.sshsCancel = context.WithCancel(context.Background())

	cmd := exec.CommandContext(na.sshsCxt, "sshs", "-node-address", na.node.GetListenAddress())
	err = cmd.Start()
	if err != nil {
		return
	}

	na.Unlock()
	result = []byte("true")
	return
}

func (na *NodeApi) runSockss(w http.ResponseWriter, r *http.Request) (result []byte, err error) {
	na.Lock()
	if na.sockssCancel != nil {
		na.sockssCancel()
	}
	na.sockssCxt, na.sockssCancel = context.WithCancel(context.Background())

	cmd := exec.CommandContext(na.sshsCxt, "sockss", "-node-address", na.node.GetListenAddress())
	err = cmd.Start()
	if err != nil {
		return
	}

	na.Unlock()
	result = []byte("true")
	return
}
