package nodemanager

import "github.com/skycoin/net/skycoin-messenger/factory"

type Manager struct {
	net *factory.MessengerFactory
}

func New() *Manager {
	return &Manager{net:factory.NewMessengerFactory()}
}

func (manager *Manager) Listen(addr string) error {
	return manager.net.Listen(addr)
}
