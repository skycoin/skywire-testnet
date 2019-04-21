// Package node implements skywire node.
package node

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/rpc"
	"time"

	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/internal/noise"
	"github.com/skycoin/skywire/pkg/messaging"
	routeFinder "github.com/skycoin/skywire/pkg/route-finder/client"
	"github.com/skycoin/skywire/pkg/router"
	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/transport"
)

// ErrUnknownApp represents lookup error for App related calls.
var ErrUnknownApp = errors.New("unknown app")

// Version is the node version.
const Version = "0.0.1"

const supportedProtocolVersion = "0.0.1"

// Node provides messaging runtime for Apps by setting up all
// necessary connections and performing messaging gateway functions.
type Node struct {
	c  *Config
	m  *messaging.Client
	tm *transport.Manager
	rt routing.Table
	r  router.Router
	am router.AppsManager

	Logger *logging.MasterLogger
	logger *logging.Logger

	rpcL net.Listener
	rpcD []*noise.RPCClientDialer
}

// NewNode constructs new Node.
func NewNode(config *Config) (*Node, error) {
	node := &Node{c: config}

	node.Logger = logging.NewMasterLogger()
	node.logger = node.Logger.PackageLogger("skywire")

	pk := config.Node.PubKey
	sk := config.Node.SecKey

	/* SETUP: MESSAGING */

	mConfig, err := config.MessagingConfig()
	if err != nil {
		return nil, fmt.Errorf("invalid Messaging config: %s", err)
	}

	node.m = messaging.NewClient(mConfig)
	node.m.Logger = node.Logger.PackageLogger("messenger")

	/* SETUP: TRANSPORT MANAGER */

	trDiscovery, err := config.TransportDiscovery()
	if err != nil {
		return nil, fmt.Errorf("invalid MessagingConfig: %s", err)
	}
	logStore, err := config.TransportLogStore()
	if err != nil {
		return nil, fmt.Errorf("invalid TransportLogStore: %s", err)
	}
	tmConfig := &transport.ManagerConfig{
		PubKey: pk, SecKey: sk,
		DiscoveryClient: trDiscovery,
		LogStore:        logStore,
		DefaultNodes:    config.TrustedNodes,
	}
	node.tm, err = transport.NewManager(tmConfig, node.m)
	if err != nil {
		return nil, fmt.Errorf("transport manager: %s", err)
	}
	node.tm.Logger = node.Logger.PackageLogger("tp_manager")

	/* SETUP: ROUTER */

	node.rt, err = config.RoutingTable()
	if err != nil {
		return nil, fmt.Errorf("routing table: %s", err)
	}
	rConf := &router.Config{
		PubKey:           pk,
		SecKey:           sk,
		SetupNodes:       config.Routing.SetupNodes,
	}
	r := router.New(node.Logger.PackageLogger("router"), node.tm, node.rt,routeFinder.NewHTTP(config.Routing.RouteFinder), rConf)
	node.r = r

	/* SETUP: APPS */

	binDir, err := config.AppsDir()
	if err != nil {
		return nil, fmt.Errorf("app manager: %s", err)
	}
	localDir, err := config.LocalDir()
	if err != nil {
		return nil, fmt.Errorf("app manager: %s", err)
	}

	appsMgr := router.NewAppsManager(rConf, r, 10, binDir, localDir)
	node.am = appsMgr
	for _, ac := range node.c.Apps {
		host, err := appsMgr.RegisterApp(ac.App, ac.Args)
		if err != nil {
			node.logger.Warnf("failed to setup app '%s': %s", ac.App, err)
			continue
		}
		if ac.Port != 0 {
			if err := appsMgr.AllocGivenPort(host, ac.Port); err != nil {
				node.logger.Warnf("failed to allocate port '%d' to app '%s': %s", ac.Port, ac.App, err)
			}
		}
	}

	if lvl, err := logging.LevelFromString(config.LogLevel); err == nil {
		node.Logger.SetLevel(lvl)
	}

	/* SETUP: MANAGER */

	if config.Interfaces.RPCAddress != "" {
		l, err := net.Listen("tcp", config.Interfaces.RPCAddress)
		if err != nil {
			return nil, fmt.Errorf("failed to setup RPC listener: %s", err)
		}
		node.rpcL = l
	}
	node.rpcD = make([]*noise.RPCClientDialer, len(config.ManagerNodes))
	for i, entry := range config.ManagerNodes {
		node.rpcD[i] = noise.NewRPCClientDialer(entry.Addr, noise.HandshakeXK, noise.Config{
			LocalPK:   pk,
			LocalSK:   sk,
			RemotePK:  entry.PubKey,
			Initiator: true,
		})
	}

	return node, err
}

// Start spawns auto-started Apps, starts router and RPC interfaces .
func (node *Node) Start() error {
	ctx := context.Background()

	/* START: MESSAGING */

	err := node.m.ConnectToInitialServers(ctx, node.c.Messaging.ServerCount)
	if err != nil {
		return fmt.Errorf("messaging: %s", err)
	}
	node.logger.Info("Connected to messaging servers")

	/* START: TRANSPORTS */

	node.tm.ReconnectTransports(ctx)
	node.tm.CreateDefaultTransports(ctx)

	/* START: APPS */

	for _, ac := range node.c.Apps {
		if !ac.AutoStart {
			continue
		}
		host, ok := node.am.AppOfName(ac.App)
		if !ok {
			continue
		}
		if err := host.Start(); err != nil {
			node.logger.Warnf("Failed to start %s: %s\n", ac.App, err)
		}
	}

	/* START: MANAGER */

	rpcSvr := rpc.NewServer()
	if err := rpcSvr.RegisterName(RPCPrefix, &RPC{node: node}); err != nil {
		return fmt.Errorf("rpc server created failed: %s", err)
	}
	if node.rpcL != nil {
		node.logger.Info("Starting RPC interface on ", node.rpcL.Addr())
		go rpcSvr.Accept(node.rpcL)
	}
	for _, dialer := range node.rpcD {
		go func(dialer *noise.RPCClientDialer) {
			if err := dialer.Run(rpcSvr, time.Second); err != nil {
				node.logger.Errorf("Dialer exited with error: %v", err)
			}
		}(dialer)
	}

	/* START: ROUTER */

	node.logger.Info("Starting packet router")
	if err := node.r.Serve(ctx, node.am); err != nil {
		return fmt.Errorf("failed to start Node: %s", err)
	}

	return nil
}

// Close safely stops spawned Apps and messaging Node.
func (node *Node) Close() (err error) {
	if node.rpcL != nil {
		node.logger.Info("Stopping RPC interface")
		err = node.rpcL.Close()
	}
	for _, dialer := range node.rpcD {
		err = dialer.Close()
	}

	err = node.am.Close()

	if node.rpcL != nil {
		node.logger.Info("Stopping RPC interface")
		err = node.rpcL.Close()
	}

	node.logger.Info("Stopping router")
	err = node.r.Close()

	return err
}

// Apps returns list of AppStates for all registered apps.
func (node *Node) Apps() []router.AppState {
	var res []router.AppState
	node.am.RangeApps(func(host *router.AppHost) (next bool) {
		res = append(res, host.State())
		return true
	})
	return res
}

// StartApp starts registered App.
func (node *Node) StartApp(appName string) error {
	host, ok := node.am.AppOfName(appName)
	if !ok {
		return router.ErrAppNotFound
	}
	return host.Start()
}

// StopApp stops a registered App.
func (node *Node) StopApp(appName string) error {
	host, ok := node.am.AppOfName(appName)
	if !ok {
		return router.ErrAppNotFound
	}
	return host.Stop()
}

// SetAutoStart sets an app to auto start or not.
func (node *Node) SetAutoStart(appName string, autoStart bool) error {
	for i, ac := range node.c.Apps {
		if ac.App == appName {
			node.c.Apps[i].AutoStart = autoStart
			return nil
		}
	}
	return ErrUnknownApp
}
