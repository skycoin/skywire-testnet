package discovery

import (
	"github.com/gogap/logrus_mate"
	_ "github.com/gogap/logrus_mate/hooks/file"
	"github.com/sirupsen/logrus"
	"github.com/skycoin/net/skycoin-messenger/factory"
	"github.com/skycoin/net/skycoin-messenger/monitor"
	"io/ioutil"
	"os"
)

type Discovery struct {
	messenger *factory.MessengerFactory
	monitor   *monitor.Monitor

	address string
	webDir  string
}

func New(seedPath, address, webAddress, webDir string) *Discovery {
	if _, e := os.Stat(logFilePath); os.IsNotExist(e) {
		e := ioutil.WriteFile(logFilePath, []byte(logConf), 0660)
		if e != nil {
			logrus.Fatal(e)
			return nil
		}
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
	m.SetDefaultSeedConfigPath(seedPath)
	m.SetLoggerLevel(factory.DebugLevel)
	m.SetAppVersion(Version)
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
