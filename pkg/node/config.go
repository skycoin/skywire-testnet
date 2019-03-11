package node

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/skycoin/skywire/pkg/cipher"
	mClient "github.com/skycoin/skywire/pkg/messaging-discovery/client"
	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/transport"
	trClient "github.com/skycoin/skywire/pkg/transport-discovery/client"
)

// Config defines configuration parameters for Node.
type Config struct {
	Version string `json:"version"`

	Node struct {
		StaticPubKey cipher.PubKey `json:"static_public_key"`
		StaticSecKey cipher.SecKey `json:"static_secret_key"`
	} `json:"node"`

	Messaging struct {
		Discovery   string `json:"discovery"`
		ServerCount int    `json:"server_count"`
	} `json:"messaging"`

	Transport struct {
		Discovery string `json:"discovery"`
		LogStore  struct {
			Type     string `json:"type"`
			Location string `json:"location"`
		} `json:"log_store"`
	} `json:"transport"`

	Routing struct {
		SetupNodes  []cipher.PubKey `json:"setup_nodes"`
		RouteFinder string          `json:"route_finder"`
		Table       struct {
			Type     string `json:"type"`
			Location string `json:"location"`
		} `json:"table"`
	} `json:"routing"`

	Apps []AppConfig `json:"apps"`

	TrustedNodes []cipher.PubKey `json:"trusted_nodes"`
	ManagerNodes []ManagerConfig `json:"manager_nodes"`

	AppsPath  string `json:"apps_path"`
	LocalPath string `json:"local_path"`

	LogLevel string `json:"log_level"`

	Interfaces InterfaceConfig `json:"interfaces"`
}

// MessagingDiscovery returns messaging discovery client.
func (c *Config) MessagingDiscovery() (mClient.APIClient, error) {
	msgConfig := c.Messaging

	if msgConfig.Discovery == "" {
		return nil, errors.New("empty discovery")
	}

	return mClient.NewHTTP(msgConfig.Discovery), nil
}

// TransportDiscovery returns transport discovery client.
func (c *Config) TransportDiscovery() (transport.DiscoveryClient, error) {
	if c.Transport.Discovery == "" {
		return nil, errors.New("empty transport_discovery")
	}

	return trClient.NewHTTP(c.Transport.Discovery, c.Node.StaticPubKey, c.Node.StaticSecKey)
}

// TransportLogStore returns configure transport.LogStore.
func (c *Config) TransportLogStore() (transport.LogStore, error) {
	if c.Transport.LogStore.Type == "file" {
		return transport.FileTransportLogStore(c.Transport.LogStore.Location)
	}

	return transport.InMemoryTransportLogStore(), nil
}

// RoutingTable returns configure routing.Table.
func (c *Config) RoutingTable() (routing.Table, error) {
	if c.Routing.Table.Type == "boltdb" {
		return routing.BoltDBRoutingTable(c.Routing.Table.Location)
	}

	return routing.InMemoryRoutingTable(), nil
}

// AppsConfig decodes AppsConfig from a local json config file.
func (c *Config) AppsConfig() ([]AppConfig, error) {
	apps := []AppConfig{}
	for _, app := range c.Apps {
		if app.Version == "" {
			app.Version = c.Version
		}
		apps = append(apps, app)
	}

	return apps, nil
}

// AppsDir returns absolute path for directory with application
// binaries. Directory will be created if necessary.
func (c *Config) AppsDir() (string, error) {
	if c.AppsPath == "" {
		return "", errors.New("empty AppsPath")
	}

	return ensureDir(c.AppsPath)
}

// LocalDir returns absolute path for app work directory. Directory
// will be created if necessary.
func (c *Config) LocalDir() (string, error) {
	if c.LocalPath == "" {
		return "", errors.New("empty AppsPath")
	}

	return ensureDir(c.LocalPath)
}

func ensureDir(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to expand path: %s", err)
	}

	if _, err := os.Stat(absPath); !os.IsNotExist(err) {
		return absPath, nil
	}

	if err := os.MkdirAll(absPath, 0750); err != nil {
		return "", fmt.Errorf("failed to create dir: %s", err)
	}

	return absPath, nil
}

// ManagerConfig represents a connection to a manager.
type ManagerConfig struct {
	PubKey cipher.PubKey `json:"public_key"`
	Addr   string        `json:"address"`
}

// AppConfig defines app startup parameters.
type AppConfig struct {
	Version   string   `json:"version"`
	App       string   `json:"app"`
	AutoStart bool     `json:"auto_start"`
	Port      uint16   `json:"port"`
	Args      []string `json:"args"`
}

// InterfaceConfig defines listening interfaces for skywire Node.
type InterfaceConfig struct {
	RPCAddress string `json:"rpc"` // RPC address and port for command-line interface (leave blank to disable RPC interface).
}
