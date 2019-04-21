package router

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/skycoin/skycoin/src/util/logging"
	"github.com/skycoin/skywire/internal/appnet"
	"github.com/skycoin/skywire/internal/noise"
	"github.com/skycoin/skywire/pkg/app"
	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/util/pathutil"
	"math"
	"os"
	"path/filepath"
	"sync"
)

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

func (ld *loopDispatch) IsConfirmed() bool { return ld.rtID != 0 }

type AppState struct {
	app.Meta
	Running bool `json:"running"`
	Loops   int  `json:"loops"`
}

type AppHost struct {
	host    *app.Host
	running bool
	loops   map[app.LoopMeta]*loopDispatch
	mx      sync.RWMutex
	log     *logrus.Entry

	// back-references
	config  *Config
	manager AppsManager
	router  Router
}

func NewAppHost(l *logging.Logger, c *Config, m AppsManager, r Router, ah *app.Host) (*AppHost, error) {
	return &AppHost{
		host:    ah,
		running: false,
		loops:   make(map[app.LoopMeta]*loopDispatch),
		log:     l.WithField("_app", ah.AppName),
		config:  c,
		manager: m,
		router:  r,
	}, nil
}

func (a *AppHost) State() AppState {
	a.mx.RLock()
	state := AppState{
		Meta:    a.host.Meta,
		Running: a.running,
		Loops:   len(a.loops),
	}
	a.mx.RUnlock()
	return state
}

func (a *AppHost) IsRunning() bool {
	a.mx.RLock()
	running := a.running
	a.mx.RUnlock()
	return running
}

func (a *AppHost) Start() error {
	a.mx.Lock()
	defer a.mx.Unlock()
	if a.running {
		return app.ErrAlreadyStarted
	}
	done, err := a.host.Start(a.makeHandler(), a.makeUIHandler())
	if err != nil {
		return err
	}
	go func(a *AppHost, done <-chan struct{}) {
		<-done
		_ = a.Stop() // TODO(evanlinjin): log this!
	}(a, done)
	a.running = true
	return nil
}

func (a *AppHost) Stop() error {
	a.mx.Lock()
	defer a.mx.Unlock()

	if !a.running {
		return app.ErrAlreadyStopped
	}
	for lm := range a.loops {
		// TODO(evanlinjin): log the below.
		_ = a.router.CloseLoop(lm)
		_, _ = a.host.Call(appnet.FrameCloseLoop, lm.Encode())
		delete(a.loops, lm)
	}
	a.running = false
	return a.host.Stop()
}

func (a *AppHost) ConfirmLoop(lm app.LoopMeta, tpID uuid.UUID, rtID routing.RouteID, nsMsg []byte) ([]byte, error) {
	ld, isNew := a.setOrGetLoop(lm, tpID, rtID)
	if isNew {
		ns, err := noise.KKAndSecp256k1(noise.Config{
			LocalPK:   a.config.PubKey,
			LocalSK:   a.config.SecKey,
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
		if _, err := a.host.Call(appnet.FrameConfirmLoop, lm.Encode()); err != nil {
			a.log.Warnf("Failed to notify App about new loop: %s", err)
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
	if _, err := a.host.Call(appnet.FrameConfirmLoop, lm.Encode()); err != nil {
		a.log.Warnf("Failed to notify App about new loop: %s", err)
	}
	return nil, nil
}

func (a *AppHost) ConfirmCloseLoop(lm app.LoopMeta) error {
	a.mx.Lock()
	delete(a.loops, lm)
	a.mx.Unlock()
	if _, err := a.host.Call(appnet.FrameCloseLoop, lm.Encode()); err != nil {
		return err
	}
	a.log.Infof("confirm close loop: %s", lm)
	return nil
}

func (a *AppHost) ConsumePacket(lm app.LoopMeta, ciphertext []byte) error {
	ld, err := a.getLoop(lm)
	if err != nil {
		return err
	}
	plaintext, err := ld.ns.Decrypt(ciphertext)
	if err != nil {
		return fmt.Errorf("%s: %s", ErrDecryptionFailed.Error(), err.Error())
	}
	df := &app.DataFrame{Meta: lm, Data: plaintext}
	_, err = a.host.Call(appnet.FrameData, df.Encode())
	return err
}

// obtains dispatch information of a loop.
func (a *AppHost) getLoop(lm app.LoopMeta) (ld *loopDispatch, err error) {
	a.mx.RLock()
	var ok bool
	if a.loops == nil {
		err = app.ErrAppClosed
	} else if ld, ok = a.loops[lm]; !ok {
		err = ErrLoopNotFound
	}
	a.mx.RUnlock()
	return ld, err
}

func (a *AppHost) newLoop(lm app.LoopMeta, ns *noise.Noise) (*loopDispatch, error) {
	if _, ok := a.loops[lm]; ok {
		return nil, ErrLoopAlreadyExists
	}
	ld := &loopDispatch{ns: ns}
	a.loops[lm] = ld
	return ld, nil
}

// returns true if value is successfully set.
func (a *AppHost) setOrGetLoop(lm app.LoopMeta, trID uuid.UUID, rtID routing.RouteID) (*loopDispatch, bool) {
	a.mx.Lock()
	defer a.mx.Unlock()

	if ld, ok := a.loops[lm]; ok {
		return ld, false
	}
	ld := &loopDispatch{trID: trID, rtID: rtID}
	a.loops[lm] = ld
	return ld, true
}

func (a *AppHost) makeHandler() appnet.HandlerMap {

	// triggered when App sends 'CreateLoop' frame to Host
	reqLoop := func(rAddr app.LoopAddr) (app.LoopMeta, error) {
		ns, err := noise.KKAndSecp256k1(noise.Config{
			LocalPK:   a.config.PubKey,
			LocalSK:   a.config.SecKey,
			RemotePK:  rAddr.PubKey,
			Initiator: true,
		})
		if err != nil {
			return app.LoopMeta{}, err
		}
		msg, err := ns.HandshakeMessage()
		if err != nil {
			return app.LoopMeta{}, err
		}
		lPort, err := a.manager.AllocPort(a)
		if err != nil {
			return app.LoopMeta{}, err
		}
		lm := app.LoopMeta{
			Local:  app.LoopAddr{PubKey: a.config.PubKey, Port: lPort},
			Remote: rAddr,
		}
		if _, err := a.newLoop(lm, ns); err != nil {
			return app.LoopMeta{}, err
		}
		if lm.IsLoopback() {
			rA, ok := a.manager.AppOfPort(lm.Remote.Port)
			if !ok {
				return app.LoopMeta{}, ErrAppNotFound
			}
			_, err := rA.host.Call(appnet.FrameConfirmLoop, lm.Swap().Encode())
			return app.LoopMeta{}, err
		}
		return lm, a.router.FindRoutesAndSetupLoop(lm, msg)
	}

	// triggered when App sends 'CloseLoop' frame to Host
	closeLoop := func(lm app.LoopMeta) error {
		a.mx.Lock()
		delete(a.loops, lm)
		a.mx.Unlock()
		return a.router.CloseLoop(lm)
	}

	// triggered when App sends 'Data' frame to Host
	fwdPacket := func(lm app.LoopMeta, plaintext []byte) error {
		if lm.IsLoopback() {
			rA, ok := a.manager.AppOfPort(lm.Remote.Port)
			if !ok {
				return ErrLoopNotFound
			}
			df := app.DataFrame{Meta: *lm.Swap(), Data: plaintext}
			_, err := rA.host.Call(appnet.FrameData, df.Encode())
			return err
		}
		ld, err := a.getLoop(lm)
		if err != nil {
			return err
		}
		return a.router.ForwardPacket(ld.trID, ld.rtID, ld.ns.Encrypt(plaintext))
	}

	return appnet.HandlerMap{
		appnet.FrameCreateLoop: func(_ *appnet.Protocol, b []byte) ([]byte, error) {
			var rAddr app.LoopAddr
			if err := rAddr.Decode(b); err != nil {
				return nil, err
			}
			lm, err := reqLoop(rAddr)
			if err != nil {
				return nil, err
			}
			return lm.Encode(), nil
		},
		appnet.FrameCloseLoop: func(_ *appnet.Protocol, b []byte) ([]byte, error) {
			var lm app.LoopMeta
			if err := lm.Decode(b); err != nil {
				return nil, err
			}
			return nil, closeLoop(lm)
		},
		appnet.FrameData: func(_ *appnet.Protocol, b []byte) ([]byte, error) {
			var df app.DataFrame
			if err := df.Decode(b); err != nil {
				return nil, err
			}
			return nil, fwdPacket(df.Meta, df.Data)
		},
	}
}

func (a *AppHost) makeUIHandler() appnet.HandlerMap {
	// TODO(evanlinjin): implement.
	return appnet.HandlerMap{}
}

/*
	<<< APPS_MANAGER >>>
*/

// AppsManager manages local Apps and the associated ports and loops.
type AppsManager interface {
	RegisterApp(appName string, args []string) (*AppHost, error)
	DeregisterApp(appHost *AppHost) error
	AppOfName(name string) (*AppHost, bool)
	AppOfPort(lPort uint16) (*AppHost, bool)
	RangeApps(fn AppFunc)
	AllocPort(appHost *AppHost) (port uint16, err error)
	AllocGivenPort(appHost *AppHost, port uint16) error
	Close() error
}

func NewAppsManager(c *Config, r Router, minPort uint16, binDir, localDir string) AppsManager {
	return &appsManager{
		ports:    make(map[uint16]*AppHost),
		log:      logging.MustGetLogger("apps_manager"),
		minPort:  minPort,
		binDir:   binDir,
		localDir: localDir,
		config:   c,
		router:   r,
	}
}

type appsManager struct {
	apps  []*AppHost
	ports map[uint16]*AppHost // key(local_port)
	mx    sync.RWMutex
	log   *logging.Logger

	minPort  uint16
	binDir   string // root dir for app bins.
	localDir string // root dir for app local files.

	// back-references
	config *Config
	router Router
}

func (am *appsManager) RegisterApp(appName string, args []string) (*AppHost, error) {
	am.mx.Lock()
	defer am.mx.Unlock()
	binLoc := filepath.Join(am.binDir, appName)
	if _, err := os.Stat(binLoc); os.IsNotExist(err) {
		return nil, fmt.Errorf("app binLoc: %s", err)
	}
	wkDir, err := pathutil.EnsureDir(filepath.Join(am.localDir, appName))
	if err != nil {
		return nil, fmt.Errorf("app wkDir: %s", err)
	}
	host, err := app.NewHost(am.config.PubKey, wkDir, binLoc, args)
	if err != nil {
		return nil, err
	}
	appHost, err := NewAppHost(am.log, am.config, am, am.router, host)
	if err != nil {
		return nil, err
	}
	for _, a := range am.apps {
		if a.host.AppName == appHost.host.AppName {
			return nil, ErrAppAlreadyExists
		}
	}
	am.apps = append(am.apps, appHost)
	return appHost, nil
}

func (am *appsManager) DeregisterApp(appHost *AppHost) error {
	am.mx.Lock()
	defer am.mx.Unlock()
	if appHost.IsRunning() {
		return errors.New("cannot deregister the app when it is running")
	}
	for i, a := range am.apps {
		if a == appHost {
			am.apps = append(am.apps[:i], am.apps[i+1:]...)
			break
		}
	}
	for port, a := range am.ports {
		if a == appHost {
			delete(am.ports, port)
			break
		}
	}
	return nil
}

func (am *appsManager) AppOfName(name string) (*AppHost, bool) {
	am.mx.RLock()
	defer am.mx.RUnlock()
	for _, a := range am.apps {
		if a.host.AppName == name {
			return a, true
		}
	}
	return nil, false
}

// obtains the app reserving the given port.
func (am *appsManager) AppOfPort(lPort uint16) (*AppHost, bool) {
	am.mx.RLock()
	mApp, ok := am.ports[lPort]
	am.mx.RUnlock()
	return mApp, ok
}

type AppFunc func(host *AppHost) (next bool)

func (am *appsManager) RangeApps(fn AppFunc) {
	am.mx.RLock()
	defer am.mx.RUnlock()
	for _, host := range am.apps {
		if next := fn(host); !next {
			return
		}
	}
}

func (am *appsManager) AllocPort(appHost *AppHost) (uint16, error) {
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

func (am *appsManager) AllocGivenPort(appHost *AppHost, port uint16) error {
	am.mx.Lock()
	defer am.mx.Unlock()
	if _, ok := am.ports[port]; ok {
		return ErrPortAlreadyTaken
	}
	am.ports[port] = appHost
	return nil
}

func (am *appsManager) Close() error {
	am.mx.Lock()
	defer am.mx.Unlock()

	for _, a := range am.apps {
		_ = a.host.Stop()
	}
	return nil
}
