package router

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/internal/appnet"
	"github.com/skycoin/skywire/internal/noise"
	"github.com/skycoin/skywire/pkg/app"
	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/util/pathutil"
)

// Errors associated with App Management.
var (
	ErrAppNotFound       = errors.New("app not found")
	ErrAppAlreadyExists  = errors.New("app already exists")
	ErrLoopNotFound      = errors.New("loop not found")
	ErrLoopAlreadyExists = errors.New("loop already exists")
	ErrDecryptionFailed  = errors.New("failed to decrypt packet payload")
	ErrNoFreePorts       = errors.New("ran out of free ports")
	ErrPortAlreadyTaken  = errors.New("port already taken")
)

// loopDispatch contains the necessities to dispatch packets for a given route.
// It also contains encryption data.
type loopDispatch struct {
	trID uuid.UUID
	rtID routing.RouteID
	ns   *noise.Noise
}

// IsConfirmed determines whether the loop is fully established.
// To confirm a loop, 2 sets of frames are sent back and forth.
func (ld *loopDispatch) IsConfirmed() bool { return ld.rtID != 0 }

// AppConfig is the app's config.
type AppConfig struct {
	Args      []string `json:"args"`
	AutoStart bool     `json:"auto_start"`
	Port      uint16   `json:"port"`
}

// AppState is an app's state.
type AppState struct {
	Running bool `json:"running"`
	Loops   int  `json:"loops"`
}

// AppInfo provides complete info on a given app.
type AppInfo struct {
	app.Meta
	Config AppConfig `json:"config"`
	State  AppState  `json:"state"`
}

// HostedApp hosts an App and keeps track or whether the App is running or not.
// It also manages incoming/outgoing packets to/from the App when the app is running.
type HostedApp struct {
	ah      *app.Host
	running bool
	loops   map[app.LoopMeta]*loopDispatch
	mx      sync.RWMutex
	log     *logrus.Entry

	// back-references
	config  *Config
	manager AppsManager
	router  Router
}

// NewHostedApp creates a new HostedApp
func NewHostedApp(l *logging.Logger, c *Config, m AppsManager, r Router, ah *app.Host) (*HostedApp, error) {
	ha := &HostedApp{
		ah:      ah,
		running: false,
		loops:   make(map[app.LoopMeta]*loopDispatch),
		log:     l.WithField("_app", fmt.Sprintf("%s(%s)", ah.AppName, ah.AppVersion)),
		config:  c,
		manager: m,
		router:  r,
	}
	return ha, nil
}

// Meta returns the meta data of the hosted App.
func (ar *HostedApp) Meta() app.Meta {
	return ar.ah.Meta
}

// State returns the state of the hosted App.
func (ar *HostedApp) State() AppState {
	ar.mx.RLock()
	state := AppState{
		Running: ar.running,
		Loops:   len(ar.loops),
	}
	ar.mx.RUnlock()
	return state
}

// IsRunning returns whether the hosted App is running.
func (ar *HostedApp) IsRunning() bool {
	ar.mx.RLock()
	running := ar.running
	ar.mx.RUnlock()
	return running
}

// Start runs the hosted App and begins handling incoming/outgoing packets.
// An error is returned if app fails to start or app is already running.
func (ar *HostedApp) Start() error {
	ar.mx.Lock()
	defer ar.mx.Unlock()

	if ar.running {
		return app.ErrAlreadyStarted
	}

	ar.log.Info("starting...")

	done, err := ar.ah.Start(ar.makeHandler(), ar.makeUIHandler())
	if err != nil {
		ar.log.Errorf("failed to start: %s", err)
		return err
	}

	go func(a *HostedApp, done <-chan struct{}) {
		<-done
		if err := a.Stop(); err != nil && err != app.ErrAlreadyStopped {
			ar.log.Warnf("stopped with error: %s", err)
		}
		ar.log.Info("stopped cleanly")
	}(ar, done)

	ar.running = true
	ar.log.Info("started")
	return nil
}

// Stop sends SIGTERM to the hosted App, closes all loops and ends handling of packets.
// An error is returned if app does not end cleanly, or if app already stopped.
func (ar *HostedApp) Stop() error {
	ar.mx.Lock()
	defer ar.mx.Unlock()

	if !ar.running {
		return app.ErrAlreadyStopped
	}

	ar.log.Info("stopping...")

	for lm := range ar.loops {
		_ = ar.router.CloseLoop(lm)                           //nolint:errcheck
		_, _ = ar.ah.Call(appnet.FrameCloseLoop, lm.Encode()) //nolint:errcheck
		delete(ar.loops, lm)
	}
	ar.running = false
	return ar.ah.Stop()
}

// ConfirmLoop attempts to confirm a loop with the hosted App.
func (ar *HostedApp) ConfirmLoop(lm app.LoopMeta, tpID uuid.UUID, rtID routing.RouteID, nsMsg []byte) ([]byte, error) {
	ar.mx.Lock()
	defer ar.mx.Unlock()

	if !ar.running {
		return nil, ErrAppNotFound
	}

	// if loop 'lm' is not found, sets a new loop with tpID and rtID.
	// else, retrieve loop 'lm'.
	// returns true if a new loop is set.
	setOrGetLoop := func(lm app.LoopMeta, tpID uuid.UUID, rtID routing.RouteID) (*loopDispatch, bool) {
		if ld, ok := ar.loops[lm]; ok {
			return ld, false
		}
		ld := &loopDispatch{trID: tpID, rtID: rtID}
		ar.loops[lm] = ld
		return ld, true
	}

	ld, isNew := setOrGetLoop(lm, tpID, rtID)
	if isNew {
		ns, err := noise.KKAndSecp256k1(noise.Config{
			LocalPK:   ar.config.PubKey,
			LocalSK:   ar.config.SecKey,
			RemotePK:  lm.Remote.PubKey,
			Initiator: false,
		})
		if err != nil {
			return nil, err
		}
		if err := ns.ProcessMessage(nsMsg); err != nil {
			return nil, err
		}
		nsResp, err := ns.HandshakeMessage()
		if err != nil {
			return nil, err
		}
		ld.ns, ld.trID, ld.rtID = ns, tpID, rtID
		if _, err := ar.ah.Call(appnet.FrameConfirmLoop, lm.Encode()); err != nil {
			ar.log.Warnf("Failed to notify App about new loop: %s", err)
		}
		return nsResp, nil
	}
	if ld.IsConfirmed() {
		return nil, ErrLoopAlreadyExists
	}
	if err := ld.ns.ProcessMessage(nsMsg); err != nil {
		return nil, err
	}
	ld.trID, ld.rtID = tpID, rtID
	if _, err := ar.ah.Call(appnet.FrameConfirmLoop, lm.Encode()); err != nil {
		ar.log.Warnf("Failed to notify App about new loop: %s", err)
	}
	return nil, nil
}

// ConfirmCloseLoop attempts to inform the hosted App that a loop is closed.
func (ar *HostedApp) ConfirmCloseLoop(lm app.LoopMeta) error {
	ar.mx.Lock()
	defer ar.mx.Unlock()

	if !ar.running {
		return ErrAppNotFound
	}
	delete(ar.loops, lm)

	if _, err := ar.ah.Call(appnet.FrameCloseLoop, lm.Encode()); err != nil {
		return err
	}
	ar.log.Infof("confirm close loop: %s", lm.String())
	return nil
}

// ConsumePacket attempts to send a DataFrame to the hosted App.
func (ar *HostedApp) ConsumePacket(lm app.LoopMeta, ciphertext []byte) error {
	ar.mx.RLock()
	defer ar.mx.RUnlock()

	if !ar.running {
		return ErrAppNotFound
	}

	ld, ok := ar.loops[lm]
	if !ok {
		return ErrLoopNotFound
	}
	plaintext, err := ld.ns.Decrypt(ciphertext)
	if err != nil {
		return fmt.Errorf("%s: %s", ErrDecryptionFailed.Error(), err.Error())
	}
	df := &app.DataFrame{Meta: lm, Data: plaintext}
	_, err = ar.ah.Call(appnet.FrameData, df.Encode())
	return err
}

// respondFunc is for allowing the separation of;
//  - operations that should be under the protection of sync.RWMutex
//  - operations that don't need to be under the protection of sync.RWMutex
// For example, if we define a function as `func action() respondFunc`;
//  1. logic in 'action' can be protected by sync.RWMutex.
//  2. the returned 'respondFunc' can be made to not be under the same protection.
type respondFunc func() ([]byte, error)

// returns a 'respondFunc' that always returns the given error.
func failWith(err error) respondFunc {
	return func() ([]byte, error) { return nil, err }
}

func (ar *HostedApp) makeHandler() appnet.HandlerMap {

	// triggered when App sends 'CreateLoop' frame to Host
	requestLoop := func(rAddr app.LoopAddr) respondFunc {
		ar.mx.Lock()
		defer ar.mx.Unlock()

		// prepare noise
		ns, err := noise.KKAndSecp256k1(noise.Config{
			LocalPK:   ar.config.PubKey,
			LocalSK:   ar.config.SecKey,
			RemotePK:  rAddr.PubKey,
			Initiator: true,
		})
		if err != nil {
			return failWith(err)
		}
		msg, err := ns.HandshakeMessage()
		if err != nil {
			return failWith(err)
		}

		// allocate local listening port for the new loop
		lPort, err := ar.manager.AllocPort(ar)
		if err != nil {
			return failWith(err)
		}

		lm := app.LoopMeta{Local: app.LoopAddr{PubKey: ar.config.PubKey, Port: lPort}, Remote: rAddr}

		// keep track of the new loop (if not already exists)
		if _, ok := ar.loops[lm]; ok {
			return failWith(ErrLoopAlreadyExists)
		}
		ar.loops[lm] = &loopDispatch{ns: ns}

		// if loop is of loopback type (dst app is on localhost) send to local app, else send to router.
		if lm.IsLoopback() {
			return func() ([]byte, error) {
				a2, ok := ar.manager.AppOfPort(lm.Remote.Port)
				if !ok {
					return nil, ErrAppNotFound
				}
				_, err := a2.ah.Call(appnet.FrameConfirmLoop, lm.Swap().Encode())
				return lm.Encode(), err
			}
		}
		return func() ([]byte, error) {
			return lm.Encode(), ar.router.FindRoutesAndSetupLoop(lm, msg)
		}
	}

	// triggered when App sends 'CloseLoop' frame to Host
	closeLoop := func(lm app.LoopMeta) respondFunc {
		ar.mx.Lock()
		delete(ar.loops, lm)
		ar.mx.Unlock()

		if lm.IsLoopback() {
			return func() ([]byte, error) {
				a2, ok := ar.manager.AppOfPort(lm.Remote.Port)
				if !ok {
					return nil, ErrAppNotFound
				}
				_, err := a2.ah.Call(appnet.FrameCloseLoop, lm.Encode())
				return nil, err
			}
		}
		return func() ([]byte, error) {
			return nil, ar.router.CloseLoop(lm)
		}
	}

	// triggered when App sends 'Data' frame to Host
	fwdPacket := func(lm app.LoopMeta, plaintext []byte) respondFunc {
		if lm.IsLoopback() {
			return func() ([]byte, error) {
				rA, ok := ar.manager.AppOfPort(lm.Remote.Port)
				if !ok {
					return nil, ErrLoopNotFound
				}
				df := app.DataFrame{Meta: *lm.Swap(), Data: plaintext}
				_, err := rA.ah.Call(appnet.FrameData, df.Encode())
				return nil, err
			}
		}
		ar.mx.RLock()
		ld, ok := ar.loops[lm]
		ar.mx.RUnlock()
		if !ok {
			return failWith(ErrLoopNotFound)
		}
		return func() ([]byte, error) {
			return nil, ar.router.ForwardPacket(ld.trID, ld.rtID, ld.ns.Encrypt(plaintext))
		}
	}

	return appnet.HandlerMap{
		appnet.FrameCreateLoop: func(_ *appnet.Protocol, b []byte) ([]byte, error) {
			var rAddr app.LoopAddr
			if err := rAddr.Decode(b); err != nil {
				return nil, err
			}
			return requestLoop(rAddr)()
		},
		appnet.FrameCloseLoop: func(_ *appnet.Protocol, b []byte) ([]byte, error) {
			var lm app.LoopMeta
			if err := lm.Decode(b); err != nil {
				return nil, err
			}
			return closeLoop(lm)()
		},
		appnet.FrameData: func(_ *appnet.Protocol, b []byte) ([]byte, error) {
			var df app.DataFrame
			if err := df.Decode(b); err != nil {
				return nil, err
			}
			return fwdPacket(df.Meta, df.Data)()
		},
	}
}

func (ar *HostedApp) makeUIHandler() appnet.HandlerMap {
	// TODO(evanlinjin): implement.
	return appnet.HandlerMap{}
}

// AppsManager manages local Apps and the associated ports and loops.
type AppsManager interface {
	RegisterApp(appName string, config AppConfig) (*HostedApp, error)
	DeregisterApp(appName string) error
	AppOfName(name string) (*HostedApp, bool)
	AppOfPort(lPort uint16) (*HostedApp, bool)
	AppInfo(name string) (*AppInfo, bool)
	RangeApps(fn AppFunc)
	AllocPort(appHost *HostedApp) (port uint16, err error)
	Close() error
}

// NewAppsManager creates a new AppsManager.
func NewAppsManager(c *Config, r Router, minPort uint16, binDir, localDir string) AppsManager {
	return &appsManager{
		minPort:  minPort,
		binDir:   binDir,
		localDir: localDir,
		apps: make(map[string]struct {
			*HostedApp
			*AppConfig
		}),
		ports:  make(map[uint16]*HostedApp),
		log:    logging.MustGetLogger("apps_manager"),
		config: c,
		router: r,
	}
}

type appsManager struct {
	minPort  uint16
	binDir   string // root dir for app bins.
	localDir string // root dir for app local files.

	// k: app_name, v: hosted_app + app_config
	apps map[string]struct {
		*HostedApp
		*AppConfig
	}

	// k: local_port, v: hosted_app
	ports map[uint16]*HostedApp

	mx  sync.RWMutex
	log *logging.Logger

	// back-references
	config *Config
	router Router
}

func (am *appsManager) RegisterApp(name string, config AppConfig) (*HostedApp, error) {
	binLoc := filepath.Join(am.binDir, name)
	if _, err := os.Stat(binLoc); os.IsNotExist(err) {
		return nil, fmt.Errorf("app binLoc: %s", err)
	}
	wkDir, err := pathutil.EnsureDir(filepath.Join(am.localDir, name))
	if err != nil {
		return nil, fmt.Errorf("app wkDir: %s", err)
	}
	h, err := app.NewHost(am.config.PubKey, wkDir, binLoc, config.Args)
	if err != nil {
		return nil, err
	}

	am.mx.Lock()
	defer am.mx.Unlock()

	if _, ok := am.apps[name]; ok {
		return nil, ErrAppAlreadyExists
	}

	ha, err := NewHostedApp(am.log, am.config, am, am.router, h)
	if err != nil {
		return nil, err
	}

	if config.Port != 0 {
		am.log.Info("assigned to port %d", config.Port)
		am.ports[config.Port] = ha
	}
	if config.AutoStart {
		go func() { _ = ha.Start() }() //nolint:errcheck
	}

	am.apps[name] = struct {
		*HostedApp
		*AppConfig
	}{HostedApp: ha, AppConfig: &config}

	return ha, nil
}

func (am *appsManager) DeregisterApp(name string) error {
	am.mx.Lock()
	defer am.mx.Unlock()

	a, ok := am.apps[name]
	if !ok {
		return ErrAppNotFound
	}

	delete(am.apps, name)

	for port, a := range am.ports {
		if a.Meta().AppName == name {
			delete(am.ports, port)
			break
		}
	}

	if a.IsRunning() {
		_ = a.Stop() //nolint:errcheck
	}

	return nil
}

func (am *appsManager) AppOfName(name string) (*HostedApp, bool) {
	am.mx.RLock()
	defer am.mx.RUnlock()
	for _, a := range am.apps {
		if a.ah.AppName == name {
			return a.HostedApp, true
		}
	}
	return nil, false
}

func (am *appsManager) AppOfPort(lPort uint16) (*HostedApp, bool) {
	am.mx.RLock()
	mApp, ok := am.ports[lPort]
	am.mx.RUnlock()
	return mApp, ok
}

func (am *appsManager) AppInfo(name string) (*AppInfo, bool) {
	am.mx.RLock()
	defer am.mx.RUnlock()

	a, ok := am.apps[name]
	if !ok {
		return nil, false
	}
	return &AppInfo{
		Meta:   a.Meta(),
		State:  a.State(),
		Config: *a.AppConfig,
	}, true
}

// AppFunc is triggered when ranging all hosted Apps.
type AppFunc func(config *AppConfig, host *HostedApp) (next bool)

func (am *appsManager) RangeApps(fn AppFunc) {
	am.mx.RLock()
	defer am.mx.RUnlock()
	for _, a := range am.apps {
		if next := fn(a.AppConfig, a.HostedApp); !next {
			return
		}
	}
}

func (am *appsManager) AllocPort(appHost *HostedApp) (uint16, error) {
	am.mx.Lock()
	defer am.mx.Unlock()

	for port := am.minPort; port < math.MaxUint16; port++ {
		if _, ok := am.ports[port]; !ok {
			am.ports[port] = appHost
			return port, nil
		}
	}
	return 0, ErrNoFreePorts
}

func (am *appsManager) Close() error {
	am.mx.Lock()
	defer am.mx.Unlock()

	am.apps = make(map[string]struct {
		*HostedApp
		*AppConfig
	})
	am.ports = make(map[uint16]*HostedApp)

	for _, a := range am.apps {
		_ = a.ah.Stop() //nolint:errcheck
	}
	return nil
}
