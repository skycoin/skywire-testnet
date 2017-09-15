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
	server      bool

	AppConnectionInitCallback func(resp *factory.AppConnResp)
}

func New(server bool, service, addr string) *App {
	messengerFactory := factory.NewMessengerFactory()
	messengerFactory.SetLoggerLevel(factory.DebugLevel)
	return &App{net: messengerFactory, service: service, serviceAddr: addr, server: server}
}

func (app *App) Start(addr string) {
	app.net.ConnectWithConfig(addr, &factory.ConnConfig{
		Reconnect:     true,
		ReconnectWait: 10 * time.Second,
		OnConnected: func(connection *factory.Connection) {
			if app.server {
				connection.OfferServiceWithAddress(app.serviceAddr, app.service)
			} else {
				connection.FindServiceNodesByAttributes(app.service)
			}
		},
		FindServiceNodesByAttributesCallback: app.FindServiceByAttributesCallback,
		AppConnectionInitCallback:            app.AppConnectionInitCallback,
	})
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
