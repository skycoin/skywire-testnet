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
	seedConfigPath string
	webPort        string
	lnAddr         string
}

func New(seedPath, webPort string) *Node {
	apps := factory.NewMessengerFactory()
	apps.SetLoggerLevel(factory.DebugLevel)
	apps.Proxy = true
	return &Node{apps: apps, seedConfigPath: seedPath, webPort: webPort}
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
	_, err = n.apps.ConnectWithConfig(managerAddr, &factory.ConnConfig{
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
	FromNode string `json:"from_node"`
	ToNode   string `json:"to_node"`
	FromApp  string `json:"from_app"`
	ToApp    string `json:"to_app"`
}

func (n *Node) GetTransport() (ts []NodeTransport) {
	n.apps.ForEachAcceptedConnection(func(key cipher.PubKey, conn *factory.Connection) {
		for _, v := range conn.GetTransports() {
			ts = append(ts, NodeTransport{FromNode: v.FromNode.Hex(), ToNode: v.ToNode.Hex(), FromApp: v.FromApp.Hex(), ToApp: v.ToApp.Hex()})
		}
	})
	return
}
