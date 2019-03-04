// Package node implements skywire node.
package node

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/rpc"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/internal/noise"
	"github.com/skycoin/skywire/pkg/app"
	"github.com/skycoin/skywire/pkg/messaging"
	routeFinder "github.com/skycoin/skywire/pkg/route-finder/client"
	"github.com/skycoin/skywire/pkg/router"
	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/transport"
)

// AppStatus defines running status of an App.
type AppStatus int

const (
	// AppStatusStopped represents status of a stopped App.
	AppStatusStopped AppStatus = iota

	// AppStatusRunning  represents status of a running App.
	AppStatusRunning
)

// ErrUnknownApp represents lookup error for App related calls.
var ErrUnknownApp = errors.New("unknown app")

const supportedProtocolVersion = "0.0.1"

var reservedPorts = map[uint16]string{0: "router", 1: "chat", 2: "therealssh", 3: "therealproxy"}

// AppState defines state parameters for a registered App.
type AppState struct {
	Name      string    `json:"name"`
	AutoStart bool      `json:"autostart"`
	Port      uint16    `json:"port"`
	Status    AppStatus `json:"status"`
}

type appExecuter interface {
	Start(cmd *exec.Cmd) (int, error)
	Stop(pid int) error
	Wait(cmd *exec.Cmd) error
}

type appBind struct {
	conn net.Conn
	pid  int
}

// PacketRouter performs routing of the skywire packets.
type PacketRouter interface {
	io.Closer
	Serve(ctx context.Context) error
	ServeApp(conn net.Conn, port uint16, appConf *app.Config) error
}

// Node provides messaging runtime for Apps by setting up all
// necessary connections and performing messaging gateway functions.
type Node struct {
	config    *Config
	router    PacketRouter
	messenger *messaging.Client
	tm        *transport.Manager
	rt        routing.Table
	executer  appExecuter

	Logger *logging.MasterLogger
	logger *logging.Logger

	appsPath  string
	localPath string
	appsConf  []AppConfig

	startedMu   sync.RWMutex
	startedApps map[string]*appBind

	rpcListener net.Listener
	rpcDialers  []*noise.RPCClientDialer
}

// NewNode constructs new Node.
func NewNode(config *Config) (*Node, error) {
	node := &Node{
		config:      config,
		executer:    newOSExecuter(),
		startedApps: make(map[string]*appBind),
	}

	node.Logger = logging.NewMasterLogger()
	node.logger = node.Logger.PackageLogger("skywire")

	pk := config.Node.StaticPubKey
	sk := config.Node.StaticSecKey
	mDiscovery, err := config.MessagingDiscovery()
	if err != nil {
		return nil, fmt.Errorf("invalid MessagingConfig: %s", err)
	}

	node.messenger = messaging.NewClient(pk, sk, mDiscovery)
	node.messenger.Logger = node.Logger.PackageLogger("messenger")

	trDiscovery, err := config.TransportDiscovery()
	if err != nil {
		return nil, fmt.Errorf("invalid MessagingConfig: %s", err)
	}
	tmConfig := &transport.ManagerConfig{
		PubKey: pk, SecKey: sk,
		DiscoveryClient: trDiscovery,
		LogStore:        config.TransportLogStore(),
		DefaultNodes:    config.TrustedNodes,
	}
	node.tm, err = transport.NewManager(tmConfig, node.messenger)
	if err != nil {
		return nil, fmt.Errorf("transport manager: %s", err)
	}
	node.tm.Logger = node.Logger.PackageLogger("trmanager")

	node.rt, err = config.RoutingTable()
	if err != nil {
		return nil, fmt.Errorf("routing table: %s", err)
	}
	rConfig := &router.Config{
		Logger:           node.Logger.PackageLogger("router"),
		PubKey:           pk,
		SecKey:           sk,
		TransportManager: node.tm,
		RoutingTable:     node.rt,
		RouteFinder:      routeFinder.NewHTTP(config.Routing.RouteFinder),
		SetupNodes:       config.Routing.SetupNodes,
	}
	r := router.New(rConfig)
	node.router = r

	node.appsConf, err = config.AppsConfig()
	if err != nil {
		return nil, fmt.Errorf("invalid AppsConfig: %s", err)
	}

	node.appsPath, err = config.AppsDir()
	if err != nil {
		return nil, fmt.Errorf("invalid AppsPath: %s", err)
	}

	node.localPath, err = config.LocalDir()
	if err != nil {
		return nil, fmt.Errorf("invalid LocalPath: %s", err)
	}

	if lvl, err := logging.LevelFromString(config.LogLevel); err == nil {
		node.Logger.SetLevel(lvl)
	}

	if config.Interfaces.RPCAddress != "" {
		l, err := net.Listen("tcp", config.Interfaces.RPCAddress)
		if err != nil {
			return nil, fmt.Errorf("failed to setup RPC listener: %s", err)
		}
		node.rpcListener = l
	}
	node.rpcDialers = make([]*noise.RPCClientDialer, len(config.ManagerNodes))
	for i, entry := range config.ManagerNodes {
		node.rpcDialers[i] = noise.NewRPCClientDialer(entry.Addr, noise.HandshakeXK, noise.Config{
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
	err := node.messenger.ConnectToInitialServers(ctx, node.config.Messaging.ServerCount)
	if err != nil {
		return fmt.Errorf("messaging: %s", err)
	}
	node.logger.Info("Connected to messaging servers")

	node.tm.ReconnectTransports(ctx)
	node.tm.CreateDefaultTransports(ctx)

	for _, ac := range node.appsConf {
		if !ac.AutoStart {
			continue
		}

		go func(a AppConfig) {
			if err := node.SpawnApp(&a, nil); err != nil {
				node.logger.Warnf("Failed to start %s: %s\n", a.App, err)
			}
		}(ac)
	}

	rpcSvr := rpc.NewServer()
	if err := rpcSvr.RegisterName(RPCPrefix, &RPC{node: node}); err != nil {
		return fmt.Errorf("rpc server created failed: %s", err)
	}
	if node.rpcListener != nil {
		node.logger.Info("Starting RPC interface on ", node.rpcListener.Addr())
		go rpcSvr.Accept(node.rpcListener)
	}
	for _, dialer := range node.rpcDialers {
		go func(dialer *noise.RPCClientDialer) {
			if err := dialer.Run(rpcSvr, time.Second); err != nil {
				node.logger.Errorf("Dialer exited with error: %v", err)
			}
		}(dialer)
	}

	node.logger.Info("Starting packet router")
	if err := node.router.Serve(ctx); err != nil {
		return fmt.Errorf("failed to start Node: %s", err)
	}

	return nil
}

// Close safely stops spawned Apps and messaging Node.
func (node *Node) Close() (err error) {
	if node.rpcListener != nil {
		node.logger.Info("Stopping RPC interface")
		if rpcErr := node.rpcListener.Close(); rpcErr != nil && err == nil {
			err = rpcErr
		}
	}
	for _, dialer := range node.rpcDialers {
		err = dialer.Close()
	}

	node.startedMu.Lock()
	for app, bind := range node.startedApps {
		if appErr := node.stopApp(app, bind); appErr != nil && err == nil {
			err = appErr
		}
	}
	node.startedMu.Unlock()

	if node.rpcListener != nil {
		node.logger.Info("Stopping RPC interface")
		if rpcErr := node.rpcListener.Close(); rpcErr != nil && err == nil {
			err = rpcErr
		}
	}

	node.logger.Info("Stopping router")
	if msgErr := node.router.Close(); msgErr != nil && err == nil {
		err = msgErr
	}

	return err
}

// Apps returns list of AppStates for all registered apps.
func (node *Node) Apps() []*AppState {
	res := []*AppState{}
	for _, app := range node.appsConf {
		state := &AppState{app.App, app.AutoStart, app.Port, AppStatusStopped}
		node.startedMu.RLock()
		if node.startedApps[app.App] != nil {
			state.Status = AppStatusRunning
		}
		node.startedMu.RUnlock()

		res = append(res, state)
	}

	return res
}

// StartApp starts registered App.
func (node *Node) StartApp(appName string) error {
	for _, app := range node.appsConf {
		if app.App == appName {
			startCh := make(chan struct{})
			go func() {
				if err := node.SpawnApp(&app, startCh); err != nil {
					node.logger.Warnf("Failed to start app %s: %s", appName, err)
				}
			}()

			<-startCh
			return nil
		}
	}

	return ErrUnknownApp
}

// SpawnApp configures and starts new App.
func (node *Node) SpawnApp(config *AppConfig, startCh chan<- struct{}) error {
	node.logger.Infof("Starting %s.v%s", config.App, config.Version)
	conn, cmd, err := app.Command(
		&app.Config{ProtocolVersion: supportedProtocolVersion, AppName: config.App, AppVersion: config.Version},
		node.appsPath,
		config.Args,
	)
	if err != nil {
		return fmt.Errorf("failed to initialise App server: %s", err)
	}

	bind := &appBind{conn, -1}
	if app, ok := reservedPorts[config.Port]; ok && app != config.App {
		return fmt.Errorf("can't bind to reserved port %d", config.Port)
	}

	node.startedMu.Lock()
	if node.startedApps[config.App] != nil {
		node.startedMu.Unlock()
		return fmt.Errorf("App %s is already started", config.App)
	}

	node.startedApps[config.App] = bind
	node.startedMu.Unlock()

	// TODO: make PackageLogger return *Entry. FieldLogger doesn't expose Writer.
	logger := node.logger.WithField("_module", fmt.Sprintf("%s.v%s", config.App, config.Version)).Writer()
	defer logger.Close()

	cmd.Stdout = logger
	cmd.Stderr = logger
	cmd.Dir = filepath.Join(node.localPath, config.App, fmt.Sprintf("v%s", config.Version))
	if _, err := ensureDir(cmd.Dir); err != nil {
		return err
	}

	appCh := make(chan error)
	go func() {
		pid, err := node.executer.Start(cmd)
		if err != nil {
			appCh <- err
			return
		}

		node.startedMu.Lock()
		bind.pid = pid
		node.startedMu.Unlock()
		appCh <- node.executer.Wait(cmd)
	}()

	srvCh := make(chan error)
	go func() {
		srvCh <- node.router.ServeApp(conn, config.Port, &app.Config{AppName: config.App, AppVersion: config.Version})
	}()

	if startCh != nil {
		startCh <- struct{}{}
	}

	var appErr error
	select {
	case err := <-appCh:
		if err != nil {
			if _, ok := err.(*exec.ExitError); !ok {
				appErr = fmt.Errorf("failed to run app executable: %s", err)
			}
		}
	case err := <-srvCh:
		if err != nil {
			appErr = fmt.Errorf("failed to start communication server: %s", err)
		}
	}

	node.startedMu.Lock()
	delete(node.startedApps, config.App)
	node.startedMu.Unlock()

	return appErr
}

// StopApp stops running App.
func (node *Node) StopApp(appName string) error {
	node.startedMu.Lock()
	bind := node.startedApps[appName]
	node.startedMu.Unlock()

	if bind == nil {
		return ErrUnknownApp
	}

	return node.stopApp(appName, bind)
}

// SetAutoStart sets an app to auto start or not.
func (node *Node) SetAutoStart(appName string, autoStart bool) error {
	for i, ac := range node.appsConf {
		if ac.App == appName {
			node.appsConf[i].AutoStart = autoStart
			return nil
		}
	}
	return ErrUnknownApp
}

func (node *Node) stopApp(app string, bind *appBind) (err error) {
	node.logger.Infof("Stopping app %s and closing ports", app)

	if excErr := node.executer.Stop(bind.pid); excErr != nil && err == nil {
		node.logger.Warn("Failed to stop app: ", excErr)
		err = excErr
	}

	if srvErr := bind.conn.Close(); srvErr != nil && err == nil {
		node.logger.Warnf("Failed to close App conn: %s", srvErr)
		err = srvErr
	}

	return err
}

type osExecuter struct {
	processes []*os.Process
	mu        sync.Mutex
}

func newOSExecuter() *osExecuter {
	return &osExecuter{processes: make([]*os.Process, 0)}
}

func (exc *osExecuter) Start(cmd *exec.Cmd) (int, error) {
	if err := cmd.Start(); err != nil {
		return -1, err
	}
	exc.mu.Lock()
	exc.processes = append(exc.processes, cmd.Process)
	exc.mu.Unlock()
	return cmd.Process.Pid, nil
}

func (exc *osExecuter) Stop(pid int) (err error) {
	exc.mu.Lock()
	defer exc.mu.Unlock()

	for _, process := range exc.processes {
		if process.Pid != pid {
			continue
		}

		if sigErr := process.Signal(syscall.SIGTERM); sigErr != nil && err == nil {
			err = sigErr
		}
	}

	return err
}

func (exc *osExecuter) Wait(cmd *exec.Cmd) error {
	return cmd.Wait()
}
