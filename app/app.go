package app

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/skycoin/net/skycoin-messenger/factory"
	"github.com/skycoin/skycoin/src/cipher"
)

type App struct {
	net         *factory.MessengerFactory
	service     string
	serviceAddr string
	appType     Type
	allowNodes  NodeKeys
	Version     string

	AppConnectionInitCallback func(resp *factory.AppConnResp) *factory.AppFeedback
}

type NodeKeys []string

func (keys *NodeKeys) String() string {
	return fmt.Sprintf("%v", []string(*keys))
}

func (keys *NodeKeys) Set(key string) error {
	*keys = append(*keys, key)
	return nil
}

type Type int

const (
	Client Type = iota
	Public
	Private
)

func NewServer(appType Type, service, addr, version string) *App {
	messengerFactory := factory.NewMessengerFactory()
	messengerFactory.SetLoggerLevel(factory.DebugLevel)
	return &App{
		net:         messengerFactory,
		service:     service,
		serviceAddr: addr,
		appType:     appType,
		Version:     version,
	}
}

func NewClient(appType Type, service, version string) *App {
	messengerFactory := factory.NewMessengerFactory()
	messengerFactory.SetLoggerLevel(factory.DebugLevel)
	return &App{
		net:     messengerFactory,
		service: service,
		appType: appType,
		Version: version,
	}
}

func (app *App) Start(addr, scPath string) error {
	err := app.net.ConnectWithConfig(addr, &factory.ConnConfig{
		SeedConfigPath: scPath,
		OnConnected: func(connection *factory.Connection) {
			switch app.appType {
			case Public:
				connection.OfferServiceWithAddress(app.serviceAddr, app.Version, app.service)
			case Client:
				fallthrough
			case Private:
				connection.OfferPrivateServiceWithAddress(app.serviceAddr, app.Version, app.allowNodes, app.service)
			}
		},
		OnDisconnected: func(connection *factory.Connection) {
			log.Debug("exit on disconnected")
			os.Exit(1)
		},
		FindServiceNodesByAttributesCallback: app.FindServiceByAttributesCallback,
		AppConnectionInitCallback:            app.AppConnectionInitCallback,
	})
	return err
}

func (app *App) FindServiceByAttributesCallback(resp *factory.QueryByAttrsResp) {
	log.Debugf("findServiceByAttributesCallback resp %#v", resp)
}

func (app *App) SetAllowNodes(nodes NodeKeys) {
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
