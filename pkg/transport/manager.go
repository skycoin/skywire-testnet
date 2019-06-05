package transport

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/cipher"
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

	config *ManagerConfig

	factories map[string]Factory
	fMx       sync.RWMutex

	transports map[uuid.UUID]*ManagedTransport
	entries    map[Entry]struct{}
	tpMx       sync.RWMutex

	doneChan chan struct{}
	TrChan   chan *ManagedTransport
}

// NewManager creates a Manager with the provided configuration and transport factories.
// 'factories' should be ordered by preference.
func NewManager(config *ManagerConfig, factories ...Factory) (*Manager, error) {
	entries, _ := config.DiscoveryClient.GetTransportsByEdge(context.Background(), config.PubKey) // nolint

	mEntries := make(map[Entry]struct{})
	for _, entry := range entries {
		mEntries[*entry.Entry] = struct{}{}
	}

	fMap := make(map[string]Factory)
	for _, factory := range factories {
		fMap[factory.Type()] = factory
	}

	return &Manager{
		Logger:     logging.MustGetLogger("trmanager"),
		config:     config,
		factories:  fMap,
		transports: make(map[uuid.UUID]*ManagedTransport),
		entries:    mEntries,
		TrChan:     make(chan *ManagedTransport, 9), // TODO: eliminate or justify buffering here
		doneChan:   make(chan struct{}),
	}, nil
}

// Factories returns all the factory types contained within the TransportManager.
func (tm *Manager) Factories() []string {
	tm.fMx.RLock()
	fTypes, i := make([]string, len(tm.factories)), 0
	for _, f := range tm.factories {
		fTypes[i], i = f.Type(), i+1
	}
	tm.fMx.RUnlock()
	return fTypes
}

// Transport obtains a Transport via a given Transport ID.
func (tm *Manager) Transport(id uuid.UUID) *ManagedTransport {
	tm.tpMx.RLock()
	tr := tm.transports[id]
	tm.tpMx.RUnlock()
	return tr
}

// WalkTransports ranges through all transports.
func (tm *Manager) WalkTransports(walk func(tp *ManagedTransport) bool) {
	tm.tpMx.RLock()
	for _, tp := range tm.transports {
		if ok := walk(tp); !ok {
			break
		}
	}
	tm.tpMx.RUnlock()
}

// reconnectTransports tries to reconnect previously established transports.
func (tm *Manager) reconnectTransports(ctx context.Context) {
	tm.tpMx.RLock()
	entries := make(map[Entry]struct{})
	for tmEntry := range tm.entries {
		entries[tmEntry] = struct{}{}
	}
	tm.tpMx.RUnlock()

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
	tm.fMx.RLock()
	for _, factory := range tm.factories {
		wg.Add(1)
		go func(f Factory) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case <-tm.doneChan:
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
	tm.fMx.RUnlock()

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
	tm.tpMx.Lock()
	tp := tm.transports[id]
	delete(tm.transports, id)
	tm.tpMx.Unlock()

	if _, err := tm.config.DiscoveryClient.UpdateStatuses(context.Background(), &Status{ID: id, IsUp: false}); err != nil {
		tm.Logger.Warnf("Failed to change transport status: %s", err)
	}

	tm.Logger.Infof("Unregistered transport %s", id)
	if tp != nil {
		return tp.Close()
	}

	return nil
}

// Close closes opened transports and registered factories.
func (tm *Manager) Close() error {

	close(tm.doneChan)

	tm.Logger.Info("Closing transport manager")
	tm.tpMx.Lock()
	statuses := make([]*Status, 0)
	for _, tr := range tm.transports {
		if !tr.Public {
			continue
		}
		statuses = append(statuses, &Status{ID: tr.ID, IsUp: false})

		tr.Close()
	}
	tm.tpMx.Unlock()

	if _, err := tm.config.DiscoveryClient.UpdateStatuses(context.Background(), statuses...); err != nil {
		tm.Logger.Warnf("Failed to change transport status: %s", err)
	}

	for _, f := range tm.factories {
		go f.Close()
	}

	return nil
}

func (tm *Manager) dialTransport(ctx context.Context, factory Factory, remote cipher.PubKey, public bool) (Transport, *Entry, error) {

	tr, err := factory.Dial(ctx, remote)
	if err != nil {
		return nil, nil, err
	}

	entry, err := settlementInitiatorHandshake(public).Do(tm, tr, time.Minute)
	if err != nil {
		tr.Close()
		return nil, nil, err
	}

	return tr, entry, nil
}

func (tm *Manager) createTransport(ctx context.Context, remote cipher.PubKey, tpType string, public bool) (*ManagedTransport, error) {
	tm.fMx.RLock()
	factory, ok := tm.factories[tpType]
	tm.fMx.RUnlock()
	if !ok {
		return nil, errors.New("unknown transport type")
	}

	tr, entry, err := tm.dialTransport(ctx, factory, remote, public)
	if err != nil {
		return nil, err
	}

	tm.Logger.Infof("Dialed to %s using %s factory. Transport ID: %s", remote, tpType, entry.ID)
	managedTr := newManagedTransport(entry.ID, tr, entry.Public, false)
	tm.tpMx.Lock()
	tm.transports[entry.ID] = managedTr
	select {
	case <-tm.doneChan:
	case tm.TrChan <- managedTr:
	default:
	}
	tm.tpMx.Unlock()

	go tm.manageTransport(ctx, managedTr, factory, remote, public, false)

	go tm.manageTransportLogs(managedTr)

	return managedTr, nil
}

func (tm *Manager) acceptTransport(ctx context.Context, factory Factory) (*ManagedTransport, error) {
	tr, err := factory.Accept(ctx)
	if err != nil {
		return nil, err
	}

	entry, err := settlementResponderHandshake().Do(tm, tr, 30*time.Second)
	if err != nil {
		tr.Close()
		return nil, err
	}

	remote, ok := tm.Remote(tr.Edges())
	if !ok {
		return nil, errors.New("remote pubkey not found in edges")
	}

	tm.Logger.Infof("Accepted new transport with type %s from %s. ID: %s", factory.Type(), remote, entry.ID)
	managedTr := newManagedTransport(entry.ID, tr, entry.Public, true)
	tm.tpMx.Lock()

	tm.transports[entry.ID] = managedTr
	select {
	case <-tm.doneChan:
	case tm.TrChan <- managedTr:
	default:
	}
	tm.tpMx.Unlock()

	go tm.manageTransport(ctx, managedTr, factory, remote, true, true)

	go tm.manageTransportLogs(managedTr)

	return managedTr, nil
}

func (tm *Manager) walkEntries(walkFunc func(*Entry) bool) *Entry {
	tm.tpMx.Lock()
	defer tm.tpMx.Unlock()

	for entry := range tm.entries {
		if walkFunc(&entry) {
			return &entry
		}
	}

	return nil
}

func (tm *Manager) addEntry(entry *Entry) {
	tm.tpMx.Lock()
	tm.entries[*entry] = struct{}{}
	tm.tpMx.Unlock()
}

func (tm *Manager) addIfNotExist(entry *Entry) (isNew bool) {
	tm.tpMx.Lock()
	if _, ok := tm.entries[*entry]; !ok {
		tm.entries[*entry] = struct{}{}
		isNew = true
	}
	tm.tpMx.Unlock()
	return isNew
}

func (tm *Manager) manageTransport(ctx context.Context, managedTr *ManagedTransport, factory Factory, remote cipher.PubKey, public bool, accepted bool) {
	select {
	case <-managedTr.doneChan:
		tm.Logger.Infof("Transport %s closed", managedTr.ID)
		return
	case err := <-managedTr.errChan:
		if atomic.LoadInt32(&managedTr.isClosing) == 0 {
			tm.Logger.Infof("Transport %s failed with error: %s. Re-dialing...", managedTr.ID, err)
			if accepted {
				if err := tm.DeleteTransport(managedTr.ID); err != nil {
					tm.Logger.Warnf("Failed to delete accepted transport: %s", err)
				}
			} else {
				tr, _, err := tm.dialTransport(ctx, factory, remote, public)
				if err != nil {
					tm.Logger.Infof("Failed to re-dial Transport %s: %s", managedTr.ID, err)
					if err := tm.DeleteTransport(managedTr.ID); err != nil {
						tm.Logger.Warnf("Failed to delete re-dialled transport: %s", err)
					}
				} else {
					managedTr.updateTransport(tr)
				}
			}
		} else {
			tm.Logger.Infof("Transport %s is already closing. Skipped error: %s", managedTr.ID, err)
		}

	}
}

func (tm *Manager) manageTransportLogs(tr *ManagedTransport) {
	for {
		select {
		case <-tr.doneChan:
			return
		case n := <-tr.readLogChan:
			tr.LogEntry.ReceivedBytes.Add(tr.LogEntry.ReceivedBytes, big.NewInt(int64(n)))
		case n := <-tr.writeLogChan:
			tr.LogEntry.SentBytes.Add(tr.LogEntry.SentBytes, big.NewInt(int64(n)))
		}

		if err := tm.config.LogStore.Record(tr.ID, tr.LogEntry); err != nil {
			tm.Logger.Warnf("Failed to record log entry: %s", err)
		}
	}
}
