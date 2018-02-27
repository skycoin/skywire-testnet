package discovery

import (
	"github.com/skycoin/net/skycoin-messenger/factory"
	"github.com/skycoin/net/skycoin-messenger/monitor"
)

type Discovery struct {
	messenger *factory.MessengerFactory
	monitor   *monitor.Monitor

	address string
	webDir  string
}

func New(seedPath, address, webAddress, webDir string) *Discovery {
	m := factory.NewMessengerFactory()
	m.SetDefaultSeedConfigPath(seedPath)
	m.SetLoggerLevel(factory.DebugLevel)
	mon := monitor.New(m, address, webAddress, "", "")
	return &Discovery{
		messenger: m,
		monitor:   mon,

		address: address,
		webDir:  webDir,
	}
}

func (d *Discovery) Start() (err error) {
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
