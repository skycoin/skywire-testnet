package app

import (
	"net"
	"sync"
	"time"

	"github.com/skycoin/skywire/messages"
)

type SocksClient struct {
	proxyClient
}

func NewSocksClient(appId messages.AppId, nodeAddr string, proxyAddress string) (*SocksClient, error) {
	setLimit(16384) // set limit of simultaneously opened files to 16384
	socksClient := &SocksClient{}
	socksClient.id = appId
	socksClient.appType = "socks_client"
	socksClient.lock = &sync.Mutex{}
	socksClient.timeout = time.Duration(messages.GetConfig().AppTimeout)
	socksClient.responseNodeAppChannels = make(map[uint32]chan []byte)

	err := socksClient.RegisterAtNode(nodeAddr)
	if err != nil {
		return nil, err
	}

	socksClient.connections = map[string]*net.Conn{}

	socksClient.ProxyAddress = proxyAddress

	return socksClient, nil
}

//SocksClient doesn't have differences from ProxyClient, only servers do
