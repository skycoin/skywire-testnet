package transport

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"

	"github.com/skycoin/skywire/pkg/routing"

	"github.com/google/uuid"
	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/skycoin/src/util/logging"
)

// ManagerConfig configures a Manager.
type ManagerConfig struct {
	PubKey          cipher.PubKey
	SecKey          cipher.SecKey
	DiscoveryClient DiscoveryClient
	LogStore        LogStore
	DefaultNodes    []cipher.PubKey // Nodes to automatically connect to
}

// Manager manages Transports.
type Manager struct {
	Logger   *logging.Logger
	conf     *ManagerConfig
	setupPKS []cipher.PubKey
	facs     map[string]Factory
	tps      map[uuid.UUID]*ManagedTransport

	setupCh chan Transport
	readCh  chan routing.Packet
	mx      sync.RWMutex
	done    chan struct{}
}

// NewManager creates a Manager with the provided configuration and transport factories.
// 'factories' should be ordered by preference.
func NewManager(config *ManagerConfig, setupPKs []cipher.PubKey, factories ...Factory) (*Manager, error) {
	tm := &Manager{
		Logger:   logging.MustGetLogger("tp_manager"),
		conf:     config,
		setupPKS: setupPKs,
		facs:     make(map[string]Factory),
		tps:      make(map[uuid.UUID]*ManagedTransport),
		setupCh:  make(chan Transport, 9), // TODO: eliminate or justify buffering here
		readCh:   make(chan routing.Packet, 20),
		done:     make(chan struct{}),
	}
	for _, factory := range factories {
		tm.facs[factory.Type()] = factory
	}
	return tm, nil
}

// Serve runs listening loop across all registered factories.
func (tm *Manager) Serve(ctx context.Context) error {
	tm.mx.Lock()
	tm.initTransports(ctx)
	tm.mx.Unlock()

	var wg sync.WaitGroup
	for _, factory := range tm.facs {
		wg.Add(1)
		go func(f Factory) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case <-tm.done:
					return
				default:
					if err := tm.acceptTransport(ctx, f); err != nil {
						if strings.Contains(err.Error(), "closed") {
							return
						}
						tm.Logger.Warnf("Failed to accept connection: %s", err)
					}
				}
			}
		}(factory)
	}

	tm.Logger.Info("TransportManager is serving.")
	wg.Wait()
	return nil
}

func (tm *Manager) initTransports(ctx context.Context) {
	entries, err := tm.conf.DiscoveryClient.GetTransportsByEdge(ctx, tm.conf.PubKey)
	if err != nil {
		log.Warnf("No transports found for local node: %v", err)
	}
	for _, entry := range entries {
		var (
			tpType = entry.Entry.Type
			remote = entry.Entry.RemoteEdge(tm.conf.PubKey)
			tpID   = entry.Entry.ID
		)
		if _, err := tm.saveTransport(remote, tpType); err != nil {
			tm.Logger.Warnf("INIT: failed to init tp: type(%s) remote(%s) tpID(%s)", tpType, remote, tpID)
		}
	}
}

func (tm *Manager) acceptTransport(ctx context.Context, factory Factory) error {
	tr, err := factory.Accept(ctx)
	if err != nil {
		return err
	}

	tm.mx.Lock()
	defer tm.mx.Unlock()

	if tm.isClosing() {
		return errors.New("transport.Manager is closing. Skipping incoming transport")
	}

	if tm.IsSetupPK(tr.RemotePK()) {
		tm.setupCh <- tr
		return nil
	}

	// For transports for purpose(data).
	tpID := tm.tpIDFromPK(tr.RemotePK(), tr.Type())

	mTp, ok := tm.tps[tpID]
	if !ok {
		mTp = NewManagedTransport(factory, tm.conf.DiscoveryClient, tm.conf.LogStore, tr.RemotePK(), tm.conf.SecKey)
		if err := mTp.Accept(ctx, tr); err != nil {
			return err
		}
		go mTp.Serve(tm.readCh, tm.done)
		tm.tps[tpID] = mTp

	} else {
		if err := mTp.Accept(ctx, tr); err != nil {
			return err
		}
	}

	tm.Logger.Infof("accepted tp: type(%s) remote(%s) tpID(%s) new(%v)", factory.Type(), tr.RemotePK(), tpID, !ok)
	return nil
}

// SaveTransport begins to attempt to establish data transports to the given 'remote' node.
func (tm *Manager) SaveTransport(ctx context.Context, remote cipher.PubKey, tpType string) (*ManagedTransport, error) {
	tm.mx.Lock()
	defer tm.mx.Unlock()
	if tm.isClosing() {
		return nil, io.ErrClosedPipe
	}
	mTp, err := tm.saveTransport(remote, tpType)
	if err != nil {
		return nil, err
	}
	if err := mTp.Dial(ctx); err != nil {
		tm.Logger.Warnf("underlying 'write' tp failed, will retry: %v", err)
	}
	return mTp, nil
}

func (tm *Manager) saveTransport(remote cipher.PubKey, tpType string) (*ManagedTransport, error) {
	factory, ok := tm.facs[tpType]
	if !ok {
		return nil, errors.New("unknown transport type")
	}

	tpID := tm.tpIDFromPK(remote, tpType)

	tp, ok := tm.tps[tpID]
	if ok {
		return tp, nil
	}

	mTp := NewManagedTransport(factory, tm.conf.DiscoveryClient, tm.conf.LogStore, remote, tm.conf.SecKey)
	go mTp.Serve(tm.readCh, tm.done)
	tm.tps[tpID] = mTp

	tm.Logger.Infof("saved transport: remote(%s) type(%s) tpID(%s)", remote, tpType, tpID)
	return mTp, nil
}

// DeleteTransport disconnects and removes the Transport of Transport ID.
func (tm *Manager) DeleteTransport(id uuid.UUID) {
	tm.mx.Lock()
	defer tm.mx.Unlock()
	if tm.isClosing() {
		return
	}

	if tp, ok := tm.tps[id]; ok {
		tp.Close()
		delete(tm.tps, id)
		tm.Logger.Infof("Unregistered transport %s", id)
	}
}

// ReadPacket reads data packets from routes.
func (tm *Manager) ReadPacket() (routing.Packet, error) {
	p, ok := <-tm.readCh
	if !ok {
		return nil, ErrNotServing
	}
	return p, nil
}

/*
	SETUP LOGIC
*/

// SetupPKs returns setup node list contained within the TransportManager.
func (tm *Manager) SetupPKs() []cipher.PubKey {
	return tm.setupPKS
}

// IsSetupPK checks whether provided `pk` is of `setup` purpose.
func (tm *Manager) IsSetupPK(pk cipher.PubKey) bool {
	for _, sPK := range tm.setupPKS {
		if sPK == pk {
			return true
		}
	}
	return false
}

// DialSetupConn dials to a remote setup node.
func (tm *Manager) DialSetupConn(ctx context.Context, remote cipher.PubKey, tpType string) (Transport, error) {
	tm.mx.Lock()
	defer tm.mx.Unlock()
	if tm.isClosing() {
		return nil, io.ErrClosedPipe
	}

	factory, ok := tm.facs[tpType]
	if !ok {
		return nil, errors.New("unknown transport type")
	}
	tr, err := factory.Dial(ctx, remote)
	if err != nil {
		return nil, err
	}
	tm.Logger.Infof("Dialed to setup node %s using %s factory.", remote, tpType)
	return tr, nil
}

// AcceptSetupConn accepts a connection from a remote setup node.
func (tm *Manager) AcceptSetupConn() (Transport, error) {
	tp, ok := <-tm.setupCh
	if !ok {
		return nil, ErrNotServing
	}
	return tp, nil
}

/*
	STATE
*/

// Factories returns all the factory types contained within the TransportManager.
func (tm *Manager) Factories() []string {
	fTypes, i := make([]string, len(tm.facs)), 0
	for _, f := range tm.facs {
		fTypes[i], i = f.Type(), i+1
	}
	return fTypes
}

// Transport obtains a Transport via a given Transport ID.
func (tm *Manager) Transport(id uuid.UUID) *ManagedTransport {
	tm.mx.RLock()
	tr := tm.tps[id]
	tm.mx.RUnlock()
	return tr
}

// WalkTransports ranges through all transports.
func (tm *Manager) WalkTransports(walk func(tp *ManagedTransport) bool) {
	tm.mx.RLock()
	for _, tp := range tm.tps {
		if ok := walk(tp); !ok {
			break
		}
	}
	tm.mx.RUnlock()
}

// Local returns Manager.config.PubKey
func (tm *Manager) Local() cipher.PubKey {
	return tm.conf.PubKey
}

// Close closes opened transports and registered factories.
func (tm *Manager) Close() error {
	if tm == nil {
		return nil
	}

	tm.mx.Lock()
	defer tm.mx.Unlock()

	close(tm.done)
	tm.Logger.Info("closing transport manager...")
	defer tm.Logger.Infof("transport manager closed.")

	go func() {
		for range tm.readCh {
		}
	}()

	i, statuses := 0, make([]*Status, len(tm.tps))
	for _, tr := range tm.tps {
		tr.close()
		statuses[i] = &Status{ID: tr.Entry.ID, IsUp: false}
		i++
	}
	if _, err := tm.conf.DiscoveryClient.UpdateStatuses(context.Background(), statuses...); err != nil {
		tm.Logger.Warnf("failed to update transport statuses: %v", err)
	}

	tm.Logger.Infof("closing transport factories...")
	for _, f := range tm.facs {
		if err := f.Close(); err != nil {
			tm.Logger.Warnf("Failed to close factory: %s", err)
		}
	}

	close(tm.setupCh)
	close(tm.readCh)
	return nil
}

func (tm *Manager) isClosing() bool {
	select {
	case <-tm.done:
		return true
	default:
		return false
	}
}

func (tm *Manager) tpIDFromPK(pk cipher.PubKey, tpType string) uuid.UUID {
	return MakeTransportID(tm.conf.PubKey, pk, tpType)
}
