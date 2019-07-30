package node

import (
	"encoding/base64"
	"fmt"
	"path/filepath"
	"time"

	"github.com/skycoin/dmsg/cipher"
	"github.com/spf13/cobra"

	"github.com/skycoin/skywire/pkg/util/pathutil"
	"github.com/skycoin/skywire/pkg/visor"
)

func init() {
	RootCmd.AddCommand(genConfigCmd)
}

var (
	output        string
	replace       bool
	configLocType = pathutil.WorkingDirLoc
)

func init() {
	genConfigCmd.Flags().StringVarP(&output, "output", "o", "", "path of output config file. Uses default of 'type' flag if unspecified.")
	genConfigCmd.Flags().BoolVarP(&replace, "replace", "r", false, "whether to allow rewrite of a file that already exists.")
	genConfigCmd.Flags().VarP(&configLocType, "type", "m", fmt.Sprintf("config generation mode. Valid values: %v", pathutil.AllConfigLocationTypes()))
}

var genConfigCmd = &cobra.Command{
	Use:   "gen-config",
	Short: "Generates a config file",
	PreRun: func(_ *cobra.Command, _ []string) {
		if output == "" {
			output = pathutil.NodeDefaults().Get(configLocType)
			log.Infof("No 'output' set; using default path: %s", output)
		}
		var err error
		if output, err = filepath.Abs(output); err != nil {
			log.WithError(err).Fatalln("invalid output provided")
		}
	},
	Run: func(_ *cobra.Command, _ []string) {
		var conf *visor.Config
		switch configLocType {
		case pathutil.WorkingDirLoc:
			conf = defaultConfig()
		case pathutil.HomeLoc:
			conf = homeConfig()
		case pathutil.LocalLoc:
			conf = localConfig()
		default:
			log.Fatalln("invalid config type:", configLocType)
		}
		pathutil.WriteJSONConfig(conf, output, replace)
	},
}

func homeConfig() *visor.Config {
	c := defaultConfig()
	c.AppsPath = filepath.Join(pathutil.HomeDir(), ".skycoin/skywire/apps")
	c.Transport.LogStore.Location = filepath.Join(pathutil.HomeDir(), ".skycoin/skywire/transport_logs")
	c.Routing.Table.Location = filepath.Join(pathutil.HomeDir(), ".skycoin/skywire/routing.db")
	return c
}

func localConfig() *visor.Config {
	c := defaultConfig()
	c.AppsPath = "/usr/local/skycoin/skywire/apps"
	c.Transport.LogStore.Location = "/usr/local/skycoin/skywire/transport_logs"
	c.Routing.Table.Location = "/usr/local/skycoin/skywire/routing.db"
	return c
}

func defaultConfig() *visor.Config {
	conf := &visor.Config{}
	conf.Version = "1.0"

	pk, sk := cipher.GenerateKeyPair()
	conf.Node.StaticPubKey = pk
	conf.Node.StaticSecKey = sk

	conf.DMSG.Discovery = "https://dmsg.discovery.skywire.skycoin.net"
	conf.DMSG.ServerCount = 1

	passcode := base64.StdEncoding.EncodeToString(cipher.RandByte(8))
	conf.Apps = []visor.AppConfig{
		{App: "skychat", Version: "1.0", Port: 1, AutoStart: true, Args: []string{}},
		{App: "SSH", Version: "1.0", Port: 2, AutoStart: true, Args: []string{}},
		{App: "socksproxy", Version: "1.0", Port: 3, AutoStart: true, Args: []string{"-passcode", passcode}},
	}
	conf.TrustedNodes = []cipher.PubKey{}

	conf.Transport.Discovery = "https://transport.discovery.skywire.skycoin.net"
	conf.Transport.LogStore.Type = "file"
	conf.Transport.LogStore.Location = "./skywire/transport_logs"

	conf.Routing.RouteFinder = "https://routefinder.skywire.skycoin.net/"

	const defaultSetupNodePK = "0324579f003e6b4048bae2def4365e634d8e0e3054a20fc7af49daf2a179658557"
	sPK := cipher.PubKey{}
	if err := sPK.UnmarshalText([]byte(defaultSetupNodePK)); err != nil {
		log.WithError(err).Warnf("Failed to unmarshal default setup node public key %s", defaultSetupNodePK)
	}
	conf.Routing.SetupNodes = []cipher.PubKey{sPK}
	conf.Routing.Table.Type = "boltdb"
	conf.Routing.Table.Location = "./skywire/routing.db"
	conf.Routing.RouteFinderTimeout = visor.Duration(10 * time.Second)

	conf.Hypervisors = []visor.HypervisorConfig{}

	conf.Uptime.Tracker = ""

	conf.AppsPath = "./apps"
	conf.LocalPath = "./local"

	conf.LogLevel = "info"

	conf.ShutdownTimeout = visor.Duration(10 * time.Second)

	conf.Interfaces.RPCAddress = "localhost:3435"

	return conf
}
