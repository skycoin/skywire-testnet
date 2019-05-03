package router

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/google/uuid"
	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/internal/appnet"
	"github.com/skycoin/skywire/internal/noise"
	"github.com/skycoin/skywire/pkg/app"
	"github.com/skycoin/skywire/pkg/routing"
)

// Errors associated with App Management.
var (
	ErrProcNotFound      = errors.New("proc not found")
	ErrLoopNotFound      = errors.New("loop not found")
	ErrLoopAlreadyExists = errors.New("loop already exists")
	ErrDecryptionFailed  = errors.New("failed to decrypt packet payload")
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

// ProcID is the execution ID of a running App.
type ProcID uint16

// AppProc hosts an App and keeps track or whether the App is running or not.
// It also manages incoming/outgoing packets to/from the App when the app is running.
type AppProc struct {

	// stores a boolean; 'true' if no longer running.
	// unsafe.Pointer is used alongside 'atomic' module for fast, thread-safe access.
	stopped unsafe.Pointer

	pid  ProcID
	e    app.Executor
	lps  map[app.LoopMeta]*loopDispatch
	mx   sync.RWMutex
	log  *logging.Logger
	port uint16

	// back-references
	pm ProcManager
	r  Router
}

// NewAppProc creates a new AppProc
func NewAppProc(pm ProcManager, r Router, port uint16, pid ProcID, exec app.Executor) (*AppProc, error) {

	proc := &AppProc{
		stopped: unsafe.Pointer(new(bool)), //nolint:gosec
		pid:     pid,
		e:       exec,
		lps:     make(map[app.LoopMeta]*loopDispatch),
		log:     exec.Logger(),
		pm:      pm,
		r:       r,
		port:    port,
	}
	done, err := exec.Run(proc.makeDataHandlerMap(), proc.makeCtrlHandlerMap())
	if err != nil {
		return nil, err
	}
	go func() {
		<-done
		if err := proc.Stop(); err != nil && err != app.ErrAlreadyStopped {
			proc.log.Warnf("stopped with error: %s", err)
		}
		proc.log.Info("stopped cleanly")
	}()
	return proc, nil
}

// ProcID returns the process ID.
func (ar *AppProc) ProcID() ProcID {
	return ar.pid
}

// Stopped returns true if process is stopped.
func (ar *AppProc) Stopped() bool {
	return *(*bool)(atomic.LoadPointer(&ar.stopped))
}

// Stop sends SIGTERM to the hosted App, closes all loops and ends handling of packets.
// An error is returned if app does not end cleanly, or if app already stopped.
func (ar *AppProc) Stop() error {
	ar.mx.Lock()
	defer ar.mx.Unlock()

	// load 'true' to 'stopped'.
	t := true
	atomic.StorePointer(&ar.stopped, unsafe.Pointer(&t)) //nolint:gosec

	ar.log.Info("stopping...")

	for lm := range ar.lps {
		_ = ar.r.CloseLoop(lm)                               //nolint:errcheck
		_, _ = ar.e.Call(appnet.FrameCloseLoop, lm.Encode()) //nolint:errcheck
		delete(ar.lps, lm)
	}

	return ar.e.Stop()
}

// ConfirmLoop attempts to confirm a loop with the hosted App.
func (ar *AppProc) ConfirmLoop(lm app.LoopMeta, tpID uuid.UUID, rtID routing.RouteID, nsMsg []byte) ([]byte, error) {
	ar.mx.Lock()
	defer ar.mx.Unlock()

	// if loop 'lm' is not found, sets a new loop with tpID and rtID.
	// else, retrieve loop 'lm'.
	// returns true if a new loop is set.
	setOrGetLoop := func(lm app.LoopMeta, tpID uuid.UUID, rtID routing.RouteID) (*loopDispatch, bool) {
		if ld, ok := ar.lps[lm]; ok {
			return ld, false
		}
		ld := &loopDispatch{trID: tpID, rtID: rtID}
		ar.lps[lm] = ld
		return ld, true
	}

	ld, isNew := setOrGetLoop(lm, tpID, rtID)
	if isNew {
		ns, err := noise.KKAndSecp256k1(noise.Config{
			LocalPK:   ar.e.Config().HostPK,
			LocalSK:   ar.e.Config().HostSK,
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
		if _, err := ar.e.Call(appnet.FrameConfirmLoop, lm.Encode()); err != nil {
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
	if _, err := ar.e.Call(appnet.FrameConfirmLoop, lm.Encode()); err != nil {
		ar.log.Warnf("Failed to notify App about new loop: %s", err)
	}
	return nil, nil
}

// ConfirmCloseLoop attempts to inform the hosted App that a loop is closed.
func (ar *AppProc) ConfirmCloseLoop(lm app.LoopMeta) error {
	ar.mx.Lock()
	defer ar.mx.Unlock()

	delete(ar.lps, lm)

	if _, err := ar.e.Call(appnet.FrameCloseLoop, lm.Encode()); err != nil {
		return err
	}
	ar.log.Infof("confirm close loop: %s", lm.String())
	return nil
}

// ConsumePacket attempts to send a DataFrame to the hosted App.
func (ar *AppProc) ConsumePacket(lm app.LoopMeta, ciphertext []byte) error {
	ar.mx.RLock()
	defer ar.mx.RUnlock()

	ld, ok := ar.lps[lm]
	if !ok {
		return ErrLoopNotFound
	}
	plaintext, err := ld.ns.Decrypt(ciphertext)
	if err != nil {
		return fmt.Errorf("%s: %s", ErrDecryptionFailed.Error(), err.Error())
	}
	df := &app.DataFrame{Meta: lm, Data: plaintext}
	_, err = ar.e.Call(appnet.FrameData, df.Encode())
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

func (ar *AppProc) makeDataHandlerMap() appnet.HandlerMap {

	// triggered when App sends 'CreateLoop' frame to Host
	requestLoop := func(rAddr app.LoopAddr) respondFunc {
		ar.mx.Lock()
		defer ar.mx.Unlock()

		// prepare noise
		ns, err := noise.KKAndSecp256k1(noise.Config{
			LocalPK:   ar.e.Config().HostPK,
			LocalSK:   ar.e.Config().HostSK,
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
		lPort := ar.pm.AllocPort(ar.pid)

		lm := app.LoopMeta{
			Local:  app.LoopAddr{PubKey: ar.e.Config().HostPK, Port: lPort},
			Remote: rAddr,
		}

		// keep track of the new loop (if not already exists)
		if _, ok := ar.lps[lm]; ok {
			return failWith(ErrLoopAlreadyExists)
		}
		ar.lps[lm] = &loopDispatch{ns: ns}

		// if loop is of loopback type (dst app is on localhost) send to local app, else send to router.
		if lm.IsLoopback() {
			return func() ([]byte, error) {
				a2, ok := ar.pm.ProcOfPort(lm.Remote.Port)
				if !ok {
					return nil, ErrProcNotFound
				}
				_, err := a2.e.Call(appnet.FrameConfirmLoop, lm.Swap().Encode())
				return lm.Encode(), err
			}
		}
		return func() ([]byte, error) {
			return lm.Encode(), ar.r.FindRoutesAndSetupLoop(lm, msg)
		}
	}

	// triggered when App sends 'CloseLoop' frame to Host
	closeLoop := func(lm app.LoopMeta) respondFunc {
		ar.mx.Lock()
		delete(ar.lps, lm)
		ar.mx.Unlock()

		if lm.IsLoopback() {
			return func() ([]byte, error) {
				a2, ok := ar.pm.ProcOfPort(lm.Remote.Port)
				if !ok {
					return nil, ErrProcNotFound
				}
				_, err := a2.e.Call(appnet.FrameCloseLoop, lm.Encode())
				return nil, err
			}
		}
		return func() ([]byte, error) {
			return nil, ar.r.CloseLoop(lm)
		}
	}

	// triggered when App sends 'Data' frame to Host
	fwdPacket := func(lm app.LoopMeta, plaintext []byte) respondFunc {
		if lm.IsLoopback() {
			return func() ([]byte, error) {
				rA, ok := ar.pm.ProcOfPort(lm.Remote.Port)
				if !ok {
					return nil, ErrLoopNotFound
				}
				df := app.DataFrame{Meta: *lm.Swap(), Data: plaintext}
				_, err := rA.e.Call(appnet.FrameData, df.Encode())
				return nil, err
			}
		}
		ar.mx.RLock()
		ld, ok := ar.lps[lm]
		ar.mx.RUnlock()
		if !ok {
			return failWith(ErrLoopNotFound)
		}
		return func() ([]byte, error) {
			return nil, ar.r.ForwardPacket(ld.trID, ld.rtID, ld.ns.Encrypt(plaintext))
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

func (ar *AppProc) makeCtrlHandlerMap() appnet.HandlerMap {
	// TODO(evanlinjin): implement.
	return appnet.HandlerMap{}
}

// ProcManager manages local Apps and the associated ports and loops.
type ProcManager interface {
	RunProc(r Router, port uint16, exec app.Executor) (*AppProc, error)
	AllocPort(pid ProcID) uint16

	Proc(pid ProcID) (*AppProc, bool)
	ProcOfPort(lPort uint16) (*AppProc, bool)
	RangeProcIDs(fn ProcIDFunc)
	RangePorts(fn PortFunc)
	ListProcs() []*ProcInfo

	Close() error
}

// NewProcManager creates a new ProcManager.
func NewProcManager(minPort uint16) ProcManager {
	return &procManager{
		minPort:     minPort,
		currentPort: minPort - 1,
		currentPID:  0,
		ports:       make(map[uint16]ProcID),
		procs:       make(map[ProcID]*AppProc),
		log:         logging.MustGetLogger("apps_manager"),
	}
}

type procManager struct {
	minPort     uint16
	currentPort uint16
	currentPID  ProcID

	ports map[uint16]ProcID
	procs map[ProcID]*AppProc

	mx  sync.RWMutex
	log *logging.Logger
}

func (pm *procManager) RunProc(r Router, port uint16, exec app.Executor) (*AppProc, error) {
	pm.mx.Lock()
	defer pm.mx.Unlock()

	// grab next available pid
	pid := pm.nextFreePID()

	// check port
	if port != 0 {
		if proc, ok := pm.portAllocated(port); ok {
			return nil, fmt.Errorf("port already allocated to pid %d", proc.ProcID())
		}
	}


	// [2019-04-23T17:18:54+08:00] INFO [proc.2(chat)]: log message.
	log := logging.MustGetLogger(fmt.Sprintf("proc.%d(%s)", pid, exec.Meta().AppName))
	exec.SetLogger(log)

	// run app
	proc, err := NewAppProc(pm, r, port, pid, exec)
	if err != nil {
		return nil, err
	}
	pm.procs[pid] = proc

	// assign port
	if port != 0 {
		pm.ports[port] = pid
		pm.log.Infof("port %d allocated to pid %d", port, proc.ProcID())
	}

	return proc, nil
}

func (pm *procManager) AllocPort(pid ProcID) uint16 {
	pm.mx.Lock()
	defer pm.mx.Unlock()

	port := pm.nextFreePort()
	pm.ports[port] = pid
	return port
}

func (pm *procManager) Proc(pid ProcID) (*AppProc, bool) {
	pm.mx.RLock()
	defer pm.mx.RUnlock()

	if proc, ok := pm.procs[pid]; ok && !proc.Stopped() {
		return proc, true
	}
	return nil, false
}

// ProcInfo holds information about procs to be used on RPC methods to display such information
type ProcInfo struct {
	PID  ProcID `json:"proc-id"`
	Port uint16 `json:"port"`
	*app.ExecConfig
	*app.Meta
}

// ListProcs list meta info about the processes managed by procManager
func (pm *procManager) ListProcs() []*ProcInfo {
	pm.mx.RLock()
	defer pm.mx.RUnlock()

	procsList := make([]*ProcInfo, len(pm.procs))
	i := 0
	for pid, proc := range pm.procs {
		procsList[i] = &ProcInfo{
			PID:        pid,
			Port:       proc.port,
			ExecConfig: proc.e.Config(),
			Meta:       proc.e.Meta(),
		}
	}

	return procsList
}

func (pm *procManager) ProcOfPort(lPort uint16) (*AppProc, bool) {
	pm.mx.RLock()
	defer pm.mx.RUnlock()

	return pm.portAllocated(lPort)
}

// ProcIDFunc is triggered when ranging all running processes.
type ProcIDFunc func(pid ProcID, proc *AppProc) (next bool)

func (pm *procManager) RangeProcIDs(fn ProcIDFunc) {
	pm.mx.RLock()
	defer pm.mx.RUnlock()

	for pid, proc := range pm.procs {
		if proc.Stopped() {
			continue
		}
		if next := fn(pid, proc); next {
			continue
		}
		return
	}
}

// PortFunc is triggered when ranging all allocated ports.
type PortFunc func(port uint16, proc *AppProc) (next bool)

func (pm *procManager) RangePorts(fn PortFunc) {
	pm.mx.RLock()
	defer pm.mx.RUnlock()

	for port, pid := range pm.ports {
		proc, ok := pm.procs[pid]
		if !ok || proc.Stopped() {
			continue
		}
		if next := fn(port, proc); next {
			continue
		}
		return
	}
}

func (pm *procManager) Close() error {
	pm.mx.Lock()
	defer pm.mx.Unlock()

	pm.ports = make(map[uint16]ProcID)

	for pid, proc := range pm.procs {
		if !proc.Stopped() {
			_ = proc.Stop() //nolint:errcheck
		}
		delete(pm.procs, pid)
	}

	return nil
}

// returns true (with the proc) if given proc of pid is running.
func (pm *procManager) procRunning(pid ProcID) (*AppProc, bool) {
	if proc, ok := pm.procs[pid]; ok && !proc.Stopped() {
		return proc, true
	}
	return nil, false
}

// returns true (with the proc) id port is allocated to a running app.
func (pm *procManager) portAllocated(port uint16) (*AppProc, bool) {
	pid, ok := pm.ports[port]
	if !ok {
		return nil, false
	}
	return pm.procRunning(pid)
}

// returns the next available and valid pid.
func (pm *procManager) nextFreePID() ProcID {
	for {
		if pm.currentPID++; pm.currentPID == 0 {
			continue
		}
		if _, ok := pm.procRunning(pm.currentPID); ok {
			continue
		}
		return pm.currentPID
	}
}

// returns the next available and valid port.
func (pm *procManager) nextFreePort() uint16 {
	for {
		if pm.currentPort++; pm.currentPort < pm.minPort {
			continue
		}
		if _, ok := pm.portAllocated(pm.currentPort); ok {
			continue
		}
		return pm.currentPort
	}
}
