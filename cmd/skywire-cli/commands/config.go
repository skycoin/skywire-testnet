package commands

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	"github.com/skycoin/skywire/internal/pathutil"

	"github.com/spf13/cobra"

	"github.com/skycoin/skywire/pkg/cipher"
	"github.com/skycoin/skywire/pkg/node"
)

func init() {
	rootCmd.AddCommand(configCmd)
}

var (
	configMode string
)

var configCmd = &cobra.Command{
	Use:   "config [skywire.json]",
	Short: "Generate default config file",
	Run: func(_ *cobra.Command, args []string) {
		configFile := "skywire.json"
		if len(args) > 0 {
			configFile = args[0]
		}

		var conf *node.Config
		switch configMode {
		case "local":
			conf = defaultConfig()
		case "home":
			conf = homeConfig()
		case "system":
			conf = systemConfig()
		}
		confFile, err := os.OpenFile(configFile, os.O_RDWR|os.O_CREATE, 0600)
		if err != nil {
			log.Fatal("Failed to create config files: ", err)
		}

		enc := json.NewEncoder(confFile)
		enc.SetIndent("", "  ")
		if err := enc.Encode(conf); err != nil {
			log.Fatal("Failed to encode json config: ", err)
		}

		log.Println("Done!")
	},
}

func homeConfig() *node.Config {
	c := defaultConfig()

	c.AppsPath = filepath.Join(pathutil.HomeDir(), ".skycoin/skywire/apps")
	c.Transport.LogStore.Location = filepath.Join(pathutil.HomeDir(), ".skycoin/skywire/transport_logs")
	c.Routing.Table.Location = filepath.Join(pathutil.HomeDir(), ".skycoin/skywire/routing.db")

	return c
}

func systemConfig() *node.Config {
	c := defaultConfig()

	c.AppsPath = "/usr/local/skycoin/skywire/apps"
	c.Transport.LogStore.Location = "/usr/local/skycoin/skywire/transport_logs"
	c.Routing.Table.Location = "/usr/local/skycoin/skywire/routing.db"

	return c
}

func defaultConfig() *node.Config {
	conf := &node.Config{}
	conf.Version = "1.0"

	pk, sk := cipher.GenerateKeyPair()
	conf.Node.StaticPubKey = pk
	conf.Node.StaticSecKey = sk

	conf.Messaging.Discovery = "https://messaging.discovery.skywire.skycoin.net"
	conf.Messaging.ServerCount = 1

	passcode := base64.StdEncoding.EncodeToString(cipher.RandByte(8))
	conf.Apps = []node.AppConfig{
		{App: "chat", Version: "1.0", Port: 1, AutoStart: true, Args: []string{}},
		{App: "therealssh", Version: "1.0", Port: 2, AutoStart: true, Args: []string{}},
		{App: "therealproxy", Version: "1.0", Port: 3, AutoStart: true, Args: []string{"-passcode", passcode}},
	}
	conf.TrustedNodes = []cipher.PubKey{}

	conf.Transport.Discovery = "https://transport.discovery.skywire.skycoin.net"
	conf.Transport.LogStore.Type = "file"
	conf.Transport.LogStore.Location = "./skywire/transport_logs"

	conf.Routing.RouteFinder = "https://routefinder.skywire.skycoin.net/"
	sPK := cipher.PubKey{}
	sPK.UnmarshalText([]byte("0324579f003e6b4048bae2def4365e634d8e0e3054a20fc7af49daf2a179658557")) // nolint: errcheck
	conf.Routing.SetupNodes = []cipher.PubKey{sPK}
	conf.Routing.Table.Type = "boltdb"
	conf.Routing.Table.Location = "./skywire/routing.db"

	conf.ManagerNodes = []node.ManagerConfig{}

	conf.AppsPath = "./apps"
	conf.LocalPath = "./local"

	conf.LogLevel = "info"

	conf.Interfaces.RPCAddress = "localhost:3435"

	return conf
}

func init() {
	configCmd.Flags().StringVar(&configMode, "mode", "home", "either home or local")
}
