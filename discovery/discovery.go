package discovery

import (
	"io/ioutil"
	"os"

	"github.com/gogap/logrus_mate"
	_ "github.com/gogap/logrus_mate/hooks/file"
	"github.com/sirupsen/logrus"
	"github.com/skycoin/net/skycoin-messenger/factory"
	"github.com/skycoin/net/skycoin-messenger/monitor"
	"github.com/skycoin/skywire/discovery/db"
)

type Discovery struct {
	messenger *factory.MessengerFactory
	monitor   *monitor.Monitor

	address string
	webDir  string
}

func New(seedPath, address, webAddress, webDir string) *Discovery {
	if _, err := os.Stat(logFilePath); os.IsNotExist(err) {
		err := ioutil.WriteFile(logFilePath, []byte(logConf), 0660)
		if err != nil {
			logrus.Fatal(err)
			return nil
		}
	} else if err != nil {
		logrus.Fatal(err)
	}

	mate, err := logrus_mate.NewLogrusMate(
		logrus_mate.ConfigFile(logFilePath),
	)
	if err != nil {
		logrus.Fatal(err)
		return nil
	}
	mate.Hijack(logrus.StandardLogger(), "discovery")
	m := factory.NewMessengerFactory()
	m.DisableLogger()
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
	err = db.Init()
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
