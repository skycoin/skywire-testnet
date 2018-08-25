package discovery

import (
	"github.com/skycoin/net/skycoin-messenger/factory"
	"github.com/skycoin/net/skycoin-messenger/monitor"
	"github.com/skycoin/skywire/discovery/db"
)

type Discovery struct {
	messenger *factory.MessengerFactory
	monitor   *monitor.Monitor

	address string
	webDir  string

	ShowSQL     bool
	SQLLogLevel string
}

func New(seedPath, address, webAddress, webDir string) *Discovery {
	m := factory.NewMessengerFactory()
	m.SetDefaultSeedConfigPath(seedPath)
	m.SetAppVersion(Version)
	m.RegisterService = db.RegisterService
	m.UnRegisterService = db.UnRegisterService
	m.FindByAttributes = db.FindResultByAttrs
	m.FindByAttributesAndPaging = db.FindResultByAttrsAndPaging
	m.FindServiceAddresses = db.FindServiceAddresses
	mon := monitor.New(m, address, webAddress, "", "")
	return &Discovery{
		messenger: m,
		monitor:   mon,

		address: address,
		webDir:  webDir,
	}
}

func (d *Discovery) Start() (err error) {
	err = db.Init(d.ShowSQL, d.SQLLogLevel)
	if err != nil {
		return
	}
	err = d.messenger.Listen(d.address)
	if err != nil {
		return
	}
	d.monitor.Start(d.webDir)
	return
}

func (d *Discovery) Close() {
	d.monitor.Close()
	d.messenger.Close()
}

func (d *Discovery) GetDiscoveryKey() string {
	return d.messenger.GetDefaultSeedConfig().PublicKey
}
