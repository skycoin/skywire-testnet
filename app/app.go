package app

import (
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/skycoin/net/skycoin-messenger/factory"
	"github.com/skycoin/skycoin/src/cipher"
)

type App struct {
	net         *factory.MessengerFactory
	service     string
	serviceAddr string
	appType     Type
	allowNodes  []string

	AppConnectionInitCallback func(resp *factory.AppConnResp)
}

type Type int

const (
	Client Type = iota
	Public
	Private
)

func New(appType Type, service, addr string) *App {
	messengerFactory := factory.NewMessengerFactory()
	messengerFactory.SetLoggerLevel(factory.DebugLevel)
	return &App{net: messengerFactory, service: service, serviceAddr: addr, appType: appType}
}

func (app *App) Start(addr, scPath string) error {
	_, err := app.net.ConnectWithConfig(addr, &factory.ConnConfig{
		SeedConfigPath: scPath,
		Reconnect:      true,
		ReconnectWait:  10 * time.Second,
		OnConnected: func(connection *factory.Connection) {
			switch app.appType {
			case Public:
				connection.OfferServiceWithAddress(app.serviceAddr, app.service)
			case Client:
				fallthrough
			case Private:
				connection.OfferPrivateServiceWithAddress(app.serviceAddr, app.allowNodes, app.service)
			}
		},
		FindServiceNodesByAttributesCallback: app.FindServiceByAttributesCallback,
		AppConnectionInitCallback:            app.AppConnectionInitCallback,
	})
	return err
}

func (app *App) FindServiceByAttributesCallback(resp *factory.QueryByAttrsResp) {
	log.Debugf("findServiceByAttributesCallback resp %#v", resp)
	if len(resp.Result) < 1 {
		return
	}
	for k, v := range resp.Result {
		log.Debugf("node %x %v", k, v)
	}
	for k, v := range resp.Result {
		node, err := cipher.PubKeyFromHex(k)
		if err != nil {
			log.Debugf("node key string invalid %s", k)
			continue
		}
		for _, a := range v {
			app.net.ForEachConn(func(connection *factory.Connection) {
				connection.BuildAppConnection(node, a)
			})
		}
		break
	}
}

func (app *App) SetAllowNodes(nodes []string) {
	app.allowNodes = nodes
}

func (app *App) ConnectTo(nodeKeyHex, appKeyHex string) (err error) {
	nodeKey, err := cipher.PubKeyFromHex(nodeKeyHex)
	if err != nil {
		return
	}
	appKey, err := cipher.PubKeyFromHex(appKeyHex)
	if err != nil {
		return
	}
	app.net.ForEachConn(func(connection *factory.Connection) {
		connection.BuildAppConnection(nodeKey, appKey)
	})
	return
}
