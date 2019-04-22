package node

import (
	"errors"
	"time"

	"github.com/skycoin/skywire/pkg/util/pathutil"

	"github.com/skycoin/skywire/pkg/messaging"

	"github.com/skycoin/skywire/pkg/cipher"
	mClient "github.com/skycoin/skywire/pkg/messaging-discovery/client"
	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/transport"
	trClient "github.com/skycoin/skywire/pkg/transport-discovery/client"
)

// KeyFields is a member of Config.
type KeyFields struct {
	PubKey cipher.PubKey `json:"static_public_key"`
	SecKey cipher.SecKey `json:"static_secret_key"`
}

// MessagingFields is a member of Config.
type MessagingFields struct {
	Discovery   string `json:"discovery"`
	ServerCount int    `json:"server_count"`
}

// LogStoreFields is a member of TransportFields.
type LogStoreFields struct {
	Type     string `json:"type"`
	Location string `json:"location"`
}

// TransportFields is a member of Config.
type TransportFields struct {
	Discovery string         `json:"discovery"`
	LogStore  LogStoreFields `json:"log_store"`
}

// RoutingTableFields is a member of RoutingFields.
type RoutingTableFields struct {
	Type     string `json:"type"`
	Location string `json:"location"`
}

// RoutingFields is a member of Config.
type RoutingFields struct {
	SetupNodes  []cipher.PubKey    `json:"setup_nodes"`
	RouteFinder string             `json:"route_finder"`
	Table       RoutingTableFields `json:"table"`
}

// Config defines configuration parameters for Node.
type Config struct {
	Version   string          `json:"version"`
	Node      KeyFields       `json:"node"`
	Messaging MessagingFields `json:"messaging"`
	Transport TransportFields `json:"transport"`
	Routing   RoutingFields   `json:"routing"`

	Apps []AppConfig `json:"apps"`

	TrustedNodes []cipher.PubKey `json:"trusted_nodes"`
	ManagerNodes []ManagerConfig `json:"manager_nodes"`

	AppsPath  string `json:"apps_path"`
	LocalPath string `json:"local_path"`

	LogLevel string `json:"log_level"`

	Interfaces InterfaceConfig `json:"interfaces"`
}

// MessagingConfig returns config for messaging client.
func (c *Config) MessagingConfig() (*messaging.Config, error) {

	msgConfig := c.Messaging

	if msgConfig.Discovery == "" {
		return nil, errors.New("empty discovery")
	}

	return &messaging.Config{
		PubKey:     c.Node.PubKey,
		SecKey:     c.Node.SecKey,
		Discovery:  mClient.NewHTTP(msgConfig.Discovery),
		Retries:    5,
		RetryDelay: time.Second,
	}, nil
}

// TransportDiscovery returns transport discovery client.
func (c *Config) TransportDiscovery() (transport.DiscoveryClient, error) {
	if c.Transport.Discovery == "" {
		return nil, errors.New("empty transport_discovery")
	}

	return trClient.NewHTTP(c.Transport.Discovery, c.Node.PubKey, c.Node.SecKey)
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

// AppsDir returns absolute path for directory with application
// binaries. Directory will be created if necessary.
func (c *Config) AppsDir() (string, error) {
	if c.AppsPath == "" {
		return "", errors.New("empty AppsPath")
	}

	return pathutil.EnsureDir(c.AppsPath)
}

// LocalDir returns absolute path for app work directory. Directory
// will be created if necessary.
func (c *Config) LocalDir() (string, error) {
	if c.LocalPath == "" {
		return "", errors.New("empty AppsPath")
	}

	return pathutil.EnsureDir(c.LocalPath)
}

// ManagerConfig represents a connection to a manager.
type ManagerConfig struct {
	PubKey cipher.PubKey `json:"public_key"`
	Addr   string        `json:"address"`
}

// AppConfig defines app startup parameters.
type AppConfig struct {
	App       string   `json:"app"`
	AutoStart bool     `json:"auto_start"`
	Port      uint16   `json:"port"`
	Args      []string `json:"args"`
}

// InterfaceConfig defines listening interfaces for skywire Node.
type InterfaceConfig struct {
	RPCAddress string `json:"rpc"` // RPC address and port for command-line interface (leave blank to disable RPC interface).
}
