package transport

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"sync"
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

	config     *ManagerConfig
	factories  map[string]Factory
	transports map[uuid.UUID]*ManagedTransport
	entries    []*Entry

	doneChan       chan struct{}
	acceptedTrChan chan *ManagedTransport
	dialedTrChan   chan *ManagedTransport
	mu             sync.RWMutex
}

// NewManager creates a Manager with the provided configuration and transport factories.
// 'factories' should be ordered by preference.
func NewManager(config *ManagerConfig, factories ...Factory) (*Manager, error) {
	entries, _ := config.DiscoveryClient.GetTransportsByEdge(context.Background(), config.PubKey) // nolint

	mEntries := []*Entry{}
	for _, entry := range entries {
		mEntries = append(mEntries, entry.Entry)
	}

	fMap := make(map[string]Factory)
	for _, factory := range factories {
		fMap[factory.Type()] = factory
	}

	return &Manager{
		Logger:         logging.MustGetLogger("trmanager"),
		config:         config,
		factories:      fMap,
		transports:     make(map[uuid.UUID]*ManagedTransport),
		entries:        mEntries,
		acceptedTrChan: make(chan *ManagedTransport, 10),
		dialedTrChan:   make(chan *ManagedTransport, 10),
		doneChan:       make(chan struct{}),
	}, nil
}

// Observe returns channel for notifications about new Transport
// registration. Only single observer is supported.
func (tm *Manager) Observe() (accept <-chan *ManagedTransport, dial <-chan *ManagedTransport) {
	dialCh := make(chan *ManagedTransport)
	acceptCh := make(chan *ManagedTransport)
	go func() {
		for {
			select {
			case <-tm.doneChan:
				close(dialCh)
				close(acceptCh)
				return
			case tr := <-tm.acceptedTrChan:
				acceptCh <- tr
			case tr := <-tm.dialedTrChan:
				dialCh <- tr
			}
		}
	}()
	return acceptCh, dialCh
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

// ReconnectTransports tries to reconnect previously established transports.
func (tm *Manager) ReconnectTransports(ctx context.Context) {
	tm.mu.RLock()
	entries := tm.entries
	tm.mu.RUnlock()
	for _, entry := range entries {

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

// CreateDefaultTransports created transports to DefaultNodes if they don't exist.
func (tm *Manager) CreateDefaultTransports(ctx context.Context) {
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
	tm.ReconnectTransports(ctx)
	tm.CreateDefaultTransports(ctx)

	var wg sync.WaitGroup
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

	tm.Logger.Info("Starting transport manager")
	wg.Wait()
	return nil
}

// MakeTransportID generates uuid.UUID from pair of keys + type + public
// Generated uuid is:
// - always the same for a given pair
// - GenTransportUUID(keyA,keyB) == GenTransportUUID(keyB, keyA)
func MakeTransportID(keyA, keyB cipher.PubKey, tpType string, public bool) uuid.UUID {
	keys := SortPubKeys(keyA, keyB)
	if public {
		return uuid.NewSHA1(uuid.UUID{},
			append(append(append(keys[0][:], keys[1][:]...), []byte(tpType)...), 1))
	}
	return uuid.NewSHA1(uuid.UUID{},
		append(append(append(keys[0][:], keys[1][:]...), []byte(tpType)...), 0))
}

// SortPubKeys sorts keys so that least-significant comes first
func SortPubKeys(keyA, keyB cipher.PubKey) [2]cipher.PubKey {
	for i := 0; i < 33; i++ {
		if keyA[i] != keyB[i] {
			if keyA[i] < keyB[i] {
				return [2]cipher.PubKey{keyA, keyB}
			}
			return [2]cipher.PubKey{keyB, keyA}
		}
	}
	return [2]cipher.PubKey{keyA, keyB}
}

// SortEdges sorts edges so that list-significant comes firs
func SortEdges(edges [2]cipher.PubKey) [2]cipher.PubKey {
	return SortPubKeys(edges[0], edges[1])
}

// CreateTransport begins to attempt to establish transports to the given 'remote' node.
func (tm *Manager) CreateTransport(ctx context.Context, remote cipher.PubKey, tpType string, public bool) (*ManagedTransport, error) {
	return tm.createTransport(ctx, remote, tpType, public)
}

// DeleteTransport disconnects and removes the Transport of Transport ID.
func (tm *Manager) DeleteTransport(id uuid.UUID) error {
	tm.Logger.Info("Inside Manager.DeleteTransport. Updating map")
	tm.mu.Lock()
	tr := tm.transports[id]
	delete(tm.transports, id)
	tm.mu.Unlock()

	tm.Logger.Info("Inside Manager.DeleteTransport. Updating discovery status..")
	if _, err := tm.config.DiscoveryClient.UpdateStatuses(context.Background(), &Status{ID: id, IsUp: false}); err != nil {
		tm.Logger.Warnf("Failed to change transport status: %s", err)
	}

	tm.Logger.Infof("De-registered transport %s", id)
	if tr != nil {
		return tr.Close()
	}

	return nil
}

// Close closes opened transports and registered factories.
func (tm *Manager) Close() error {
	for _, f := range tm.factories {
		f.Close()
	}

	tm.Logger.Info("Closing transport manager")
	tm.mu.Lock()
	close(tm.doneChan)
	statuses := make([]*Status, 0)
	for _, tr := range tm.transports {
		if !tr.Public {
			continue
		}
		statuses = append(statuses, &Status{ID: tr.ID, IsUp: false})
		tr.Close()
	}
	tm.transports = make(map[uuid.UUID]*ManagedTransport)
	tm.mu.Unlock()

	if _, err := tm.config.DiscoveryClient.UpdateStatuses(context.Background(), statuses...); err != nil {
		tm.Logger.Warnf("Failed to change transport status: %s", err)
	}

	return nil
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

	tm.Logger.Infof("Dialed to %s using %s factory. Transport ID: %s", remote, tpType, entry.ID)
	managedTr := newManagedTransport(entry.ID, tr, entry.Public)
	tm.mu.Lock()
	tm.transports[entry.ID] = managedTr
	select {
	case <-tm.doneChan:
	case tm.dialedTrChan <- managedTr:
	default:
	}
	tm.mu.Unlock()

	go func() {
		select {
		case <-managedTr.doneChan:
			tm.Logger.Infof("Transport %s closed", managedTr.ID)
			return
		case <-tm.doneChan:
			tm.Logger.Infof("Transport %s closed", managedTr.ID)
			return
		case err := <-managedTr.errChan:
			tm.Logger.Infof("Transport %s failed with error: %s. Re-dialing...", managedTr.ID, err)
			tr, _, err := tm.dialTransport(ctx, factory, remote, public)
			if err != nil {
				tm.Logger.Infof("Failed to re-dial Transport %s: %s", managedTr.ID, err)
				if err := tm.DeleteTransport(managedTr.ID); err != nil {
					tm.Logger.Warnf("Failed to delete transport: %s", err)
				}
			} else {
				managedTr.updateTransport(tr)
			}
		}
	}()

	go tm.manageTransportLogs(managedTr)

	return managedTr, nil
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

func (tm *Manager) acceptTransport(ctx context.Context, factory Factory) (*ManagedTransport, error) {
	tr, err := factory.Accept(ctx)
	if err != nil {
		return nil, err
	}

	var handshake settlementHandshake = settlementResponderHandshake
	entry, err := handshake.Do(tm, tr, 30*time.Second)
	if err != nil {
		tr.Close()
		return nil, err
	}

	remote, ok := tm.Remote(tr.Edges())
	if !ok {
		return nil, errors.New("remote pubkey not found in edges")
	}

	tm.Logger.Infof("Accepted new transport with type %s from %s. ID: %s", factory.Type(), remote, entry.ID)
	managedTr := newManagedTransport(entry.ID, tr, entry.Public)
	tm.mu.Lock()

	tm.transports[entry.ID] = managedTr
	select {
	case <-tm.doneChan:
	case tm.acceptedTrChan <- managedTr:
	default:
	}
	tm.mu.Unlock()

	// go func(managedTr *ManagedTransport, tm *Manager) {
	go func() {
		select {
		case <-managedTr.doneChan:
			tm.Logger.Infof("Transport %s closed", managedTr.ID)
			return
		case <-tm.doneChan:
			tm.Logger.Infof("Transport %s closed", managedTr.ID)
			return
		case err := <-managedTr.errChan:
			tm.Logger.Infof("Transport %s failed with error: %s. Re-dialing...", managedTr.ID, err)
			if err := tm.DeleteTransport(managedTr.ID); err != nil {
				tm.Logger.Warnf("Failed to delete transport: %s", err)
			}
		}
	}()

	go tm.manageTransportLogs(managedTr)

	return managedTr, nil
}

func (tm *Manager) walkEntries(walkFunc func(*Entry) bool) *Entry {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	for _, entry := range tm.entries {
		if walkFunc(entry) {
			return entry
		}
	}

	return nil
}

func (tm *Manager) addEntry(entry *Entry) {
	tm.mu.Lock()
	tm.entries = append(tm.entries, entry)
	tm.mu.Unlock()
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
