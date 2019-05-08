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

//type ProcInfo struct {
//	PID  ProcID         `json:"pid"`
//	App  app.Meta       `json:"app"`
//	Exec app.ExecConfig `json:"exec"`
//}

// AppProc hosts an App and keeps track or whether the App is running or not.
// It also manages incoming/outgoing packets to/from the App when the app is running.
type AppProc struct {

	// stores a boolean; 'true' if no longer running.
	// unsafe.Pointer is used alongside 'atomic' module for fast, thread-safe access.
	stopped unsafe.Pointer

	pid ProcID
	e   app.Executor
	lps map[app.LoopMeta]*loopDispatch
	mx  sync.RWMutex
	log *logging.Logger

	// back-references
	pm ProcManager
	r  Router
}

// NewAppProc creates a new AppProc
func NewAppProc(pm ProcManager, r Router, pid ProcID, m *app.Meta, c *app.ExecConfig) (*AppProc, error) {
	// [2019-04-23T17:18:54+08:00] INFO [proc.2(chat)]: log message.
	log := logging.MustGetLogger(fmt.Sprintf("proc.%d(%s)", pid, c.AppName()))

	exec, err := app.NewExecutor(log, m, c)
	if err != nil {
		return nil, err
	}
	proc := &AppProc{
		stopped: unsafe.Pointer(new(bool)), //nolint:gosec
		pid:     pid,
		e:       exec,
		lps:     make(map[app.LoopMeta]*loopDispatch),
		log:     log,
		pm:      pm,
		r:       r,
	}
	done, err := exec.Run(makeDataHandlers(proc), makeCtrlHandlers(proc))
	if err != nil {
		return nil, err
	}
	go func() {
		<-done
		if err := proc.Stop(); err != nil && err != app.ErrAlreadyStopped {
			log.Warnf("stopped with error: %s", err)
		}
		log.Info("stopped cleanly")
	}()
	return proc, nil
}

// ProcID returns the process ID.
func (ap *AppProc) ProcID() ProcID {
	return ap.pid
}

// Stopped returns true if process is stopped.
func (ap *AppProc) Stopped() bool {
	return *(*bool)(atomic.LoadPointer(&ap.stopped))
}

// Stop sends SIGTERM to the hosted App, closes all loops and ends handling of packets.
// An error is returned if app does not end cleanly, or if app already stopped.
func (ap *AppProc) Stop() error {
	ap.mx.Lock()
	defer ap.mx.Unlock()

	// load 'true' to 'stopped'.
	t := true
	atomic.StorePointer(&ap.stopped, unsafe.Pointer(&t)) //nolint:gosec

	ap.log.Info("stopping...")

	for lm := range ap.lps {
		_ = ap.r.CloseLoop(lm)                               //nolint:errcheck
		_, _ = ap.e.Call(appnet.FrameCloseLoop, lm.Encode()) //nolint:errcheck
		delete(ap.lps, lm)
	}

	return ap.e.Stop()
}

// ConfirmLoop attempts to confirm a loop with the hosted App.
func (ap *AppProc) ConfirmLoop(lm app.LoopMeta, tpID uuid.UUID, rtID routing.RouteID, nsMsg []byte) ([]byte, error) {
	ap.mx.Lock()
	defer ap.mx.Unlock()

	// if loop 'lm' is not found, sets a new loop with tpID and rtID.
	// else, retrieve loop 'lm'.
	// returns true if a new loop is set.
	setOrGetLoop := func(lm app.LoopMeta, tpID uuid.UUID, rtID routing.RouteID) (*loopDispatch, bool) {
		if ld, ok := ap.lps[lm]; ok {
			return ld, false
		}
		ld := &loopDispatch{trID: tpID, rtID: rtID}
		ap.lps[lm] = ld
		return ld, true
	}

	ld, isNew := setOrGetLoop(lm, tpID, rtID)
	if isNew {
		ns, err := noise.KKAndSecp256k1(noise.Config{
			LocalPK:   ap.e.Config().HostPK,
			LocalSK:   ap.e.Config().HostSK,
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
		if _, err := ap.e.Call(appnet.FrameConfirmLoop, lm.Encode()); err != nil {
			ap.log.Warnf("Failed to notify App about new loop: %s", err)
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
	if _, err := ap.e.Call(appnet.FrameConfirmLoop, lm.Encode()); err != nil {
		ap.log.Warnf("Failed to notify App about new loop: %s", err)
	}
	return nil, nil
}

// ConfirmCloseLoop attempts to inform the hosted App that a loop is closed.
func (ap *AppProc) ConfirmCloseLoop(lm app.LoopMeta) error {
	ap.mx.Lock()
	defer ap.mx.Unlock()

	delete(ap.lps, lm)

	if _, err := ap.e.Call(appnet.FrameCloseLoop, lm.Encode()); err != nil {
		return err
	}
	ap.log.Infof("confirm close loop: %s", lm.String())
	return nil
}

// ConsumePacket attempts to send a DataFrame to the hosted App.
func (ap *AppProc) ConsumePacket(lm app.LoopMeta, ciphertext []byte) error {
	ap.mx.RLock()
	defer ap.mx.RUnlock()

	ld, ok := ap.lps[lm]
	if !ok {
		return ErrLoopNotFound
	}
	plaintext, err := ld.ns.Decrypt(ciphertext)
	if err != nil {
		return fmt.Errorf("%s: %s", ErrDecryptionFailed.Error(), err.Error())
	}
	df := &app.DataFrame{Meta: lm, Data: plaintext}
	_, err = ap.e.Call(appnet.FrameData, df.Encode())
	return err
}

// ProcManager manages local Apps and the associated ports and loops.
type ProcManager interface {
	RunProc(r Router, port uint16, m *app.Meta, c *app.ExecConfig) (*AppProc, error)
	AllocPort(pid ProcID) uint16

	Proc(pid ProcID) (*AppProc, bool)
	ProcOfPort(lPort uint16) (*AppProc, bool)
	RangeProcIDs(fn ProcIDFunc)
	RangePorts(fn PortFunc)

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

func (pm *procManager) RunProc(r Router, port uint16, m *app.Meta, c *app.ExecConfig) (*AppProc, error) {
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

	// run app
	proc, err := NewAppProc(pm, r, pid, m, c)
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
