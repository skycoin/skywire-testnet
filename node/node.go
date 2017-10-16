package node

import (
	"fmt"

	"time"

	log "github.com/sirupsen/logrus"
	"github.com/skycoin/net/skycoin-messenger/factory"
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
}

func New(seedPath string) *Node {
	apps := factory.NewMessengerFactory()
	apps.SetLoggerLevel(factory.DebugLevel)
	apps.Proxy = true
	return &Node{apps: apps, seedConfigPath: seedPath}
}

func (n *Node) Start(discoveries Addresses, address string) (err error) {
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
		SkipFactoryReg: true,
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
