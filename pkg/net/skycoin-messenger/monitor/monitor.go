package monitor

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/astaxie/beego/session"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/util/file"
	"github.com/skycoin/skywire/pkg/net/skycoin-messenger/factory"
)

var globalSessions *session.Manager

func init() {
	sessionConfig := &session.ManagerConfig{
		CookieName:      "SWSId",
		EnableSetCookie: true,
		Gclifetime:      3600,
		Maxlifetime:     86400,
		Secure:          false,
		CookieLifeTime:  3600,
		ProviderConfig:  "./tmp",
	}
	globalSessions, _ = session.NewManager("memory", sessionConfig)
	go globalSessions.GC()
}

type Conn struct {
	Key         string `json:"key"`
	Type        string `json:"type"`
	SendBytes   uint64 `json:"send_bytes"`
	RecvBytes   uint64 `json:"recv_bytes"`
	LastAckTime int64  `json:"last_ack_time"`
	StartTime   int64  `json:"start_time"`
}
type NodeServices struct {
	Type        string `json:"type"`
	Addr        string `json:"addr"`
	SendBytes   uint64 `json:"send_bytes"`
	RecvBytes   uint64 `json:"recv_bytes"`
	LastAckTime int64  `json:"last_ack_time"`
	StartTime   int64  `json:"start_time"`
}
type App struct {
	Index      int      `json:"index"`
	Key        string   `json:"key"`
	Attributes []string `json:"attributes"`
}

var (
	NULL = "null"
)
var (
	BAD_REQUEST  = 400
	NOT_FOUND    = 404
	SERVER_ERROR = 500
)

type Monitor struct {
	factory       *factory.MessengerFactory
	serverAddress string
	address       string
	srv           *http.Server

	tag     string
	version string

	token string

	configs      map[string]*Config
	configsMutex sync.RWMutex
}

func New(f *factory.MessengerFactory, serverAddress, webAddr, tag, version string) *Monitor {
	return &Monitor{
		factory:       f,
		serverAddress: serverAddress,
		address:       webAddr,
		srv:           &http.Server{Addr: webAddr},
		tag:           tag,
		version:       version,
		token:         getRandomString(32),
		configs:       make(map[string]*Config),
	}
}

func (m *Monitor) Close() error {
	return m.srv.Close()
}
func (m *Monitor) Start(webDir string) {
	http.Handle("/", http.FileServer(http.Dir(webDir)))
	http.HandleFunc("/conn/getAll", bundle(m.getAllNode))
	http.HandleFunc("/conn/getServerInfo", bundle(m.getServerInfo))
	http.HandleFunc("/conn/getNode", bundle(m.getNode))
	http.HandleFunc("/conn/setNodeConfig", bundle(m.setNodeConfig))
	http.HandleFunc("/conn/getNodeConfig", bundle(m.getNodeConfig))
	http.HandleFunc("/conn/saveClientConnection", bundle(m.SaveClientConnection))
	http.HandleFunc("/conn/removeClientConnection", bundle(m.RemoveClientConnection))
	http.HandleFunc("/conn/editClientConnection", bundle(m.EditClientConnection))
	http.HandleFunc("/conn/getClientConnection", bundle(m.GetClientConnection))
	http.HandleFunc("/login", bundle(m.Login))
	http.HandleFunc("/checkLogin", bundle(m.checkLogin))
	http.HandleFunc("/updatePass", bundle(m.UpdatePass))
	http.HandleFunc("/req", bundle(m.req))
	http.HandleFunc("/term", m.handleNodeTerm)
	http.HandleFunc("/getPort", bundle(m.getPort))
	http.HandleFunc("/getToken", bundle(m.getToken))
	go func() {
		if err := m.srv.ListenAndServe(); err != nil {
			log.Printf("http server: ListenAndServe() error: %s", err)
		}
	}()
	log.Debugf("http server listen on %s", m.address)
}

func bundle(fn func(w http.ResponseWriter, r *http.Request) (result []byte, err error, code int)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err, code := fn(w, r)
		if err != nil {
			if code == 0 {
				code = SERVER_ERROR
			}
			http.Error(w, err.Error(), code)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(result)
	}
}

func (m *Monitor) getPort(w http.ResponseWriter, r *http.Request) (result []byte, err error, code int) {
	if !verifyLogin(w, r, false) {
		return
	}
	_, port, err := net.SplitHostPort(m.address)
	if err != nil {
		return
	}
	result = []byte(port)
	return
}

func (m *Monitor) getToken(w http.ResponseWriter, r *http.Request) (result []byte, err error, code int) {
	result = []byte(m.token)
	return
}

func (m *Monitor) req(w http.ResponseWriter, r *http.Request) (result []byte, err error, code int) {
	if !verifyLogin(w, r, false) {
		return
	}
	if r.Method != "POST" {
		code = BAD_REQUEST
		err = errors.New("please use post method")
		return
	}
	addr := r.FormValue("addr")
	if len(addr) == 0 {
		err = errors.New("Node Address is Empty")
		return
	}
	r.PostForm.Add("token", m.token)
	var res *http.Response
	method := r.FormValue("method")
	switch method {
	case "post":
		res, err = http.PostForm(addr, r.PostForm)
		break
	case "get":
		res, err = http.Get(addr)
		break
	default:
		res, err = http.PostForm(addr, r.PostForm)
		break
	}
	if err != nil {
		if res != nil {
			return result, err, res.StatusCode
		}
		return result, err, 404
	}
	defer res.Body.Close()
	result, err = ioutil.ReadAll(res.Body)
	if err != nil {
		log.Debugf("node error: %s", err.Error())
		return result, err, SERVER_ERROR
	}
	return
}

func (m *Monitor) getAllNode(w http.ResponseWriter, r *http.Request) (result []byte, err error, code int) {
	cs := make([]Conn, 0)
	m.factory.ForEachAcceptedConnection(func(key cipher.PubKey, conn *factory.Connection) {
		now := time.Now().Unix()
		content := Conn{
			Key:         key.Hex(),
			SendBytes:   conn.GetSentBytes(),
			RecvBytes:   conn.GetReceivedBytes(),
			StartTime:   now - conn.GetConnectTime(),
			LastAckTime: now - conn.GetLastTime()}
		if conn.IsTCP() {
			content.Type = "TCP"
		} else {
			content.Type = "UDP"
		}
		cs = append(cs, content)
	})
	result, err = json.Marshal(cs)
	if err != nil {
		code = SERVER_ERROR
		return
	}
	return
}

func (m *Monitor) getNode(w http.ResponseWriter, r *http.Request) (result []byte, err error, code int) {
	if !verifyLogin(w, r, false) {
		return
	}
	if r.Method != "POST" {
		code = BAD_REQUEST
		err = errors.New("please use post method")
		return
	}
	key, err := cipher.PubKeyFromHex(r.FormValue("key"))
	if err != nil {
		code = BAD_REQUEST
		return
	}
	c, ok := m.factory.GetConnection(key)
	if !ok {
		code = NOT_FOUND
		err = errors.New("No connection is found")
		return
	}
	now := time.Now().Unix()
	nodeService := NodeServices{
		SendBytes:   c.GetSentBytes(),
		RecvBytes:   c.GetReceivedBytes(),
		StartTime:   now - c.GetConnectTime(),
		LastAckTime: now - c.GetLastTime()}
	if c.IsTCP() {
		nodeService.Type = "TCP"
	} else {
		nodeService.Type = "UDP"
	}
	v, ok := c.LoadContext("node-api")
	if ok {
		webPort, ok := v.(string)
		if ok && len(webPort) > 1 {
			var host, port string
			host, _, err = net.SplitHostPort(c.GetRemoteAddr().String())
			if err != nil {
				code = SERVER_ERROR
				return
			}
			_, port, err = net.SplitHostPort(webPort)
			if err != nil {
				code = SERVER_ERROR
				return
			}
			nodeService.Addr = net.JoinHostPort(host, port)
		}
	}
	result, err = json.Marshal(nodeService)
	if err != nil {
		code = SERVER_ERROR
		return
	}
	return
}

type Config struct {
	DiscoveryAddresses []string
}

func (m *Monitor) setNodeConfig(w http.ResponseWriter, r *http.Request) (result []byte, err error, code int) {
	if !verifyLogin(w, r, false) {
		return
	}
	if r.Method != "POST" {
		code = BAD_REQUEST
		err = errors.New("please use post method")
		return
	}
	key := r.FormValue("key")
	if len(key) != 66 {
		err = errors.New("Key at least 66 characters")
		return
	}
	data := []byte(r.FormValue("data"))
	var config *Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return
	}
	m.configsMutex.Lock()
	m.configs[key] = config
	m.configsMutex.Unlock()
	result = []byte("true")
	return
}

func (m *Monitor) getNodeConfig(w http.ResponseWriter, r *http.Request) (result []byte, err error, code int) {
	if !verifyLogin(w, r, false) {
		return
	}
	if r.Method != "POST" {
		code = BAD_REQUEST
		err = errors.New("please use post method")
		return
	}
	key := r.FormValue("key")
	if len(key) != 66 {
		err = errors.New("Key at least 66 characters")
		return
	}
	m.configsMutex.Lock()
	defer m.configsMutex.Unlock()
	conf, ok := m.configs[key]
	if !ok {
		err = errors.New("no found")
		return
	}
	result, err = json.Marshal(conf)
	if err != nil {
		return
	}
	return
}

type ClientConnection struct {
	Label   string `json:"label"`
	NodeKey string `json:"nodeKey"`
	AppKey  string `json:"appKey"`
	Count   int    `json:"count"`
}
type clientConnectionSlice []ClientConnection

func (c clientConnectionSlice) Len() int           { return len(c) }
func (c clientConnectionSlice) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }
func (c clientConnectionSlice) Less(i, j int) bool { return c[i].Count > c[j].Count }
func (c clientConnectionSlice) Exist(rf ClientConnection) bool {
	for k, v := range c {
		if v.AppKey == rf.AppKey && v.NodeKey == rf.NodeKey {
			c[k].Count++
			return true
		}
	}
	return false
}

var clientPath = filepath.Join(file.UserHome(), ".skywire", "manager", "clients.json")
var clientLimit = 5

func (m *Monitor) SaveClientConnection(w http.ResponseWriter, r *http.Request) (result []byte, err error, code int) {
	if !verifyLogin(w, r, false) {
		return
	}
	data := r.FormValue("data")
	client := r.FormValue("client")
	config := ClientConnection{}
	err = json.Unmarshal([]byte(data), &config)
	if err != nil {
		return
	}
	cfs, err := readConfig()
	if err != nil {
		return
	}
	if cfs == nil {
		cfs = make(map[string]clientConnectionSlice)
	}
	cf := cfs[client]
	size := len(cf)
	isExist := false
	if size == clientLimit {
		isExist = cf.Exist(config)
		if !isExist {
			cf[4] = config
		}
	} else if size > 0 && size < clientLimit {
		isExist = cf.Exist(config)
		if !isExist {
			cf = append(cf, config)
		}
	} else {
		cf = append(cf, config)
	}
	sort.Sort(cf)
	cfs[client] = cf
	err = saveClientFile(cfs)
	if err != nil {
		return
	}
	result = []byte("true")
	return
}

func (m *Monitor) GetClientConnection(w http.ResponseWriter, r *http.Request) (result []byte, err error, code int) {
	if !verifyLogin(w, r, false) {
		return
	}
	client := r.FormValue("client")
	cfs, err := readConfig()
	result, err = json.Marshal(cfs[client])
	return
}

func (m *Monitor) RemoveClientConnection(w http.ResponseWriter, r *http.Request) (result []byte, err error, code int) {
	if !verifyLogin(w, r, false) {
		return
	}
	client := r.FormValue("client")
	index, err := strconv.Atoi(r.FormValue("index"))
	if err != nil {
		return
	}
	cfs, err := readConfig()
	if err != nil && !os.IsNotExist(err) {
		return
	}
	cf, ok := cfs[client]
	if !ok {
		errors.New("System Error")
		return
	}
	cf = append(cf[:index], cf[index+1:]...)
	cfs[client] = cf
	err = saveClientFile(cfs)
	if err != nil {
		return
	}
	result = []byte("true")
	return
}

func (m *Monitor) EditClientConnection(w http.ResponseWriter, r *http.Request) (result []byte, err error, code int) {
	if !verifyLogin(w, r, false) {
		return
	}
	client := r.FormValue("client")
	label := r.FormValue("label")
	index, err := strconv.Atoi(r.FormValue("index"))
	if err != nil {
		return
	}

	cfs, err := readConfig()
	if err != nil && !os.IsNotExist(err) {
		return
	}
	cf, ok := cfs[client]
	if !ok {
		errors.New("System Error")
		return
	}
	cf[index].Label = label
	cfs[client] = cf
	err = saveClientFile(cfs)
	if err != nil {
		return
	}
	result = []byte("true")
	return
}

func readConfig() (cfs map[string]clientConnectionSlice, err error) {
	fb, err := ioutil.ReadFile(clientPath)
	if err != nil {
		if os.IsNotExist(err) {
			cfs = nil
			err = nil
			saveClientFile(cfs)
			return
		} else {
			return
		}
	}
	cfs = make(map[string]clientConnectionSlice)
	err = json.Unmarshal(fb, &cfs)
	if err != nil {
		return
	}
	return
}

func saveClientFile(data interface{}) (err error) {
	d, err := json.Marshal(data)
	if err != nil {
		return
	}
	dir := filepath.Dir(clientPath)
	err = os.MkdirAll(dir, 0700)
	if err != nil {
		return
	}
	err = ioutil.WriteFile(clientPath, d, 0600)
	return
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (m *Monitor) handleNodeTerm(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query()["token"][0]
	if len(token) == 0 {
		http.Error(w, "Token is Empty", http.StatusBadRequest)
		return
	}
	if !verifyWs(w, r, token) {
		return
	}
	url := r.URL.Query()["url"][0]
	if len(url) == 0 {
		http.Error(w, "Url is Empty", http.StatusBadRequest)
		return
	}
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Errorf("ws error: %s", err.Error())
		conn.WriteMessage(websocket.TextMessage, []byte(err.Error()))
		return
	}
	header := http.Header{}
	header.Add("manager-token", m.token)
	c, _, err := websocket.DefaultDialer.Dial(string(url), header)
	if err != nil {
		log.Errorf("node connection error: %s", err.Error())
		conn.WriteMessage(websocket.BinaryMessage, []byte(fmt.Sprintf("node connection error: %s", err.Error())))
		return
	}
	go func() {
		defer func() {
			conn.Close()
			c.Close()
		}()
		for {
			messageType, p, err := c.ReadMessage()
			if err != nil {
				return
			}
			conn.WriteMessage(messageType, p)
		}
	}()
	go func() {
		defer func() {
			conn.Close()
			c.Close()
		}()
		for {
			messageType, p, err := conn.ReadMessage()
			if err != nil {
				return
			}
			c.WriteMessage(messageType, p)
		}
	}()
}

var userPath = filepath.Join(file.UserHome(), ".skywire", "manager", "user.json")

func (m *Monitor) checkLogin(w http.ResponseWriter, r *http.Request) (result []byte, err error, code int) {
	if !verifyLogin(w, r, false) {
		result = []byte("false")
		return
	}
	sess, _ := globalSessions.SessionStart(w, r)
	defer sess.SessionRelease(w)
	result = []byte(sess.SessionID())
	return
}

func (m *Monitor) Login(w http.ResponseWriter, r *http.Request) (result []byte, err error, code int) {
	sess, _ := globalSessions.SessionStart(w, r)
	defer sess.SessionRelease(w)
	pass := r.FormValue("pass")
	if len(pass) < 4 || len(pass) > 20 {
		result = []byte("false")
		return
	}
	err = checkPass(pass)
	if err != nil {
		result = []byte("false")
		return
	}
	err = sess.Set("user", sess.SessionID())
	if err != nil {
		return
	}
	err = sess.Set("pass", getBcrypt(sess.SessionID()))
	if err != nil {
		return
	}
	result = []byte("true")
	return
}
func (m *Monitor) UpdatePass(w http.ResponseWriter, r *http.Request) (result []byte, err error, code int) {
	if !verifyLogin(w, r, true) {
		return
	}
	oldPass := r.FormValue("oldPass")
	newPass := r.FormValue("newPass")
	// Check that the old password was provided. We dont care about its length, as long as its not zero.
	if len(oldPass) < 1 {
		result = []byte("Your existing (old) password must be provided.")
		return
	}
	// Check the length of the new password. We dont care about how long the user makes it, as long as its not zero
	if len(newPass) < 1 {
		result = []byte("A new password must be provided.")
		return
	}
	if isDefaultPassCleartext(newPass) {
		result = []byte("The default password cannot be used.")
		return
	}
	err = checkPass(oldPass)
	if err != nil {
		result = []byte("The original password is wrong, please confirm and try again.")
		return
	}
	err = WriteConfig(&User{Pass: getBcrypt(newPass)}, userPath)
	if err != nil {
		result = []byte("The server is busy. Please try again later.")
		return
	}
	globalSessions.SessionDestroy(w, r)
	result = []byte("true")
	return
}

func verifyWs(w http.ResponseWriter, r *http.Request, token string) bool {
	sess, _ := globalSessions.GetSessionStore(token)
	defer sess.SessionRelease(w)
	pass := sess.Get("user")
	if pass == nil {
		http.Error(w, "Unauthorized", http.StatusFound)
		return false
	}
	hash := sess.Get("pass")
	if pass == nil {
		http.Error(w, "Unauthorized", http.StatusFound)
		return false
	}
	hashStr, ok := hash.(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusFound)
		return false
	}
	passStr, ok := pass.(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusFound)
		return false
	}
	return matchPassword(hashStr, passStr)
}

func verifyLogin(w http.ResponseWriter, r *http.Request, checkDefaultPass bool) bool {

	sess, _ := globalSessions.SessionStart(w, r)
	defer sess.SessionRelease(w)
	pass := sess.Get("user")
	if pass == nil {
		http.Error(w, "Unauthorized", http.StatusFound)
		return false
	}
	hash := sess.Get("pass")
	if hash == nil {
		http.Error(w, "Unauthorized", http.StatusFound)
		return false
	}
	hashStr, ok := hash.(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusFound)
		return false
	}
	passStr, ok := pass.(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusFound)
		return false
	}
	if !matchPassword(hashStr, passStr) {
		http.Error(w, "Unauthorized", http.StatusFound)
		return false
	}
	if !checkDefaultPass && userHasDefaultPass() {
		http.Error(w, "Forced change password", http.StatusTemporaryRedirect)
		return false
	}
	return true
}

func (m *Monitor) getServerInfo(w http.ResponseWriter, r *http.Request) (result []byte, err error, code int) {
	sc := m.factory.GetDefaultSeedConfig()
	if sc == nil {
		err = errors.New("Unable to get configuration")
		return
	}
	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		err = errors.New("Unable to get host")
		return
	}
	_, port, err := net.SplitHostPort(m.serverAddress)
	if err != nil {
		err = errors.New("Unable to get port")
		return
	}
	result = []byte(fmt.Sprintf("%s:%s-%s", host, port, sc.PublicKey))
	return
}
