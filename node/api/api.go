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
	"strings"
	"sync"
	"time"
	"fmt"
	"io/ioutil"
	"net/url"
	"regexp"
)

type NodeApi struct {
	address  string
	node     *node.Node
	config   Config
	osSignal chan os.Signal
	srv      *http.Server

	sshsCxt      context.Context
	sshsCancel   context.CancelFunc
	sockssCxt    context.Context
	sockssCancel context.CancelFunc
	sshcCxt      context.Context
	sshcCancel   context.CancelFunc
	sockscCxt    context.Context
	sockscCancel context.CancelFunc
	sync.RWMutex
}

func New(addr string, node *node.Node, config Config, signal chan os.Signal) *NodeApi {
	return &NodeApi{address: addr, node: node, config: config, osSignal: signal, srv: &http.Server{Addr: addr}}
}

func (na *NodeApi) Close() error {
	na.RLock()
	defer na.RUnlock()

	if na.sshsCancel != nil {
		na.sshsCancel()
	}
	if na.sshcCancel != nil {
		na.sshcCancel()
	}
	if na.sockssCancel != nil {
		na.sockssCancel()
	}
	if na.sockscCancel != nil {
		na.sockscCancel()
	}
	return na.srv.Close()
}

var env = ""

func (na *NodeApi) StartSrv() {
	mux := http.NewServeMux()
	na.getConfig()
	mux.HandleFunc("/node/getInfo", wrap(na.getInfo))
	mux.HandleFunc("/node/getApps", wrap(na.getApps))
	mux.HandleFunc("/node/reboot", wrap(na.runReboot))
	mux.HandleFunc("/node/run/sshs", wrap(na.runSshs))
	mux.HandleFunc("/node/run/sshc", wrap(na.runSshc))
	mux.HandleFunc("/node/run/sockss", wrap(na.runSockss))
	mux.HandleFunc("/node/run/socksc", wrap(na.runSocksc))
	mux.HandleFunc("/node/run/update", wrap(na.update))
	mux.HandleFunc("/node/run/updateNode", wrap(na.updateNode))
	na.srv.Handler = cors.Default().Handler(mux)
	go func() {
		log.Debugf("http server listening on %s", na.address)
		if err := na.srv.ListenAndServe(); err != nil {
			log.Errorf("http server: ListenAndServe() error: %s", err)
		}
	}()
}

func (na *NodeApi) getInfo(w http.ResponseWriter, r *http.Request) (result []byte, err error) {
	result, err = json.Marshal(na.node.GetNodeInfo())
	if err != nil {
		return
	}
	return
}

func (na *NodeApi) getApps(w http.ResponseWriter, r *http.Request) (result []byte, err error) {
	result, err = json.Marshal(na.node.GetApps())
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

func (na *NodeApi) runSshc(w http.ResponseWriter, r *http.Request) (result []byte, err error) {
	na.Lock()
	toNode := r.FormValue("toNode")
	toApp := r.FormValue("toApp")
	if na.sshcCancel != nil {
		na.sshcCancel()
	}
	na.sshcCxt, na.sshcCancel = context.WithCancel(context.Background())
	cmd := exec.CommandContext(na.sshcCxt, "sshc", "-node-key", toNode, "-app-key", toApp, "-node-address", na.node.GetListenAddress())
	err = cmd.Start()
	if err != nil {
		return
	}

	na.Unlock()
	result = []byte("true")
	return
}

func (na *NodeApi) runSocksc(w http.ResponseWriter, r *http.Request) (result []byte, err error) {
	na.Lock()
	toNode := r.FormValue("toNode")
	toApp := r.FormValue("toApp")
	if na.sockscCancel != nil {
		na.sockscCancel()
	}
	na.sockscCxt, na.sockscCancel = context.WithCancel(context.Background())

	cmd := exec.CommandContext(na.sockscCxt, "socksc", "-node-key", toNode, "-app-key", toApp, "-node-address", na.node.GetListenAddress())
	err = cmd.Start()
	if err != nil {
		return
	}

	na.Unlock()
	result = []byte("true")
	return
}

func (na *NodeApi) runSshs(w http.ResponseWriter, r *http.Request) (result []byte, err error) {
	na.Lock()
	if na.sshsCancel != nil {
		na.sshsCancel()
	}
	na.sshsCxt, na.sshsCancel = context.WithCancel(context.Background())
	var arr []string
	data := r.FormValue("data")
	if data != "" {
		arr = strings.Split(data, ",")
	}
	args := make([]string, 0, len(arr)+2)
	args = append(args, "-node-address")
	args = append(args, na.node.GetListenAddress())
	for _, v := range arr {
		args = append(args, "-node-key")
		args = append(args, v)
	}
	cmd := exec.CommandContext(na.sshsCxt, "sshs", args...)
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

	cmd := exec.CommandContext(na.sockssCxt, "sockss", "-node-address", na.node.GetListenAddress())
	err = cmd.Start()
	if err != nil {
		return
	}

	na.Unlock()
	result = []byte("true")
	return
}

func (na *NodeApi) update(w http.ResponseWriter, r *http.Request) (result []byte, err error) {
	branch := r.FormValue("branch")
	cmd := exec.Command("update-skywire", branch)
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

type Config struct {
	DiscoveryAddresses node.Addresses
	ConnectManager     bool
	ManagerAddr        string
	ManagerWeb         string
	Address            string
	Seed               bool
	SeedPath           string
	WebPort            string
}

var URLMatch = `(25[0-5]|2[0-4]\d|[0-1]\d{2}|[1-9]?\d)\.(25[0-5]|2[0-4]\d|[0-1]\d{2}|[1-9]?\d)\.(25[0-5]|2[0-4]\d|[0-1]\d{2}|[1-9]?\d)\.(25[0-5]|2[0-4]\d|[0-1]\d{2}|[1-9]?\d):\d{1,5}`

func (na *NodeApi) updateNode(w http.ResponseWriter, r *http.Request) (result []byte, err error) {
	na.getConfig()
	result = []byte("true")
	return
}

func (na *NodeApi) getConfig() {
	if len(na.node.Pk) < 1 {
		return
	}
	var managerUrl = na.config.ManagerWeb
	matched, err := regexp.MatchString(URLMatch, managerUrl)
	if err != nil || !matched {
		managerUrl = fmt.Sprintf("127.0.0.1%s", managerUrl)
	}
	log.Errorf("url: %s", managerUrl)
	res, err := http.PostForm(fmt.Sprintf("http://%s/conn/getNodeConfig", managerUrl), url.Values{"key": {na.node.Pk}})
	if err != nil {
		return
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Errorf("read config err: %v", err)
		return
	}
	log.Errorf("body: %s", body)
	if body != nil && string(body) != "null" {
		var config *Config
		err = json.Unmarshal(body, &config)
		if err != nil {
			log.Errorf("Unmarshal json err: %v", err)
			return
		}
		if len(na.config.DiscoveryAddresses) == len(config.DiscoveryAddresses) {
			for i, v := range na.config.DiscoveryAddresses {
				if v != config.DiscoveryAddresses[i] {
					na.config.DiscoveryAddresses = config.DiscoveryAddresses
					na.restart()
					return
				}
			}
		}
	}

}

func (na *NodeApi) restart() (err error) {
	args := make([]string, 0, len(na.config.DiscoveryAddresses))
	for _, v := range na.config.DiscoveryAddresses {
		args = append(args, "-discovery-address")
		args = append(args, v)
	}
	args = append(args, "-manager-address", na.config.ManagerAddr)
	args = append(args, "-address", na.config.Address)
	args = append(args, "-seed-path", na.config.SeedPath)
	args = append(args, "-web-port", na.config.WebPort)
	log.Errorf("start closing...: %s", args)
	cxt, cf := context.WithCancel(context.Background())
	go na.srv.Shutdown(cxt)
	go func() {
		time.Sleep(3000 * time.Millisecond)
		log.Errorf("closed end...")
		cf()
		os.Exit(0)
	}()
	na.node.Close()
	time.Sleep(1000 * time.Millisecond)
	cmd := exec.Command("node", args...)
	log.Errorf("exec restart...")
	err = cmd.Start()
	if err != nil {
		log.Errorf("cmd start err: %v",err)
		return
	}
	err = cmd.Process.Release()
	if err != nil {
		log.Errorf("cmd release err: %v",err)
		return
	}
	log.Errorf("exec restart end...")
	return
}
