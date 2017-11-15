package node

import (
	"fmt"

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
}

func New(seedPath, webPort string) *Node {
	apps := factory.NewMessengerFactory()
	apps.SetLoggerLevel(factory.DebugLevel)
	apps.Proxy = true
	m := factory.NewMessengerFactory()
	return &Node{
		apps:           apps,
		manager:        m,
		seedConfigPath: seedPath,
		webPort:        webPort,
	}
}

func (n *Node) Start(discoveries Addresses, address string) (err error) {
	n.lnAddr = address
	err = n.apps.Listen(address)
	if err != nil {
		return
	}

	for _, addr := range discoveries {
		_, err = n.apps.ConnectWithConfig(addr, &factory.ConnConfig{
			SeedConfigPath: n.seedConfigPath,
			Reconnect:      true,
			ReconnectWait:  10 * time.Second,
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
			log.Errorf("failed to connect addr(%s) err %v", addr, err)
			return
		}
	}

	return
}

func (n *Node) ConnectManager(managerAddr string) (err error) {
	_, err = n.manager.ConnectWithConfig(managerAddr, &factory.ConnConfig{
		Context:        map[string]string{"node-api": n.webPort},
		SeedConfigPath: n.seedConfigPath,
		Reconnect:      true,
		ReconnectWait:  10 * time.Second,
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
	FromNode    string `json:"from_node"`
	ToNode      string `json:"to_node"`
	FromApp     string `json:"from_app"`
	ToApp       string `json:"to_app"`
	ServingPort string `json:"serving_port"`
}

type NodeInfo struct {
	Transports   []NodeTransport         `json:"transports"`
	Messages     [][]factory.PriorityMsg `json:"messages"`
	AppFeedbacks []FeedBackItem          `json:"app_feedbacks"`
	Version      string                  `json:"version"`
	Tag          string                  `json:"tag"`
}

type FeedBackItem struct {
	Key       string               `json:"key"`
	Feedbacks *factory.AppFeedback `json:"feedbacks"`
}

var version = "0.0.1"
var tag = "dev"

func (n *Node) GetNodeInfo() (ni NodeInfo) {
	var ts []NodeTransport
	var msgs [][]factory.PriorityMsg
	var afs []FeedBackItem
	n.apps.ForEachAcceptedConnection(func(key cipher.PubKey, conn *factory.Connection) {
		for _, v := range conn.GetTransports() {
			ts = append(ts, NodeTransport{FromNode: v.FromNode.Hex(), ToNode: v.ToNode.Hex(), FromApp: v.FromApp.Hex(), ToApp: v.ToApp.Hex()})
			msgs = append(msgs, conn.GetMessages())
			afs = append(afs, FeedBackItem{Key: key.Hex(), Feedbacks: conn.GetAppFeedback()})
		}
	})
	ni = NodeInfo{Transports: ts, Messages: msgs, AppFeedbacks: afs, Version: version, Tag: tag}
	return
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
