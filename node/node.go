package node

import (
	"fmt"
	"sync"

	"time"

	log "github.com/sirupsen/logrus"
	"github.com/skycoin/net/skycoin-messenger/factory"
	"github.com/skycoin/skycoin/src/cipher"
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
	apps           *factory.MessengerFactory
	manager        *factory.MessengerFactory
	seedConfigPath string
	webPort        string
	lnAddr         string

	discoveries   Addresses
	onDiscoveries sync.Map

	srs      []*SearchResult
	srsMutex sync.Mutex
}

func New(seedPath, webPort string) *Node {
	apps := factory.NewMessengerFactory()
	apps.SetLoggerLevel(factory.DebugLevel)
	apps.Proxy = true
	apps.SetDefaultSeedConfigPath(seedPath)
	m := factory.NewMessengerFactory()
	m.SetDefaultSeedConfigPath(seedPath)
	return &Node{
		apps:           apps,
		manager:        m,
		seedConfigPath: seedPath,
		webPort:        webPort,
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
		func(addr string) {
			n.onDiscoveries.Store(addr, false)
			err := n.apps.ConnectWithConfig(addr, &factory.ConnConfig{
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
		}(addr)
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
			feedback := conn.GetAppFeedback()
			port := 0
			if feedback != nil {
				port = feedback.Port
			}
			afs = append(afs, FeedBackItem{
				Key:            key.Hex(),
				Port:           port,
				UnreadMessages: conn.CheckMessages(),
			})
		})
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
		for _, v := range ns.Services {
			apps = append(apps, NodeApp{Key: v.Key.Hex(), Attributes: v.Attributes, AllowNodes: v.AllowNodes})
		}
	})
	return
}

type SearchResult struct {
	Result map[string][]cipher.PubKey
	Seq    uint32
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
	n.srs = append(n.srs, &SearchResult{
		Seq:    resp.Seq,
		Result: resp.Result,
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
