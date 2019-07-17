package transport

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"time"

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
	Logger *logging.Logger

	config     *ManagerConfig
	factories  map[string]Factory
	transports map[uuid.UUID]*ManagedTransport
	entries    map[Entry]struct{}

	doneChan    chan struct{}
	SetupTpChan chan Transport
	DataTpChan  chan *ManagedTransport
	mu          sync.RWMutex

	mgrQty int32 // Count of spawned manageTransport goroutines

	setupNodes []cipher.PubKey
}

// NewManager creates a Manager with the provided configuration and transport factories.
// 'factories' should be ordered by preference.
func NewManager(config *ManagerConfig, factories ...Factory) (*Manager, error) {
	entries, err := config.DiscoveryClient.GetTransportsByEdge(context.Background(), config.PubKey)
	if err != nil {
		entries = make([]*EntryWithStatus, 0)
	}

	mEntries := make(map[Entry]struct{})
	for _, entry := range entries {
		mEntries[*entry.Entry] = struct{}{}
	}

	fMap := make(map[string]Factory)
	for _, factory := range factories {
		fMap[factory.Type()] = factory
	}

	return &Manager{
		Logger:      logging.MustGetLogger("trmanager"),
		config:      config,
		factories:   fMap,
		transports:  make(map[uuid.UUID]*ManagedTransport),
		entries:     mEntries,
		SetupTpChan: make(chan Transport, 9),         // TODO: eliminate or justify buffering here
		DataTpChan:  make(chan *ManagedTransport, 9), // TODO: eliminate or justify buffering here
		doneChan:    make(chan struct{}),
	}, nil
}

// SetupNodes returns setup node list contained within the TransportManager.
func (tm *Manager) SetupNodes() []cipher.PubKey {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	return tm.setupNodes
}

// SetSetupNodes sets setup node list contained within the TransportManager.
func (tm *Manager) SetSetupNodes(nodes []cipher.PubKey) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.setupNodes = nodes
}

// Factories returns all the factory types contained within the TransportManager.
func (tm *Manager) Factories() []string {
	fTypes, i := make([]string, len(tm.factories)), 0
	for _, f := range tm.factories {
		fTypes[i], i = f.Type(), i+1
	}
	return fTypes
}

// Transport obtains a Transport via a given Transport ID.
func (tm *Manager) Transport(id uuid.UUID) *ManagedTransport {
	tm.mu.RLock()
	tr := tm.transports[id]
	tm.mu.RUnlock()
	return tr
}

// WalkTransports ranges through all transports.
func (tm *Manager) WalkTransports(walk func(tp *ManagedTransport) bool) {
	tm.mu.RLock()
	for _, tp := range tm.transports {
		if ok := walk(tp); !ok {
			break
		}
	}
	tm.mu.RUnlock()
}

// reconnectTransports tries to reconnect previously established transports.
func (tm *Manager) reconnectTransports(ctx context.Context) {
	tm.mu.RLock()
	entries := make(map[Entry]struct{})
	for tmEntry := range tm.entries {
		entries[tmEntry] = struct{}{}
	}
	tm.mu.RUnlock()
	for entry := range entries {
		if tm.Transport(entry.ID) != nil {
			continue
		}

		remote, ok := tm.Remote(entry.Edges())
		if !ok {
			tm.Logger.Warnf("Failed to re-establish transport: remote pk not found in edges")
			continue
		}

		_, err := tm.createTransport(ctx, remote, entry.Type, entry.Public)
		if err != nil {
			tm.Logger.Warnf("Failed to re-establish transport: %s", err)
			continue
		}

		if _, err := tm.config.DiscoveryClient.UpdateStatuses(ctx, &Status{ID: entry.ID, IsUp: true}); err != nil {
			tm.Logger.Warnf("Failed to change transport status: %s", err)
		}
	}
}

// Local returns Manager.config.PubKey
func (tm *Manager) Local() cipher.PubKey {
	return tm.config.PubKey
}

// Remote returns the key from the edges that is not equal to Manager.config.PubKey
// in case when both edges are different - returns  (cipher.PubKey{}, false)
func (tm *Manager) Remote(edges [2]cipher.PubKey) (cipher.PubKey, bool) {
	if tm.config.PubKey == edges[0] {
		return edges[1], true
	}
	if tm.config.PubKey == edges[1] {
		return edges[0], true
	}
	return cipher.PubKey{}, false
}

// createDefaultTransports created transports to DefaultNodes if they don't exist.
func (tm *Manager) createDefaultTransports(ctx context.Context) {
	for _, pk := range tm.config.DefaultNodes {
		pk := pk
		exist := false
		tm.WalkTransports(func(tr *ManagedTransport) bool {
			remote, ok := tm.Remote(tr.Edges())
			if ok && (remote == pk) {
				exist = true
				return false
			}
			return true
		})
		if exist {
			continue
		}
		_, err := tm.CreateTransport(ctx, pk, "messaging", true)
		if err != nil {
			tm.Logger.Warnf("Failed to establish transport to a node %s: %s", pk, err)
		}
	}
}

// Serve runs listening loop across all registered factories.
func (tm *Manager) Serve(ctx context.Context) error {
	tm.reconnectTransports(ctx)
	tm.createDefaultTransports(ctx)

	var wg sync.WaitGroup
	for _, factory := range tm.factories {
		wg.Add(1)
		go func(f Factory) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					tm.Logger.Info("Received ctx.Done()")
					return
				case <-tm.doneChan:
					tm.Logger.Info("Received tm.doneCh")
					return
				default:
					if _, err := tm.acceptTransport(ctx, f); err != nil {
						if strings.Contains(err.Error(), "closed") {
							return
						}

						tm.Logger.Warnf("Failed to accept connection: %s", err)
					}
				}

			}
		}(factory)
	}

	tm.Logger.Info("Starting transport manager")
	wg.Wait()
	return nil
}

// CreateTransport begins to attempt to establish transports to the given 'remote' node.
func (tm *Manager) CreateTransport(ctx context.Context, remote cipher.PubKey, tpType string, public bool) (*ManagedTransport, error) {
	return tm.createTransport(ctx, remote, tpType, public)
}

// DeleteTransport disconnects and removes the Transport of Transport ID.
func (tm *Manager) DeleteTransport(id uuid.UUID) error {
	tm.mu.Lock()
	if tr, ok := tm.transports[id]; ok {
		if err := tr.Close(); err != nil {
			tm.Logger.Warnf("Failed to close transport: %s", err)
		}
		delete(tm.transports, id)
	}
	tm.mu.Unlock()

	if _, err := tm.config.DiscoveryClient.UpdateStatuses(context.Background(), &Status{ID: id, IsUp: false}); err != nil {
		tm.Logger.Warnf("Failed to change transport status: %s", err)
	}

	tm.Logger.Infof("Unregistered transport %s", id)
	return nil
}

// Close closes opened transports and registered factories.
func (tm *Manager) Close() error {
	if tm == nil {
		return nil
	}

	close(tm.doneChan)

	tm.Logger.Info("Closing transport manager")
	tm.mu.Lock()
	statuses := make([]*Status, 0)
	for _, tr := range tm.transports {
		if !tr.Entry.Public {
			continue
		}
		statuses = append(statuses, &Status{ID: tr.Entry.ID, IsUp: false})

		go func(tr io.Closer) {
			if err := tr.Close(); err != nil {
				tm.Logger.Warnf("Failed to close transport: %s", err)
			}
		}(tr)
	}
	tm.mu.Unlock()

	if _, err := tm.config.DiscoveryClient.UpdateStatuses(context.Background(), statuses...); err != nil {
		tm.Logger.Warnf("Failed to change transport status: %s", err)
	}

	for _, f := range tm.factories {
		go func(f io.Closer) {
			if err := f.Close(); err != nil {
				tm.Logger.Warnf("Failed to close factory: %s", err)
			}
		}(f)
	}

	return nil
}

func (tm *Manager) dialTransport(ctx context.Context, factory Factory, remote cipher.PubKey, public bool) (Transport, *Entry, error) {
	if tm.isClosing() {
		return nil, nil, errors.New("transport.Manager is closing. Skipping dialing transport")
	}

	tr, err := factory.Dial(ctx, remote)
	if err != nil {
		return nil, nil, err
	}

	var entry *Entry
	if tm.IsSetupTransport(tr) {
		entry = makeEntry(tr, public)
	} else {
		entry, err = settlementInitiatorHandshake(public).Do(tm, tr, time.Minute)
		if err != nil {
			go func() {
				if err := tr.Close(); err != nil {
					tm.Logger.Warnf("Failed to close transport: %s", err)
				}
			}()
			return nil, nil, err
		}
	}

	return tr, entry, nil
}

func (tm *Manager) createTransport(ctx context.Context, remote cipher.PubKey, tpType string, public bool) (*ManagedTransport, error) {
	factory := tm.factories[tpType]
	if factory == nil {
		return nil, errors.New("unknown transport type")
	}

	tr, entry, err := tm.dialTransport(ctx, factory, remote, public)
	if err != nil {
		return nil, err
	}

	oldTr := tm.Transport(entry.ID)
	if oldTr != nil {
		oldTr.killWorker()
	}

	tm.Logger.Infof("Dialed to %s using %s factory. Transport ID: %s", remote, tpType, entry.ID)
	mTr := newManagedTransport(tr, *entry, false)

	tm.mu.Lock()
	tm.transports[entry.ID] = mTr
	tm.mu.Unlock()

	var setupTpChan chan Transport
	var dataTpChan chan *ManagedTransport

	if tm.IsSetupTransport(tr) {
		setupTpChan = tm.SetupTpChan
	} else {
		dataTpChan = tm.DataTpChan
	}

	select {
	case <-tm.doneChan:
		return nil, io.ErrClosedPipe
	case dataTpChan <- mTr:
		go tm.manageTransport(ctx, mTr, factory, remote)
		return mTr, nil
	case setupTpChan <- mTr:
		return mTr, nil
	}
}

func (tm *Manager) acceptTransport(ctx context.Context, factory Factory) (*ManagedTransport, error) {
	tr, err := factory.Accept(ctx)
	if err != nil {
		return nil, err
	}

	if tm.isClosing() {
		return nil, errors.New("transport.Manager is closing. Skipping incoming transport")
	}

	var setupTpChan chan Transport
	var dataTpChan chan *ManagedTransport

	var entry *Entry
	if tm.IsSetupTransport(tr) {
		setupTpChan = tm.SetupTpChan
		entry = makeEntry(tr, false)
	} else {
		dataTpChan = tm.DataTpChan
		entry, err = settlementResponderHandshake().Do(tm, tr, 30*time.Second)
		if err != nil {
			go func() {
				if err = tr.Close(); err != nil {
					tm.Logger.Warnf("Failed to close transport: %s", err)
				}
			}()
			return nil, err
		}
	}

	remote, ok := tm.Remote(tr.Edges())
	if !ok {
		return nil, errors.New("remote pubkey not found in edges")
	}

	tm.Logger.Infof("Accepted new transport with type %s from %s. ID: %s", factory.Type(), remote, entry.ID)

	oldTr := tm.Transport(entry.ID)
	if oldTr != nil {
		oldTr.killWorker()
	}

	mTr := newManagedTransport(tr, *entry, true)

	tm.mu.Lock()
	tm.transports[entry.ID] = mTr
	tm.mu.Unlock()

	select {
	case <-tm.doneChan:
		return nil, io.ErrClosedPipe
	case dataTpChan <- mTr:
		go tm.manageTransport(ctx, mTr, factory, remote)
		return mTr, nil
	case setupTpChan <- mTr:
		return mTr, nil
	}
}

func (tm *Manager) addEntry(entry *Entry) {
	tm.mu.Lock()
	tm.entries[*entry] = struct{}{}
	tm.mu.Unlock()
}

func (tm *Manager) addIfNotExist(entry *Entry) (isNew bool) {
	tm.mu.Lock()
	if _, ok := tm.entries[*entry]; !ok {
		tm.entries[*entry] = struct{}{}
		isNew = true
	}
	tm.mu.Unlock()
	return isNew
}

func (tm *Manager) isClosing() bool {
	select {
	case <-tm.doneChan:
		return true
	default:
		return false
	}
}

func (tm *Manager) manageTransport(ctx context.Context, mTr *ManagedTransport, factory Factory, remote cipher.PubKey) {
	logTicker := time.NewTicker(logWriteInterval)
	logUpdate := false

	mgrQty := atomic.AddInt32(&tm.mgrQty, 1)
	tm.Logger.Infof("Spawned manageTransport for mTr.ID: %v. mgrQty: %v PK: %s", mTr.Entry.ID, mgrQty, remote)

	defer func() {
		logTicker.Stop()
		if logUpdate {
			if err := tm.config.LogStore.Record(mTr.Entry.ID, mTr.LogEntry); err != nil {
				tm.Logger.Warnf("Failed to record log entry: %s", err)
			}
		}
		mTr.killUpdate()

		mgrQty := atomic.AddInt32(&tm.mgrQty, -1)
		tm.Logger.Infof("manageTransport exit for %v. mgrQty: %v", mTr.Entry.ID, mgrQty)
	}()

	for {
		select {
		case <-mTr.done:
			return

		case <-logTicker.C:
			if logUpdate {
				if err := tm.config.LogStore.Record(mTr.Entry.ID, mTr.LogEntry); err != nil {
					tm.Logger.Warnf("Failed to record log entry: %s", err)
				}
			}

		case err, ok := <-mTr.update:
			if !ok {
				return
			}

			if err == nil {
				logUpdate = true
				continue
			}

			tm.Logger.Infof("Transport %s failed with error: %s. Re-dialing...", mTr.Entry.ID, err)
			if _, err := tm.config.DiscoveryClient.UpdateStatuses(ctx, &Status{ID: mTr.Entry.ID, IsUp: false, Updated: time.Now().UnixNano()}); err != nil {
				tm.Logger.Warnf("Failed to change transport status: %s", err)
			}

			// If we are the acceptor, we are not responsible for restarting transport.
			// If the transport is private, we don't need to restart.
			if mTr.Accepted || !mTr.Entry.Public {
				return
			}

			tr, _, err := tm.dialTransport(ctx, factory, remote, mTr.Entry.Public)
			if err != nil {
				tm.Logger.Infof("Failed to redial Transport %s: %s", mTr.Entry.ID, err)
				continue
			}

			tm.Logger.Infof("Updating transport %s", mTr.Entry.ID)
			if err = mTr.updateTransport(ctx, tr, tm.config.DiscoveryClient); err != nil {
				tm.Logger.Warnf("Failed to update transport: %s", err)
			}
		}
	}
}

// IsSetupTransport checks whether `tr` is running in the `setup` mode.
func (tm *Manager) IsSetupTransport(tr Transport) bool {
	for _, pk := range tm.setupNodes {
		remote, ok := tm.Remote(tr.Edges())
		if ok && (remote == pk) {
			return true
		}
	}

	return false
}
