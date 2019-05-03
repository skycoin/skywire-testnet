// Package node implements skywire node.
package node

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/rpc"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/skycoin/skywire/pkg/app"

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
	conf *Config
	mc   *messaging.Client
	tm   *transport.Manager
	rt   routing.Table
	r    router.Router
	pm   router.ProcManager

	rootBinDir   string
	rootLocalDir string
	apps         map[string]*app.Meta
	aMx          sync.RWMutex

	Logger *logging.MasterLogger
	logger *logging.Logger

	rpcL net.Listener
	rpcD []*noise.RPCClientDialer
}

// NewNode constructs new Node.
func NewNode(config *Config) (*Node, error) {
	node := &Node{conf: config}

	node.Logger = logging.NewMasterLogger()
	node.logger = node.Logger.PackageLogger("skywire")

	if lvl, err := logging.LevelFromString(config.LogLevel); err == nil {
		node.Logger.SetLevel(lvl)
	}

	pk := config.Node.PubKey
	sk := config.Node.SecKey

	/* SETUP: MESSAGING */

	mConfig, err := config.MessagingConfig()
	if err != nil {
		return nil, fmt.Errorf("invalid Messaging config: %s", err)
	}

	node.mc = messaging.NewClient(mConfig)
	node.mc.Logger = node.Logger.PackageLogger("messenger")

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
	node.tm, err = transport.NewManager(tmConfig, node.mc)
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
		PubKey:     pk,
		SecKey:     sk,
		SetupNodes: config.Routing.SetupNodes,
	}
	r := router.New(node.Logger.PackageLogger("router"), node.tm, node.rt, routeFinder.NewHTTP(config.Routing.RouteFinder), rConf)
	node.r = r

	/* SETUP: APPS */

	node.pm = router.NewProcManager(10)
	node.apps = make(map[string]*app.Meta)

	localDir, err := config.LocalDir()
	if err != nil {
		return nil, fmt.Errorf("app manager: %s", err)
	}
	node.rootLocalDir = localDir

	binDir, err := config.AppsDir()
	if err != nil {
		return nil, fmt.Errorf("app manager: %s", err)
	}
	node.rootBinDir = binDir

	node.logger.Info("reading apps ...")

	files, err := ioutil.ReadDir(node.rootBinDir)
	if err != nil {
		return nil, fmt.Errorf("cannot read root apps dir: %s", err)
	}

	node.aMx.Lock()
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		meta, err := app.ObtainMeta(pk, filepath.Join(node.rootBinDir, f.Name()))
		if err != nil {
			continue
		}
		node.apps[meta.AppName] = meta
	}
	node.aMx.Unlock()

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

	err := node.mc.ConnectToInitialServers(ctx, node.conf.Messaging.ServerCount)
	if err != nil {
		return fmt.Errorf("messaging: %s", err)
	}
	node.logger.Info("Connected to messaging servers")

	/* START: TRANSPORTS */

	//node.tm.ReconnectTransports(ctx)
	//node.tm.CreateDefaultTransports(ctx)

	/* START: AUTO-START APPS */

	for _, ac := range node.conf.AutoStartApps {
		node.aMx.RLock()
		m, ok := node.apps[ac.App]
		node.aMx.RUnlock()
		if !ok {
			node.logger.Warnf("failed to auto-start app '%s': %s", ac.App,
				errors.New("app not found"))
		}

		e, err := app.NewExecutor(nil, m, &app.ExecConfig{
			HostPK:  node.conf.Node.PubKey,
			HostSK:  node.conf.Node.SecKey,
			WorkDir: filepath.Join(node.rootLocalDir, ac.App),
			BinLoc:  filepath.Join(node.rootBinDir, ac.App),
			Args:    ac.Args,
		})
		if err != nil {
			node.logger.Fatal(err)
		}

		proc, err := node.pm.RunProc(node.r, ac.Port, e)
		if err != nil {
			node.logger.Warnf("failed to auto-start app '%s': %s", ac.App, err)
		}
		node.logger.Infof("started proc.%d: %s", proc.ProcID(), ac.App)
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
	if err := node.r.Serve(ctx, node.pm); err != nil {
		return fmt.Errorf("failed to start Node: %s", err)
	}

	return nil
}

// Close safely stops spawned Apps and messaging Node.
func (node *Node) Close() (err error) {
	if node.rpcL != nil {
		node.logger.Info("stopping rpc_listener ...")
		if e := node.rpcL.Close(); e != nil {
			err = e
			node.logger.Errorf("rpc_listener stopped with error: %s", err)
		}
		node.logger.Info("rpc_listener stopped successfully.")
	}

	node.logger.Info("stopping rpc_dialers ...")
	for i, dialer := range node.rpcD {
		if e := dialer.Close(); e != nil {
			err = e
			node.logger.Errorf("rpc_dialer %d stopped with error: %s", i, err)
		}
		node.logger.Infof("rpc_dialer %d stopped successfully.", i)
	}

	node.logger.Info("stopping apps_manager ...")
	if e := node.pm.Close(); e != nil {
		err = e
		node.logger.Errorf("apps_manager stopped with error: %s", err)
	}
	node.logger.Info("apps_manager stopped successfully.")

	node.logger.Info("stopping router ...")
	if e := node.r.Close(); e != nil {
		err = e
		node.logger.Errorf("router stopped with with error: %s", err)
	}
	node.logger.Info("router stopped successfully.")

	return err
}

// Apps returns list of AppStates for all registered apps.
func (node *Node) Apps() []*app.Meta {
	var res []*app.Meta
	node.aMx.RLock()
	for _, m := range node.apps {
		res = append(res, m)
	}
	sort.Slice(res, func(i, j int) bool { return res[i].AppName < res[j].AppName })
	node.aMx.RUnlock()
	return res
}

// StartProc starts a process.
func (node *Node) StartProc(appName string, args []string, port uint16) (router.ProcID, error) {
	node.aMx.RLock()
	m, ok := node.apps[appName]
	node.aMx.RUnlock()
	if !ok {
		return 0, fmt.Errorf("app of name '%s' not found", appName)
	}
	e, err := app.NewExecutor(nil, m, &app.ExecConfig{
		HostPK:  node.conf.Node.PubKey,
		HostSK:  node.conf.Node.SecKey,
		WorkDir: filepath.Join(node.rootLocalDir, appName),
		BinLoc:  filepath.Join(node.rootBinDir, appName),
		Args:    args,
	})
	if err != nil {
		node.logger.Fatal(err)
	}

	proc, err := node.pm.RunProc(node.r, port, e)
	if err != nil {
		return 0, err
	}
	return proc.ProcID(), err
}

// StopProc stops a process of pid.
func (node *Node) StopProc(pid router.ProcID) error {
	proc, ok := node.pm.Proc(pid)
	if !ok {
		return router.ErrProcNotFound
	}
	return proc.Stop()
}

// ListProcs list meta info about the processes managed by procManager
func (node *Node) ListProcs() []*router.ProcInfo {
	return node.pm.ListProcs()
}
