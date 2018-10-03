package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skywire/pkg/net/skycoin-messenger/factory"
	"github.com/skycoin/skywire/pkg/node"
)

type NodeApi struct {
	address  string
	node     *node.Node
	config   *node.Config
	confPath string
	osSignal chan os.Signal
	srv      *http.Server

	token string

	apps map[string]*appCxt
	sync.RWMutex

	shell
}

func (na *NodeApi) SetToken(newToken string) {
	na.token = newToken
}

type appCxt struct {
	cxt    context.Context
	cancel context.CancelFunc
	ok     chan struct{}
}

func (cxt *appCxt) shutdown() {
	cxt.cancel()
	<-cxt.ok
	time.Sleep(3 * time.Second)
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

func New(addr, token string, node *node.Node, config *node.Config, confPath string, signal chan os.Signal) *NodeApi {

	return &NodeApi{
		address:  addr,
		token:    token,
		node:     node,
		config:   config,
		confPath: confPath,
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
	//err := na.getConfig()
	//if err != nil {
	//	log.Errorf("get config error: %s", err)
	//}
	err := na.afterLaunch()
	if err != nil {
		log.Errorf("after launch error: %s", err)
	}
	http.HandleFunc("/node/getSig", na.wrap(na.getSig))
	http.HandleFunc("/node/getInfo", na.wrap(na.getInfo))
	http.HandleFunc("/node/getMsg", na.wrap(na.getMsg))
	http.HandleFunc("/node/getApps", na.wrap(na.getApps))
	http.HandleFunc("/node/reboot", na.wrap(na.runReboot))
	http.HandleFunc("/node/run/sshs", na.wrap(na.runSshs))
	http.HandleFunc("/node/run/sshc", na.wrap(na.runSshc))
	http.HandleFunc("/node/run/sockss", na.wrap(na.runSockss))
	http.HandleFunc("/node/run/socksc", na.wrap(na.runSocksc))
	http.HandleFunc("/node/run/update", na.wrap(na.update))
	http.HandleFunc("/node/run/checkUpdate", na.wrap(na.checkUpdate))
	http.HandleFunc("/node/run/setNodeConfig", na.wrap(na.setNodeConfig))
	http.HandleFunc("/node/run/updateNode", na.wrap(na.updateNode))
	http.HandleFunc("/node/run/runShell", na.wrap(na.runShell))
	http.HandleFunc("/node/run/runCmd", na.wrap(na.runCmd))
	http.HandleFunc("/node/run/getShellOutput", na.wrap(na.getShellOutput))
	http.HandleFunc("/node/run/searchServices", na.wrap(na.search))
	http.HandleFunc("/node/run/getSearchServicesResult", na.wrap(na.getSearchResult))
	http.HandleFunc("/node/run/getAutoStartConfig", na.wrap(na.getAutoStartConfig))
	http.HandleFunc("/node/run/setAutoStartConfig", na.wrap(na.setAutoStartConfig))
	http.HandleFunc("/node/run/closeApp", na.wrap(na.closeApp))
	http.HandleFunc("/node/run/term", na.handleXtermsocket)
	na.srv.Handler = http.DefaultServeMux
	go func() {
		log.Debugf("http server listening on %s", na.address)
		if err := na.srv.ListenAndServe(); err != nil {
			log.Errorf("http server: ListenAndServe() error: %s", err)
		}
	}()
}

type Sig struct {
	Sig string `json:"sig"`
}

func (na *NodeApi) getSig(w http.ResponseWriter, r *http.Request) (result []byte, err error) {
	data := r.FormValue("data")
	if len(data) == 0 {
		err = errors.New("Hash is Empty!")
		return
	}
	hash := cipher.SumSHA256([]byte(data))
	sc, err := factory.ReadSeedConfig(na.config.SeedPath)
	if err != nil {
		return
	}
	secKey := cipher.MustSecKeyFromHex(sc.SecKey)
	sig := &Sig{
		Sig: cipher.SignHash(hash, secKey).Hex(),
	}
	result, err = json.Marshal(sig)
	return
}

func (na *NodeApi) closeApp(w http.ResponseWriter, r *http.Request) (result []byte, err error) {
	key := r.FormValue("key")
	if len(key) == 0 {
		err = errors.New("Key is Empty!")
		return
	}
	na.Lock()
	defer na.Unlock()
	v, ok := na.apps[key]
	if ok {
		if v != nil {
			v.cancel()
			delete(na.apps, key)
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

func (na *NodeApi) wrap(fn func(w http.ResponseWriter, r *http.Request) (result []byte, err error)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.FormValue("token")
		if token != na.token {
			w.Write([]byte("manager token is null"))
			return
		}
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
	discoveryKey := r.FormValue("discoveryKey")
	err = na.startSshc(toNode, toApp, discoveryKey)
	if err != nil {
		return
	}
	result = []byte("true")
	return
}

func (na *NodeApi) startSshc(toNode, toApp, disvoerveryKey string) (err error) {
	var gopath = os.Getenv("GOPATH")
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
		app.shutdown()
	}
	cxt, cancel := context.WithCancel(context.Background())
	isOk := make(chan struct{})
	app = &appCxt{
		cxt:    cxt,
		cancel: cancel,
		ok:     isOk,
	}
	na.apps[key] = app
	cmd := exec.CommandContext(app.cxt, filepath.Join(gopath, "bin", "sshc"), "-node-key",
		toNode, "-app-key", toApp, "-discovery-key", disvoerveryKey, "-node-address", na.node.GetListenAddress())
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
	discoveryKey := r.FormValue("discoveryKey")
	err = na.startSocksc(toNode, toApp, discoveryKey)
	if err != nil {
		return
	}
	result = []byte("true")
	return
}

func (na *NodeApi) startSocksc(toNode, toApp, disvoerveryKey string) (err error) {
	var gopath = os.Getenv("GOPATH")
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
		app.shutdown()
	}
	cxt, cancel := context.WithCancel(context.Background())
	isOk := make(chan struct{})
	app = &appCxt{
		cxt:    cxt,
		cancel: cancel,
		ok:     isOk,
	}
	na.apps[key] = app
	cmd := exec.CommandContext(app.cxt, filepath.Join(gopath, "bin", "socksc"), "-node-key",
		toNode, "-app-key", toApp, "-discovery-key", disvoerveryKey, "-node-address", na.node.GetListenAddress())
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
	var gopath = os.Getenv("GOPATH")
	na.Lock()
	defer na.Unlock()
	key := "sshs"
	app := na.apps[key]
	if app != nil {
		app.shutdown()
	}
	cxt, cancel := context.WithCancel(context.Background())
	isOk := make(chan struct{})
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
	cmd := exec.CommandContext(app.cxt, filepath.Join(gopath, "bin", "sshs"), args...)
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
	var gopath = os.Getenv("GOPATH")
	na.Lock()
	defer na.Unlock()
	key := "sockss"
	app := na.apps[key]
	if app != nil {
		app.shutdown()
	}
	cxt, cancel := context.WithCancel(context.Background())
	isOk := make(chan struct{})
	app = &appCxt{
		cxt:    cxt,
		cancel: cancel,
		ok:     isOk,
	}
	na.apps[key] = app
	cmd := exec.CommandContext(app.cxt, filepath.Join(gopath, "bin", "sockss"), "-node-address", na.node.GetListenAddress())
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

var scriptPath = "/src/github.com/skycoin/skywire/static/script/"

func (na *NodeApi) checkUpdate(w http.ResponseWriter, r *http.Request) (result []byte, err error) {
	var cmd *exec.Cmd
	var gopath = os.Getenv("GOPATH")
	osName := runtime.GOOS
	if osName == "windows" {
		cmd = exec.Command("cmd", "/C", filepath.Join(gopath, fmt.Sprintf("%s%s", scriptPath, "win/check.bat")))
	} else {
		cmd = exec.Command(filepath.Join(gopath, fmt.Sprintf("%s%s", scriptPath, "unix/check")))
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return
	}
	result = out
	return
}

func (na *NodeApi) update(w http.ResponseWriter, r *http.Request) (result []byte, err error) {
	var cmd *exec.Cmd
	var gopath = os.Getenv("GOPATH")
	osName := runtime.GOOS
	if osName == "windows" {
		cmd = exec.Command("cmd", "/C", filepath.Join(gopath, fmt.Sprintf("%s%s", scriptPath, "win/update-skywire.bat")))
	} else {
		cmd = exec.Command(filepath.Join(gopath, fmt.Sprintf("%s%s", scriptPath, "unix/update-skywire")))
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return
	}
	result = out
	return
}

func (na *NodeApi) setNodeConfig(w http.ResponseWriter, r *http.Request) (result []byte, err error) {
	key := r.FormValue("key")
	if len(key) != 66 {
		err = errors.New("invalid key")
		return
	}
	data := r.FormValue("data")
	conf := &node.Config{}
	err = json.Unmarshal([]byte(data), conf)
	if err != nil {
		return
	}
	cfs := &node.NodeConfigs{}
	err = node.LoadConfig(cfs, na.confPath)
	if err != nil {
		return
	}
	cfs.Configs[key] = conf
	err = node.WriteConfig(cfs, na.confPath)
	if err != nil {
		return
	}
	result = []byte("true")
	return
}

var URLMatch = `(25[0-5]|2[0-4]\d|[0-1]\d{2}|[1-9]?\d)\.(25[0-5]|2[0-4]\d|[0-1]\d{2}|[1-9]?\d)\.(25[0-5]|2[0-4]\d|[0-1]\d{2}|[1-9]?\d)\.(25[0-5]|2[0-4]\d|[0-1]\d{2}|[1-9]?\d):\d{1,5}`

func (na *NodeApi) updateNode(w http.ResponseWriter, r *http.Request) (result []byte, err error) {
	na.restart()
	result = []byte("true")
	return
}

func (na *NodeApi) restart() (err error) {
	var gopath = os.Getenv("GOPATH")
	conf, err := node.GetNodeDefaultConfig(na.confPath)
	if err != nil {
		return
	}
	args := make([]string, 0, len(conf.DiscoveryAddresses))
	for _, v := range na.config.DiscoveryAddresses {
		args = append(args, "-discovery-address")
		args = append(args, v)
	}
	args = append(args, "-manager-address", na.config.ManagerAddr)
	args = append(args, "-address", na.config.Address)
	args = append(args, "-seed-path", na.config.SeedPath)
	args = append(args, "-web-port", na.config.WebPort)
	args = append(args, "-conf", na.confPath)
	na.Close()
	na.srv.Close()
	na.node.Close()
	time.Sleep(1000 * time.Millisecond)
	cmd := exec.Command(filepath.Join(gopath, "bin", "node"), args...)
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
	go func() {
		time.Sleep(3000 * time.Millisecond)
		os.Exit(0)
	}()
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
	p := r.FormValue("pages")
	pages, err := strconv.Atoi(p)
	if err != nil {
		return
	}
	l := r.FormValue("limit")
	limit, err := strconv.Atoi(l)
	if err != nil {
		return
	}
	discoveryKeyHex := r.FormValue("discoveryKey")
	if len(discoveryKeyHex) == 0 {
		return
	}
	discovery, err := cipher.PubKeyFromHex(discoveryKeyHex)
	if err != nil {
		return
	}
	seqs := na.node.Search(pages, limit, discovery, key)
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
	token := r.Header.Get("manager-token")
	if token != na.token {
		return
	}
	xterm(w, r)
}

type windowSize struct {
	Rows uint16 `json:"rows"`
	Cols uint16 `json:"cols"`
}

var autoVersion = 2

func (na *NodeApi) afterLaunch() (err error) {
	key, err := na.node.GetNodeKey()
	if err != nil {
		return
	}
	if len(key) < 66 {
		err = errors.New("Key at least 66 characters.")
		return
	}
	f, err := na.node.ReadAutoStartConfig()
	if err != nil {
		if os.IsNotExist(err) {
			f = na.node.NewAutoStartFile()

		} else {
			return
		}
	}
	if f.Version != autoVersion {
		switch f.Version {
		case 0:
			f, err = na.adaptOldConfig(key)
			if err != nil {
				return
			}
		case 1:
			lc, err := na.node.ReadOld1AutoStartConfig()
			if err != nil {
				return err
			}
			f, err = na.adaptOld1Config(lc, key)
			if err != nil {
				return err
			}
		}
	}
	conf, ok := f.Config[key]
	if !ok {
		conf = na.node.NewAutoStartConfig()
		f.Config[key] = conf
		err = na.node.WriteAutoStartConfig(f, na.config.AutoStartPath)
		if err != nil {
			return
		}
	}
	if conf.Sockss {
		log.Infof("start sockss...")
		err = na.startSockss()
		if err != nil {
			return
		}
	}
	if conf.Sshs {
		log.Infof("start sshs...")
		err = na.startSshs(nil)
		if err != nil {
			return
		}
	}

	if conf.Socksc {
		err = na.startSocksc(conf.SockscConfNodeKey, conf.SockscConfAppKey, conf.SockscConfDiscovery)
		if err != nil {
			return
		}
	}
	if conf.Sshc {
		err = na.startSshc(conf.SshcConfNodeKey, conf.SshcConfAppKey, conf.SshcConfDiscovery)
		if err != nil {
			return
		}
	}
	return
}

func (na *NodeApi) adaptOldConfig(key string) (f node.AutoStartFile, err error) {
	olc, err := na.node.ReadOldAutoStartConfig()
	if os.IsNotExist(err) {
		f = na.node.NewAutoStartFile()
		asc := na.node.NewAutoStartConfig()
		f.Config[key] = asc
		err = na.node.WriteAutoStartConfig(f, na.config.AutoStartPath)
		if err != nil {
			return
		}
	}
	asc := na.node.NewAutoStartConfig()
	asc.Sockss = olc.SocksServer
	asc.Sshs = olc.SshServer
	f.Config = make(map[string]node.AutoStartConfig)
	f.Config[key] = asc
	f.Version = autoVersion
	err = na.node.WriteAutoStartConfig(f, na.config.AutoStartPath)
	if err != nil {
		return
	}
	return
}

func (na *NodeApi) adaptOld1Config(lc node.Old1AutoStartConfig, key string) (f node.AutoStartFile, err error) {
	f = na.node.NewAutoStartFile()
	asc := node.AutoStartConfig{
		Sshs:              lc.Sshs,
		Sshc:              lc.Sshc,
		SshcConfNodeKey:   lc.SshcConfNodeKey,
		SshcConfAppKey:    lc.SshcConfAppKey,
		Sockss:            lc.Sockss,
		Socksc:            lc.Socksc,
		SockscConfNodeKey: lc.SockscConfNodeKey,
		SockscConfAppKey:  lc.SockscConfAppKey,
	}
	f.Config[key] = asc
	err = na.node.WriteAutoStartConfig(f, na.config.AutoStartPath)
	if err != nil {
		return
	}
	return
}

func (na *NodeApi) getAutoStartConfig(w http.ResponseWriter, r *http.Request) (result []byte, err error) {
	key := r.FormValue("key")
	if len(key) < 66 {
		err = errors.New("Key at least 66 characters.")
		return
	}
	f, err := na.node.ReadAutoStartConfig()
	if err != nil {
		if os.IsNotExist(err) {
			f = na.node.NewAutoStartFile()
			asc := na.node.NewAutoStartConfig()
			f.Config[key] = asc
			err = na.node.WriteAutoStartConfig(f, na.config.AutoStartPath)
			if err != nil {
				return
			}
		} else {
			log.Errorf("read launch config err:", err)
			return
		}
	}
	result, err = json.Marshal(f.Config[key])
	return
}

func (na *NodeApi) setAutoStartConfig(w http.ResponseWriter, r *http.Request) (result []byte, err error) {
	data := r.FormValue("data")
	key := r.FormValue("key")
	if len(key) < 66 {
		err = errors.New("Key at least 66 characters.")
		return
	}
	var asc = node.AutoStartConfig{}
	err = json.Unmarshal([]byte(data), &asc)
	if err != nil {
		return
	}
	f, err := na.node.ReadAutoStartConfig()
	if err != nil {
		if os.IsNotExist(err) {
			f = na.node.NewAutoStartFile()
			asc := na.node.NewAutoStartConfig()
			f.Config[key] = asc
			err = na.node.WriteAutoStartConfig(f, na.config.AutoStartPath)
			if err != nil {
				return
			}
		} else {
			log.Errorf("read launch config err:", err)
			return
		}
	}
	f.Config[key] = asc
	err = na.node.WriteAutoStartConfig(f, na.config.AutoStartPath)
	if err != nil {
		return
	}
	result = []byte("true")
	return
}
