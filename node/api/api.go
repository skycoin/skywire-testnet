package api

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skywire/node"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
	"github.com/gorilla/websocket"
	"runtime"
)

type NodeApi struct {
	address  string
	node     *node.Node
	config   Config
	osSignal chan os.Signal
	srv      *http.Server

	apps map[string]*appCxt
	sync.RWMutex

	shell
}

type appCxt struct {
	cxt    context.Context
	cancel context.CancelFunc
	ok     chan interface{}
}

type shell struct {
	cmd       *exec.Cmd
	cmdCxt    context.Context
	cmdCancel context.CancelFunc
	input     io.WriteCloser
	output    io.ReadCloser
	outChan   chan []byte
	timer     *time.Timer
	closed    bool
	sync.Mutex
}

func New(addr string, node *node.Node, config Config, signal chan os.Signal) *NodeApi {

	return &NodeApi{
		address:  addr,
		node:     node,
		config:   config,
		osSignal: signal,
		srv:      &http.Server{Addr: addr},
		apps:     make(map[string]*appCxt),
	}
}

func (na *NodeApi) Close() error {
	na.RLock()
	defer na.RUnlock()

	for _, v := range na.apps {
		if v != nil {
			v.cancel()
		}
	}
	return na.srv.Close()
}

func (na *NodeApi) StartSrv() {
	na.getConfig()
	err := na.afterLaunch()
	if err != nil {
		log.Errorf("after launch error: %s", err)
	}
	http.HandleFunc("/node/getInfo", wrap(na.getInfo))
	http.HandleFunc("/node/getMsg", wrap(na.getMsg))
	http.HandleFunc("/node/getApps", wrap(na.getApps))
	http.HandleFunc("/node/reboot", wrap(na.runReboot))
	http.HandleFunc("/node/run/sshs", wrap(na.runSshs))
	http.HandleFunc("/node/run/sshc", wrap(na.runSshc))
	http.HandleFunc("/node/run/sockss", wrap(na.runSockss))
	http.HandleFunc("/node/run/socksc", wrap(na.runSocksc))
	http.HandleFunc("/node/run/update", wrap(na.update))
	http.HandleFunc("/node/run/updateNode", wrap(na.updateNode))
	http.HandleFunc("/node/run/runShell", wrap(na.runShell))
	http.HandleFunc("/node/run/runCmd", wrap(na.runCmd))
	http.HandleFunc("/node/run/getShellOutput", wrap(na.getShellOutput))
	http.HandleFunc("/node/run/searchServices", wrap(na.search))
	http.HandleFunc("/node/run/getSearchServicesResult", wrap(na.getSearchResult))
	http.HandleFunc("/node/run/getAutoStartConfig", wrap(na.getAutoStartConfig))
	http.HandleFunc("/node/run/setAutoStartConfig", wrap(na.setAutoStartConfig))
	http.HandleFunc("/node/run/closeApp", wrap(na.closeApp))
	http.HandleFunc("/node/run/term", na.handleXtermsocket)
	na.srv.Handler = http.DefaultServeMux
	go func() {
		log.Debugf("http server listening on %s", na.address)
		if err := na.srv.ListenAndServe(); err != nil {
			log.Errorf("http server: ListenAndServe() error: %s", err)
		}
	}()
}

func (na *NodeApi) closeApp(w http.ResponseWriter, r *http.Request) (result []byte, err error) {
	key := r.FormValue("key")
	if len(key) == 0 {
		err = errors.New("Key is Empty!")
		return
	}
	na.RLock()
	defer na.RUnlock()
	v, ok := na.apps[key]
	if ok {
		if v != nil {
			v.cancel()
		}
	}
	result = []byte("true")
	return
}

func (na *NodeApi) getInfo(w http.ResponseWriter, r *http.Request) (result []byte, err error) {
	result, err = json.Marshal(na.node.GetNodeInfo())
	if err != nil {
		return
	}
	return
}

func (na *NodeApi) getMsg(w http.ResponseWriter, r *http.Request) (result []byte, err error) {
	k, err := cipher.PubKeyFromHex(r.FormValue("key"))
	if err != nil {
		return
	}
	result, err = json.Marshal(na.node.GetMessages(k))
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
		if len(w.Header().Get("Content-Type")) <= 0 {
			w.Header().Set("Content-Type", "application/json")
		}
		w.Write(result)
	}
}

func (na *NodeApi) runReboot(w http.ResponseWriter, r *http.Request) (result []byte, err error) {
	var cmd *exec.Cmd
	osName := runtime.GOOS
	switch osName {
	case "linux":
		cmd = exec.Command("reboot")
	case "windows":
		cmd = exec.Command("cmd", "/C", "shutdown", "-r", "-t", "0")
	case "darwin":
		cmd = nil
	}
	if cmd == nil {
		result = []byte("darwin system os unsupported")
		return
	}
	err = cmd.Start()
	if err != nil {
		return
	}
	go func() {
		cmd.Wait()
	}()
	result = []byte("true")
	return
}

func (na *NodeApi) runSshc(w http.ResponseWriter, r *http.Request) (result []byte, err error) {
	na.Lock()
	defer na.Unlock()
	toNode := r.FormValue("toNode")
	toApp := r.FormValue("toApp")
	err = na.startSshc(toNode, toApp)
	if err != nil {
		return
	}
	result = []byte("true")
	return
}

func (na *NodeApi) startSshc(toNode string, toApp string) (err error) {
	if len(toNode) == 0 || len(toNode) < 66 {
		err = errors.New("Node Key at least 66 characters.")
		return
	}
	if len(toApp) == 0 || len(toApp) < 66 {
		err = errors.New("App Key at least 66 characters.")
		return
	}
	key := "sshc"
	app := na.apps[key]
	if app != nil {
		app.cancel()
		<-app.ok
	}
	cxt, cancel := context.WithCancel(context.Background())
	isOk := make(chan interface{})
	app = &appCxt{
		cxt:    cxt,
		cancel: cancel,
		ok:     isOk,
	}
	na.apps[key] = app
	cmd := exec.CommandContext(app.cxt, "./sshc", "-node-key", toNode, "-app-key", toApp, "-node-address", na.node.GetListenAddress())
	err = cmd.Start()
	if err != nil {
		return
	}
	go func() {
		cmd.Wait()
		close(isOk)
	}()
	return
}

func (na *NodeApi) runSocksc(w http.ResponseWriter, r *http.Request) (result []byte, err error) {
	na.Lock()
	defer na.Unlock()
	toNode := r.FormValue("toNode")
	toApp := r.FormValue("toApp")
	err = na.startSocksc(toNode, toApp)
	if err != nil {
		return
	}
	result = []byte("true")
	return
}

func (na *NodeApi) startSocksc(toNode string, toApp string) (err error) {
	if len(toNode) == 0 || len(toNode) < 66 {
		err = errors.New("Node Key at least 66 characters.")
		return
	}
	if len(toApp) == 0 || len(toApp) < 66 {
		err = errors.New("App Key at least 66 characters.")
		return
	}
	key := "socksc"
	app := na.apps[key]
	if app != nil {
		app.cancel()
		<-app.ok
	}
	cxt, cancel := context.WithCancel(context.Background())
	isOk := make(chan interface{})
	app = &appCxt{
		cxt:    cxt,
		cancel: cancel,
		ok:     isOk,
	}
	na.apps[key] = app
	cmd := exec.CommandContext(app.cxt, "./socksc", "-node-key", toNode, "-app-key", toApp, "-node-address", na.node.GetListenAddress())
	err = cmd.Start()
	if err != nil {
		return
	}
	go func() {
		cmd.Wait()
		close(isOk)
	}()
	return
}

func (na *NodeApi) runSshs(w http.ResponseWriter, r *http.Request) (result []byte, err error) {
	var arr []string
	data := r.FormValue("data")
	if len(data) > 1 {
		arr = strings.Split(data, ",")
	}
	err = na.startSshs(arr)
	if err != nil {
		return
	}
	result = []byte("true")
	return
}

func (na *NodeApi) startSshs(arr []string) (err error) {
	na.Lock()
	defer na.Unlock()
	key := "sshs"
	app := na.apps[key]
	if app != nil {
		app.cancel()
		<-app.ok
	}
	cxt, cancel := context.WithCancel(context.Background())
	isOk := make(chan interface{})
	app = &appCxt{
		cxt:    cxt,
		cancel: cancel,
		ok:     isOk,
	}
	na.apps[key] = app
	args := make([]string, 0, len(arr)+2)
	args = append(args, "-node-address")
	args = append(args, na.node.GetListenAddress())
	for _, v := range arr {
		args = append(args, "-node-key")
		args = append(args, v)
	}
	cmd := exec.CommandContext(app.cxt, "./sshs", args...)
	err = cmd.Start()
	if err != nil {
		return
	}
	go func() {
		cmd.Wait()
		close(isOk)
	}()
	return
}

func (na *NodeApi) runSockss(w http.ResponseWriter, r *http.Request) (result []byte, err error) {
	err = na.startSockss()
	if err != nil {
		return
	}
	result = []byte("true")
	return
}

func (na *NodeApi) startSockss() (err error) {
	na.Lock()
	defer na.Unlock()
	key := "sockss"
	app := na.apps[key]
	if app != nil {
		app.cancel()
		<-app.ok
	}
	cxt, cancel := context.WithCancel(context.Background())
	isOk := make(chan interface{})
	app = &appCxt{
		cxt:    cxt,
		cancel: cancel,
		ok:     isOk,
	}
	na.apps[key] = app
	cmd := exec.CommandContext(app.cxt, "./sockss", "-node-address", na.node.GetListenAddress())
	err = cmd.Start()
	if err != nil {
		return
	}
	go func() {
		cmd.Wait()
		close(isOk)
	}()

	return
}

func (na *NodeApi) update(w http.ResponseWriter, r *http.Request) (result []byte, err error) {
	cmd := exec.Command("./update-skywire")
	err = cmd.Start()
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
	AutoStartPath      string
	WebPort            string
}

var URLMatch = `(25[0-5]|2[0-4]\d|[0-1]\d{2}|[1-9]?\d)\.(25[0-5]|2[0-4]\d|[0-1]\d{2}|[1-9]?\d)\.(25[0-5]|2[0-4]\d|[0-1]\d{2}|[1-9]?\d)\.(25[0-5]|2[0-4]\d|[0-1]\d{2}|[1-9]?\d):\d{1,5}`

func (na *NodeApi) updateNode(w http.ResponseWriter, r *http.Request) (result []byte, err error) {
	na.getConfig()
	result = []byte("true")
	return
}

func (na *NodeApi) getConfig() {
	var managerUrl = na.config.ManagerWeb
	matched, err := regexp.MatchString(URLMatch, managerUrl)
	if err != nil || !matched {
		managerUrl = fmt.Sprintf("127.0.0.1%s", managerUrl)
	}
	res, err := http.PostForm(fmt.Sprintf("http://%s/conn/getNodeConfig", managerUrl),
		url.Values{
			"key": {na.node.GetManager().GetDefaultSeedConfig().PublicKey},
		})
	if err != nil {
		return
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Errorf("read config err: %v", err)
		return
	}
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
		log.Errorf("cmd start err: %v", err)
		return
	}
	err = cmd.Process.Release()
	if err != nil {
		log.Errorf("cmd release err: %v", err)
		return
	}
	log.Errorf("exec restart end...")
	return
}

func (na *NodeApi) getShellOutput(w http.ResponseWriter, r *http.Request) (result []byte, err error) {
	na.shell.Lock()
	defer na.shell.Unlock()

	if na.output == nil {
		result = []byte("false")
		return
	}
	select {
	case r, ok := <-na.shell.outChan:
		if !ok {
			result = []byte("false")
			return
		}
		result = r
	default:
		result = []byte{}
	}
	na.shell.resetTimer(true)
	return
}

func (na *NodeApi) runCmd(w http.ResponseWriter, r *http.Request) (result []byte, err error) {
	na.shell.Lock()
	defer na.shell.Unlock()

	if na.input == nil {
		result = []byte("false")
		return
	}

	cmd := r.FormValue("command")
	_, err = io.WriteString(na.input, fmt.Sprintf("%s\n", cmd))
	if err != nil {
		return
	}
	result = []byte("true")

	na.shell.resetTimer(true)
	return
}

func (s *shell) resetTimer(start bool) {
	if s.timer != nil {
		s.timer.Stop()
	}
	if !start {
		return
	}
	s.timer = time.AfterFunc(2*time.Minute, func() {
		s.Lock()
		defer s.Unlock()
		s.close()
	})
}

func (s *shell) close() {
	if s.closed {
		return
	}
	s.closed = true
	s.resetTimer(false)
	s.cmdCancel()
	close(s.outChan)
	go s.cmd.Wait()
	s.cmdCxt = nil
	s.cmdCancel = nil
	s.cmd = nil
	s.input = nil
	s.output = nil
	s.outChan = nil
	s.timer = nil
}

func (na *NodeApi) runShell(w http.ResponseWriter, r *http.Request) (result []byte, err error) {
	na.shell.Lock()
	defer na.shell.Unlock()

	if na.cmdCancel != nil {
		na.shell.close()
	}
	na.shell.closed = false
	na.cmdCxt, na.cmdCancel = context.WithCancel(context.Background())
	na.shell.cmd = exec.CommandContext(na.cmdCxt, "/bin/bash")
	na.shell.outChan = make(chan []byte, 1)
	na.shell.input, err = na.shell.cmd.StdinPipe()
	if err != nil {
		return
	}
	na.shell.output, err = na.shell.cmd.StdoutPipe()
	if err != nil {
		return
	}
	go func(output io.ReadCloser) {
		defer func() {
			if e := recover(); e != nil {
				return
			}
		}()
		for {
			result := make([]byte, 1024*4)
			n, err := output.Read(result)
			if err != nil {
				return
			}
			na.shell.outChan <- result[:n]
		}
	}(na.shell.output)

	na.shell.resetTimer(true)

	err = na.shell.cmd.Start()
	if err != nil {
		return
	}
	result = []byte("true")
	return
}

func (na *NodeApi) search(w http.ResponseWriter, r *http.Request) (result []byte, err error) {
	key := r.FormValue("key")
	if len(key) == 0 {
		err = errors.New("invalid key")
		return
	}

	seqs := na.node.Search(key)

	result, err = json.Marshal(seqs)
	return
}

func (na *NodeApi) getSearchResult(w http.ResponseWriter, r *http.Request) (result []byte, err error) {
	srs := na.node.GetSearchResult()
	if err != nil {
		return
	}

	result, err = json.Marshal(srs)
	return
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (na *NodeApi) handleXtermsocket(w http.ResponseWriter, r *http.Request) {
	xterm(w, r)
}

type windowSize struct {
	Rows uint16 `json:"rows"`
	Cols uint16 `json:"cols"`
}

func (na *NodeApi) afterLaunch() (err error) {
	lc, err := na.node.ReadAutoStartConfig()
	if err != nil {
		if os.IsNotExist(err) {
			lc = na.node.NewAutoStartConfig()
			err = na.node.WriteAutoStartConfig(lc, na.config.AutoStartPath)
			if err != nil {
				return
			}
		} else {
			err = errors.New("read launch config")
			return
		}
	}
	if lc["sockss"] == "true" {
		err = na.startSockss()
		if err != nil {
			return
		}
	}
	if lc["sshs"] == "true" {
		err = na.startSshs(nil)
		if err != nil {
			return
		}
	}

	if lc["socksc"] == "true" {
		err = na.startSocksc(lc["socksc_conf_nodeKey"], lc["socksc_conf_appKey"])
		if err != nil {
			return
		}
	}
	if lc["sshc"] == "true" {
		err = na.startSshc(lc["sshc_conf_nodeKey"], lc["sshc_conf_appKey"])
		if err != nil {
			return
		}
	}
	return
}

func (na *NodeApi) getAutoStartConfig(w http.ResponseWriter, r *http.Request) (result []byte, err error) {
	lc, err := na.node.ReadAutoStartConfig()
	if err != nil {
		if os.IsNotExist(err) {
			lc = na.node.NewAutoStartConfig()
			err = na.node.WriteAutoStartConfig(lc, na.config.AutoStartPath)
			if err != nil {
				return
			}
		} else {
			log.Errorf("read launch config err:", err)
			return
		}
	}
	result, err = json.Marshal(lc)
	return
}

func (na *NodeApi) setAutoStartConfig(w http.ResponseWriter, r *http.Request) (result []byte, err error) {
	data := r.FormValue("data")
	var lc = make(map[string]string)
	err = json.Unmarshal([]byte(data), &lc)
	if err != nil {
		return
	}
	err = na.node.WriteAutoStartConfig(lc, na.config.AutoStartPath)
	if err != nil {
		return
	}
	result = []byte("true")
	return
}
