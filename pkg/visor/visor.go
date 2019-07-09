// Package visor implements skywire visor.
package visor

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/rpc"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/skycoin/dmsg/noise"
	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/app"
	routeFinder "github.com/skycoin/skywire/pkg/route-finder/client"
	"github.com/skycoin/skywire/pkg/router"
	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/transport"
	"github.com/skycoin/skywire/pkg/transport/dmsg"
	"github.com/skycoin/skywire/pkg/util/pathutil"
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

// Version is the visor version.
const Version = "0.0.1"

const supportedProtocolVersion = "0.0.1"

var reservedPorts = map[uint16]string{0: "router", 1: "skychat", 2: "SSH", 3: "socksproxy"}

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
	IsSetupTransport(tr *transport.ManagedTransport) bool
}

// Visor provides messaging runtime for Apps by setting up all
// necessary connections and performing messaging gateway functions.
type Visor struct {
	config    *Config
	router    PacketRouter
	messenger *dmsg.Client
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

	pidMu sync.Mutex

	rpcListener net.Listener
	rpcDialers  []*noise.RPCClientDialer
}

// New constructs new Visor.
func New(config *Config, masterLogger *logging.MasterLogger) (*Visor, error) {
	visor := &Visor{
		config:      config,
		executer:    newOSExecuter(),
		startedApps: make(map[string]*appBind),
	}

	visor.Logger = masterLogger
	visor.logger = visor.Logger.PackageLogger("skywire")

	pk := config.Visor.StaticPubKey
	sk := config.Visor.StaticSecKey
	mConfig, err := config.MessagingConfig()
	if err != nil {
		return nil, fmt.Errorf("invalid Messaging config: %s", err)
	}

	visor.messenger = dmsg.NewClient(mConfig.PubKey, mConfig.SecKey, mConfig.Discovery, dmsg.SetLogger(visor.Logger.PackageLogger(dmsg.Type)))

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
		DefaultVisors:   config.TrustedVisors,
	}
	visor.tm, err = transport.NewManager(tmConfig, visor.messenger)
	if err != nil {
		return nil, fmt.Errorf("transport manager: %s", err)
	}
	visor.tm.Logger = visor.Logger.PackageLogger("trmanager")

	visor.rt, err = config.RoutingTable()
	if err != nil {
		return nil, fmt.Errorf("routing table: %s", err)
	}
	rConfig := &router.Config{
		Logger:           visor.Logger.PackageLogger("router"),
		PubKey:           pk,
		SecKey:           sk,
		TransportManager: visor.tm,
		RoutingTable:     visor.rt,
		RouteFinder:      routeFinder.NewHTTP(config.Routing.RouteFinder, time.Duration(config.Routing.RouteFinderTimeout)),
		SetupNodes:       config.Routing.SetupNodes,
	}
	r := router.New(rConfig)
	visor.router = r

	visor.appsConf, err = config.AppsConfig()
	if err != nil {
		return nil, fmt.Errorf("invalid AppsConfig: %s", err)
	}

	visor.appsPath, err = config.AppsDir()
	if err != nil {
		return nil, fmt.Errorf("invalid AppsPath: %s", err)
	}

	visor.localPath, err = config.LocalDir()
	if err != nil {
		return nil, fmt.Errorf("invalid LocalPath: %s", err)
	}

	if lvl, err := logging.LevelFromString(config.LogLevel); err == nil {
		visor.Logger.SetLevel(lvl)
	}

	if config.Interfaces.RPCAddress != "" {
		l, err := net.Listen("tcp", config.Interfaces.RPCAddress)
		if err != nil {
			return nil, fmt.Errorf("failed to setup RPC listener: %s", err)
		}
		visor.rpcListener = l
	}
	visor.rpcDialers = make([]*noise.RPCClientDialer, len(config.Hypervisors))
	for i, entry := range config.Hypervisors {
		visor.rpcDialers[i] = noise.NewRPCClientDialer(entry.Addr, noise.HandshakeXK, noise.Config{
			LocalPK:   pk,
			LocalSK:   sk,
			RemotePK:  entry.PubKey,
			Initiator: true,
		})
	}

	return visor, err
}

// Start spawns auto-started Apps, starts router and RPC interfaces .
func (visor *Visor) Start() error {
	ctx := context.Background()
	err := visor.messenger.InitiateServerConnections(ctx, visor.config.Messaging.ServerCount)
	if err != nil {
		return fmt.Errorf("%s: %s", dmsg.Type, err)
	}
	visor.logger.Info("Connected to messaging servers")

	pathutil.EnsureDir(visor.dir())
	visor.closePreviousApps()
	for _, ac := range visor.appsConf {
		if !ac.AutoStart {
			continue
		}

		go func(a AppConfig) {
			if err := visor.SpawnApp(&a, nil); err != nil {
				visor.logger.Warnf("Failed to start %s: %s\n", a.App, err)
			}
		}(ac)
	}

	rpcSvr := rpc.NewServer()
	if err := rpcSvr.RegisterName(RPCPrefix, &RPC{visor: visor}); err != nil {
		return fmt.Errorf("rpc server created failed: %s", err)
	}
	if visor.rpcListener != nil {
		visor.logger.Info("Starting RPC interface on ", visor.rpcListener.Addr())
		go rpcSvr.Accept(visor.rpcListener)
	}
	for _, dialer := range visor.rpcDialers {
		go func(dialer *noise.RPCClientDialer) {
			if err := dialer.Run(rpcSvr, time.Second); err != nil {
				visor.logger.Errorf("Dialer exited with error: %v", err)
			}
		}(dialer)
	}

	visor.logger.Info("Starting packet router")
	if err := visor.router.Serve(ctx); err != nil {
		return fmt.Errorf("failed to start Visor: %s", err)
	}

	return nil
}

func (visor *Visor) dir() string {
	return pathutil.VisorDir(visor.config.Visor.StaticPubKey)
}

func (visor *Visor) pidFile() *os.File {
	f, err := os.OpenFile(filepath.Join(visor.dir(), "apps-pid.txt"), os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}

	return f
}

func (visor *Visor) closePreviousApps() {
	visor.logger.Info("killing previously ran apps if any...")

	pids := visor.pidFile()
	defer pids.Close() // nocheck: err

	scanner := bufio.NewScanner(pids)
	for scanner.Scan() {
		appInfo := strings.Split(scanner.Text(), " ")
		if len(appInfo) != 2 {
			visor.logger.Fatal("error parsing %s. Err: %s", pids.Name(), errors.New("line should be: [app name] [pid]"))
		}

		pid, err := strconv.Atoi(appInfo[1])
		if err != nil {
			visor.logger.Fatal("error parsing %s. Err: %s", pids.Name(), err)
		}

		visor.stopUnhandledApp(appInfo[0], pid)
	}

	// empty file
	pathutil.AtomicWriteFile(pids.Name(), []byte{})
}

func (visor *Visor) stopUnhandledApp(name string, pid int) {
	p, err := os.FindProcess(pid)
	if err != nil {
		if runtime.GOOS != "windows" {
			visor.logger.Infof("Previous app %s ran by this visor with pid: %d not found", name, pid)
		}
		return
	}

	err = p.Signal(syscall.SIGKILL)
	if err != nil {
		return
	}

	visor.logger.Infof("Found and killed hanged app %s with pid %d previously ran by this visor", name, pid)
}

// Close safely stops spawned Apps and messaging Visor.
func (visor *Visor) Close() (err error) {
	if visor == nil {
		return nil
	}
	if visor.rpcListener != nil {
		if err = visor.rpcListener.Close(); err != nil {
			visor.logger.WithError(err).Error("failed to stop RPC interface")
		} else {
			visor.logger.Info("RPC interface stopped successfully")
		}
	}
	for i, dialer := range visor.rpcDialers {
		if err = dialer.Close(); err != nil {
			visor.logger.WithError(err).Errorf("(%d) failed to stop RPC dialer", i)
		} else {
			visor.logger.Infof("(%d) RPC dialer closed successfully", i)
		}
	}
	visor.startedMu.Lock()
	for a, bind := range visor.startedApps {
		if err = visor.stopApp(a, bind); err != nil {
			visor.logger.WithError(err).Errorf("(%s) failed to stop app", a)
		} else {
			visor.logger.Infof("(%s) app stopped successfully", a)
		}
	}
	visor.startedMu.Unlock()
	if err = visor.router.Close(); err != nil {
		visor.logger.WithError(err).Error("failed to stop router")
	} else {
		visor.logger.Info("router stopped successfully")
	}
	return err
}

// Apps returns list of AppStates for all registered apps.
func (visor *Visor) Apps() []*AppState {
	res := []*AppState{}
	for _, app := range visor.appsConf {
		state := &AppState{app.App, app.AutoStart, app.Port, AppStatusStopped}
		visor.startedMu.RLock()
		if visor.startedApps[app.App] != nil {
			state.Status = AppStatusRunning
		}
		visor.startedMu.RUnlock()

		res = append(res, state)
	}

	return res
}

// StartApp starts registered App.
func (visor *Visor) StartApp(appName string) error {
	for _, app := range visor.appsConf {
		if app.App == appName {
			startCh := make(chan struct{})
			go func() {
				if err := visor.SpawnApp(&app, startCh); err != nil {
					visor.logger.Warnf("Failed to start app %s: %s", appName, err)
				}
			}()

			<-startCh
			return nil
		}
	}

	return ErrUnknownApp
}

// SpawnApp configures and starts new App.
func (visor *Visor) SpawnApp(config *AppConfig, startCh chan<- struct{}) error {
	visor.logger.Infof("Starting %s.v%s", config.App, config.Version)
	conn, cmd, err := app.Command(
		&app.Config{ProtocolVersion: supportedProtocolVersion, AppName: config.App, AppVersion: config.Version},
		visor.appsPath,
		config.Args,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize App server: %s", err)
	}

	bind := &appBind{conn, -1}
	if app, ok := reservedPorts[config.Port]; ok && app != config.App {
		return fmt.Errorf("can't bind to reserved port %d", config.Port)
	}

	visor.startedMu.Lock()
	if visor.startedApps[config.App] != nil {
		visor.startedMu.Unlock()
		return fmt.Errorf("app %s is already started", config.App)
	}

	visor.startedApps[config.App] = bind
	visor.startedMu.Unlock()

	// TODO: make PackageLogger return *Entry. FieldLogger doesn't expose Writer.
	logger := visor.logger.WithField("_module", fmt.Sprintf("%s.v%s", config.App, config.Version)).Writer()
	defer logger.Close()

	cmd.Stdout = logger
	cmd.Stderr = logger
	cmd.Dir = filepath.Join(visor.localPath, config.App, fmt.Sprintf("v%s", config.Version))
	if _, err := ensureDir(cmd.Dir); err != nil {
		return err
	}

	appCh := make(chan error)
	go func() {
		pid, err := visor.executer.Start(cmd)
		if err != nil {
			appCh <- err
			return
		}

		visor.startedMu.Lock()
		bind.pid = pid
		visor.startedMu.Unlock()

		visor.pidMu.Lock()
		visor.logger.Infof("storing app %s pid %d", config.App, pid)
		visor.persistPID(config.App, pid)
		visor.pidMu.Unlock()
		appCh <- visor.executer.Wait(cmd)
	}()

	srvCh := make(chan error)
	go func() {
		srvCh <- visor.router.ServeApp(conn, config.Port, &app.Config{AppName: config.App, AppVersion: config.Version})
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

	visor.startedMu.Lock()
	delete(visor.startedApps, config.App)
	visor.startedMu.Unlock()

	return appErr
}

func (visor *Visor) persistPID(name string, pid int) {
	pidF := visor.pidFile()
	pidFName := pidF.Name()
	pidF.Close()

	pathutil.AtomicAppendToFile(pidFName, []byte(fmt.Sprintf("%s %d\n", name, pid)))
}

// StopApp stops running App.
func (visor *Visor) StopApp(appName string) error {
	visor.startedMu.Lock()
	bind := visor.startedApps[appName]
	visor.startedMu.Unlock()

	if bind == nil {
		return ErrUnknownApp
	}

	return visor.stopApp(appName, bind)
}

// SetAutoStart sets an app to auto start or not.
func (visor *Visor) SetAutoStart(appName string, autoStart bool) error {
	for i, ac := range visor.appsConf {
		if ac.App == appName {
			visor.appsConf[i].AutoStart = autoStart
			return nil
		}
	}
	return ErrUnknownApp
}

func (visor *Visor) stopApp(app string, bind *appBind) (err error) {
	visor.logger.Infof("Stopping app %s and closing ports", app)

	if excErr := visor.executer.Stop(bind.pid); excErr != nil && err == nil {
		visor.logger.Warn("Failed to stop app: ", excErr)
		err = excErr
	}

	if srvErr := bind.conn.Close(); srvErr != nil && err == nil {
		visor.logger.Warnf("Failed to close App conn: %s", srvErr)
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

		if sigErr := process.Signal(syscall.SIGKILL); sigErr != nil && err == nil {
			err = sigErr
		}
	}

	return err
}

func (exc *osExecuter) Wait(cmd *exec.Cmd) error {
	return cmd.Wait()
}
