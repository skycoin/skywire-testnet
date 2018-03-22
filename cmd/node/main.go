package main

import (
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/skycoin/skycoin/src/util/file"
	"github.com/skycoin/skywire/node"
	"github.com/skycoin/skywire/node/api"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
)

var (
	config   node.Config
	confPath string

	version bool
)

func parseFlags() {
	flag.StringVar(&config.Address, "address", ":5000", "address to listen on")
	flag.Var(&config.DiscoveryAddresses, "discovery-address", "addresses of discovery")
	flag.BoolVar(&config.ConnectManager, "connect-manager", true, "connect to manager if true")
	flag.StringVar(&config.ManagerAddr, "manager-address", ":5998", "address of node manager")
	flag.StringVar(&config.ManagerWeb, "manager-web", ":8000", "address of node manager")
	flag.BoolVar(&config.Seed, "seed", true, "use fixed seed to connect if true")
	flag.StringVar(&config.SeedPath, "seed-path", filepath.Join(file.UserHome(), ".skywire", "node", "keys.json"), "path to save seed info")
	flag.StringVar(&config.WebPort, "web-port", ":6001", "monitor web page port")
	flag.StringVar(&config.AutoStartPath, "auto-start-path", filepath.Join(file.UserHome(), ".skywire", "node", "autoStart.json"), "path to save launch info")
	flag.StringVar(&confPath, "conf", filepath.Join(file.UserHome(), ".skywire", "node", "conf.json"), "node default config")
	flag.BoolVar(&version, "v", false, "print current version")
	flag.Parse()
}

func main() {
	parseFlags()
	if version {
		fmt.Println(node.Version)
		return
	}

	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, os.Interrupt, os.Kill)
	var n *node.Node
	if !config.Seed {
		n = node.New("", config.AutoStartPath, config.WebPort)
	} else {
		if len(config.SeedPath) < 1 {
			config.SeedPath = filepath.Join(file.UserHome(), ".skywire", "node", "keys.json")
		}
		n = node.New(config.SeedPath, config.AutoStartPath, config.WebPort)
	}
	var err error
	if len(config.DiscoveryAddresses) == 0 {
		cfs := &node.NodeConfigs{}
		err = node.LoadConfig(cfs, confPath)
		if err != nil {
			log.Error(err)
		}
		key, err := n.GetNodeKey()
		if err != nil {
			log.Error(err)
		}
		conf, ok := cfs.Configs[key]
		if !ok {
			conf = node.NewNodeConf()
			if cfs.Configs == nil {
				cfs.Configs = make(map[string]*node.Config)
			}
			cfs.Configs[key] = conf
			node.WriteConfig(&cfs, confPath)
		}
		err = n.Start(conf.DiscoveryAddresses, config.Address)
		if err != nil {
			log.Error(err)
		}
	} else {
		err = n.Start(config.DiscoveryAddresses, config.Address)
		if err != nil {
			log.Error(err)
		}
	}
	defer n.Close()
	log.Debugf("listen on %s", config.Address)
	var na *api.NodeApi
	if config.ConnectManager {
		err = n.ConnectManager(config.ManagerAddr)
		if err != nil {
			log.Error(err)
		}
		var tokenUrl string
		if len(strings.Split(config.ManagerWeb, ":")) == 1 {
			tokenUrl = fmt.Sprintf("http://127.0.0.1%s/getToken", config.ManagerWeb)
		} else {
			tokenUrl = fmt.Sprintf("http://%s/getToken", config.ManagerWeb)
		}
		resp, err := http.Get(tokenUrl)
		if err != nil {
			log.Error(err)
		}
		defer resp.Body.Close()
		token, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Error(err)
		}
		na = api.New(config.WebPort, string(token), n, &config, confPath, osSignal)
		na.StartSrv()
		defer na.Close()
	}
	select {
	case signal := <-osSignal:
		if signal == os.Interrupt {
			log.Debugln("exit by signal Interrupt")
		} else if signal == os.Kill {
			log.Debugln("exit by signal Kill")
		}
	}
}
