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
	apps   *factory.MessengerFactory
}

func New() *Node {
	apps := factory.NewMessengerFactory()
	apps.SetLoggerLevel(factory.DebugLevel)
	apps.Proxy = true
	return &Node{apps:apps}
}

func (n *Node) Start(discoveries Addresses, address string) (err error) {
	err = n.apps.Listen(address)
	if err != nil {
		return
	}

	for _, addr := range discoveries {
		n.apps.ConnectWithConfig(addr, &factory.ConnConfig{
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
	}

	return
}
