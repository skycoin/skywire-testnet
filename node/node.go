package node

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/skycoin/net/skycoin-messenger/factory"
	"github.com/skycoin/skycoin/src/cipher"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Addresses []string

func (addrs *Addresses) String() string {
	return fmt.Sprintf("%v", []string(*addrs))
}

func (addrs *Addresses) Set(addr string) error {
	*addrs = append(*addrs, addr)
	return nil
}

type Node struct {
	apps             *factory.MessengerFactory
	manager          *factory.MessengerFactory
	seedConfigPath   string
	launchConfigPath string
	webPort          string
	lnAddr           string

	discoveries   Addresses
	onDiscoveries sync.Map

	srs      []*SearchResult
	srsMutex sync.Mutex
}

func New(seedPath, launchConfigPath, webPort string) *Node {
	apps := factory.NewMessengerFactory()
	apps.SetLoggerLevel(factory.DebugLevel)
	apps.Proxy = true
	apps.SetDefaultSeedConfigPath(seedPath)
	m := factory.NewMessengerFactory()
	m.SetDefaultSeedConfigPath(seedPath)
	return &Node{
		apps:             apps,
		manager:          m,
		seedConfigPath:   seedPath,
		launchConfigPath: launchConfigPath,
		webPort:          webPort,
	}
}

func (n *Node) GetManager() *factory.MessengerFactory {
	return n.manager
}

func (n *Node) Close() {
	n.apps.Close()
	n.manager.Close()
}

func (n *Node) Start(discoveries Addresses, address string) (err error) {
	n.discoveries = discoveries
	n.lnAddr = address
	err = n.apps.Listen(address)
	if err != nil {
		go func() {
			for {
				err := n.apps.Listen(address)
				if err != nil {
					time.Sleep(1000 * time.Millisecond)
					log.Errorf("failed to listen addr(%s) err %v", address, err)
				}
			}
		}()
	}

	for _, addr := range discoveries {
		n.onDiscoveries.Store(addr, false)
		split := strings.Split(addr, "-")
		if len(split) != 2 {
			err = fmt.Errorf("discovery address %s is not valid", addr)
			return err
		}
		tk, err := cipher.PubKeyFromHex(split[1])
		if err != nil {
			err = fmt.Errorf("discovery address %s is not valid", addr)
			return err
		}
		err = n.apps.ConnectWithConfig(split[0], &factory.ConnConfig{
			TargetKey:     tk,
			Reconnect:     true,
			ReconnectWait: 10 * time.Second,
			OnConnected: func(connection *factory.Connection) {
				go func() {
					for {
						select {
						case m, ok := <-connection.GetChanIn():
							if !ok {
								return
							}
							log.Debugf("discoveries:%x", m)
						}
					}
				}()
				n.apps.ResyncToDiscovery(connection)
				n.onDiscoveries.Store(addr, true)
			},
			OnDisconnected: func(connection *factory.Connection) {
				n.onDiscoveries.Store(addr, false)
			},
			FindServiceNodesByAttributesCallback: n.searchResultCallback,
		})
		if err != nil {
			log.Errorf("failed to connect addr(%s) err %v", addr, err)
		}
		return err
	}
	return
}

func (n *Node) ConnectManager(managerAddr string) (err error) {
	err = n.manager.ConnectWithConfig(managerAddr, &factory.ConnConfig{
		Context:       map[string]string{"node-api": n.webPort},
		Reconnect:     true,
		ReconnectWait: 10 * time.Second,
		OnConnected: func(connection *factory.Connection) {
			go func() {
				for {
					select {
					case m, ok := <-connection.GetChanIn():
						if !ok {
							return
						}
						log.Debugf("discoveries:%x", m)
					}
				}
			}()
		},
	})
	if err != nil {
		log.Errorf("failed to connect Manager addr(%s) err %v", managerAddr, err)
		return
	}
	return
}

func (n *Node) GetListenAddress() string {
	return n.lnAddr
}

type NodeTransport struct {
	FromNode string `json:"from_node"`
	ToNode   string `json:"to_node"`
	FromApp  string `json:"from_app"`
	ToApp    string `json:"to_app"`

	UploadBW      uint `json:"upload_bandwidth"`
	DownloadBW    uint `json:"download_bandwidth"`
	UploadTotal   uint `json:"upload_total"`
	DownloadTotal uint `json:"download_total"`
}

type NodeInfo struct {
	Discoveries  map[string]bool `json:"discoveries"`
	Transports   []NodeTransport `json:"transports"`
	AppFeedbacks []FeedBackItem  `json:"app_feedbacks"`
	Version      string          `json:"version"`
	Tag          string          `json:"tag"`
}

type FeedBackItem struct {
	Key            string `json:"key"`
	Port           int    `json:"port"`
	Failed         bool   `json:"failed"`
	UnreadMessages int    `json:"unread"`
}

var version = "0.0.1"
var tag = "dev"

func (n *Node) GetNodeInfo() (ni NodeInfo) {
	var ts []NodeTransport
	var afs []FeedBackItem
	n.apps.ForEachAcceptedConnection(func(key cipher.PubKey, conn *factory.Connection) {
		conn.ForEachTransport(func(v *factory.Transport) {
			ts = append(ts, NodeTransport{
				FromNode:      v.FromNode.Hex(),
				ToNode:        v.ToNode.Hex(),
				FromApp:       v.FromApp.Hex(),
				ToApp:         v.ToApp.Hex(),
				UploadBW:      v.GetUploadBandwidth(),
				DownloadBW:    v.GetDownloadBandwidth(),
				UploadTotal:   v.GetUploadTotal(),
				DownloadTotal: v.GetDownloadTotal(),
			})
		})
		feedback := conn.GetAppFeedback()
		if feedback != nil {
			afs = append(afs, FeedBackItem{
				Key:            key.Hex(),
				Port:           feedback.Port,
				Failed:         feedback.Failed,
				UnreadMessages: conn.CheckMessages(),
			})
		}
	})

	d := make(map[string]bool)
	n.onDiscoveries.Range(func(key, value interface{}) bool {
		k, ok := key.(string)
		if !ok {
			return true
		}
		v, ok := value.(bool)
		if !ok {
			return true
		}
		d[k] = v
		return true
	})
	ni = NodeInfo{
		Discoveries:  d,
		Transports:   ts,
		AppFeedbacks: afs,
		Version:      version,
		Tag:          tag,
	}
	return
}

func (n *Node) GetMessages(key cipher.PubKey) []factory.PriorityMsg {
	c, ok := n.apps.GetConnection(key)
	if ok {
		return c.GetMessages()
	}
	return nil
}

type NodeApp struct {
	Key        string   `json:"key"`
	Attributes []string `json:"attributes"`
	AllowNodes []string `json:"allow_nodes"`
}

func (n *Node) GetApps() (apps []NodeApp) {
	n.apps.ForEachAcceptedConnection(func(key cipher.PubKey, conn *factory.Connection) {
		ns := conn.GetServices()
		if ns == nil || len(ns.Services) == 0 {
			return
		}
		for _, v := range ns.Services {
			apps = append(apps, NodeApp{
				Key:        v.Key.Hex(),
				Attributes: v.Attributes,
				AllowNodes: v.AllowNodes,
			})
		}
	})
	return
}

type SearchResult struct {
	Result map[string][]string `json:"result"`
	Seq    uint32              `json:"seq"`
}

type SearchResultInfo struct {
	NodeKey string   `json:"node_key"`
	Apps    []string `json:"apps"`
}

func (n *Node) Search(attr string) (seqs []uint32) {
	n.apps.ForEachConn(func(connection *factory.Connection) {
		s, err := connection.FindServiceNodesWithSeqByAttributes(attr)
		if err != nil {
			log.Error(err)
			return
		}
		seqs = append(seqs, s)
	})
	return
}

func (n *Node) searchResultCallback(resp *factory.QueryByAttrsResp) {
	n.srsMutex.Lock()
	log.Infof("search call back: %#v", resp)
	var result = make(map[string][]string)
	for k, v := range resp.Result {
		var apps = make([]string, 0, len(v))
		for _, v1 := range v {
			apps = append(apps, v1.Hex())
		}
		result[k] = apps
	}
	n.srs = append(n.srs, &SearchResult{
		Seq:    resp.Seq,
		Result: result,
	})
	n.srsMutex.Unlock()
}

func (n *Node) GetSearchResult() (result []*SearchResult) {
	n.srsMutex.Lock()
	result = n.srs
	n.srs = nil
	n.srsMutex.Unlock()
	return
}

type AutoStartConfig struct {
	SocksServer bool `json:"socks_server"`
	SshServer   bool `json:"ssh_server"`
}

func (n *Node) NewAutoStartConfig() map[string]string {
	sc := make(map[string]string)
	sc["sshs"] = "false"
	sc["sshc"] = "false"
	sc["sshc_conf"] = ""
	sc["sshc_conf_nodeKey"] = ""
	sc["sshc_conf_appKey"] = ""
	sc["sockss"] = "true"
	sc["socksc"] = "false"
	sc["socksc_conf"] = ""
	sc["socksc_conf_nodeKey"] = ""
	sc["socksc_conf_appKey"] = ""
	return sc
}

func (n *Node) ReadAutoStartConfig() (lc map[string]string, err error) {
	fb, err := ioutil.ReadFile(n.launchConfigPath)
	if err != nil {
		return
	}
	err = json.Unmarshal(fb, &lc)
	return
}

func (n *Node) WriteAutoStartConfig(lc map[string]string, path string) (err error) {
	d, err := json.Marshal(lc)
	if err != nil {
		return
	}
	dir := filepath.Dir(path)
	err = os.MkdirAll(dir, 0700)
	if err != nil {
		return
	}
	err = ioutil.WriteFile(path, d, 0600)
	return
}
