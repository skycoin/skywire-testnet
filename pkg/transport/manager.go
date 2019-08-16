package transport

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/skycoin/skywire/pkg/network"

	"github.com/skycoin/skywire/pkg/routing"

	"github.com/google/uuid"
	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/skycoin/src/util/logging"
)

// ManagerConfig configures a Manager.
type ManagerConfig struct {
	PubKey          cipher.PubKey
	SecKey          cipher.SecKey
	DefaultNodes    []cipher.PubKey // Nodes to automatically connect to
	Networks        []string        // Networks to use.
	DiscoveryClient DiscoveryClient
	LogStore        LogStore
}

// Manager manages Transports.
type Manager struct {
	Logger *logging.Logger
	conf   *ManagerConfig
	nets   map[string]struct{}
	tps    map[uuid.UUID]*ManagedTransport
	n      *network.Network

	readCh chan routing.Packet
	mx     sync.RWMutex
	done   chan struct{}
}

// NewManager creates a Manager with the provided configuration and transport factories.
// 'factories' should be ordered by preference.
func NewManager(n *network.Network, config *ManagerConfig) (*Manager, error) {
	nets := make(map[string]struct{})
	for _, n := range config.Networks {
		nets[n] = struct{}{}
	}
	tm := &Manager{
		Logger: logging.MustGetLogger("tp_manager"),
		conf:   config,
		nets:   nets,
		tps:    make(map[uuid.UUID]*ManagedTransport),
		n:      n,
		readCh: make(chan routing.Packet, 20),
		done:   make(chan struct{}),
	}
	return tm, nil
}

// Serve runs listening loop across all registered factories.
func (tm *Manager) Serve(ctx context.Context) error {
	var listeners []*network.Listener
	var wg sync.WaitGroup

	for _, netName := range tm.conf.Networks {
		lis, err := tm.n.Listen(netName, network.TransportPort)
		if err != nil {
			return fmt.Errorf("failed to listen on network '%s' of port '%d': %v",
				netName, network.TransportPort, err)
		}
		tm.Logger.Infof("listening on network: %s", netName)
		listeners = append(listeners, lis)

		wg.Add(1)
		go func(netName string) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case <-tm.done:
					return
				default:
					if err := tm.acceptTransport(ctx, lis); err != nil {
						if strings.Contains(err.Error(), "closed") {
							return
						}
						tm.Logger.Warnf("Failed to accept connection: %s", err)
					}
				}
			}
		}(netName)
	}
	tm.Logger.Info("transport manager is serving.")

	// closing logic
	<-tm.done

	tm.Logger.Info("transport manager is closing.")
	defer tm.Logger.Info("transport manager closed.")

	// Close all listeners.
	for i, lis := range listeners {
		if err := lis.Close(); err != nil {
			tm.Logger.Warnf("listener %d of network '%s' closed with error: %v", i, lis.Network(), err)
		}
	}

	wg.Wait()
	close(tm.readCh)
	return nil
}

func (tm *Manager) initTransports(ctx context.Context) {
	tm.mx.Lock()
	defer tm.mx.Unlock()

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

func (tm *Manager) acceptTransport(ctx context.Context, lis *network.Listener) error {
	conn, err := lis.AcceptConn()
	if err != nil {
		return err
	}
	tm.Logger.Infof("recv transport connection request: type(%s) remote(%s)", lis.Network(), conn.RemotePK())

	tm.mx.Lock()
	defer tm.mx.Unlock()

	if tm.isClosing() {
		return errors.New("transport.Manager is closing. Skipping incoming transport")
	}

	// For transports for purpose(data).

	tpID := tm.tpIDFromPK(conn.RemotePK(), conn.Network())

	mTp, ok := tm.tps[tpID]
	if !ok {
		mTp = NewManagedTransport(tm.n, tm.conf.DiscoveryClient, tm.conf.LogStore, conn.RemotePK(), lis.Network())
		if err := mTp.Accept(ctx, conn); err != nil {
			return err
		}
		go mTp.Serve(tm.readCh, tm.done)
		tm.tps[tpID] = mTp

	} else {
		if err := mTp.Accept(ctx, conn); err != nil {
			return err
		}
	}

	tm.Logger.Infof("accepted tp: type(%s) remote(%s) tpID(%s) new(%v)", lis.Network(), conn.RemotePK(), tpID, !ok)
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

func (tm *Manager) saveTransport(remote cipher.PubKey, netName string) (*ManagedTransport, error) {
	if _, ok := tm.nets[netName]; !ok {
		return nil, errors.New("unknown transport type")
	}

	tpID := tm.tpIDFromPK(remote, netName)

	tp, ok := tm.tps[tpID]
	if ok {
		return tp, nil
	}

	mTp := NewManagedTransport(tm.n, tm.conf.DiscoveryClient, tm.conf.LogStore, remote, netName)
	go mTp.Serve(tm.readCh, tm.done)
	tm.tps[tpID] = mTp

	tm.Logger.Infof("saved transport: remote(%s) type(%s) tpID(%s)", remote, netName, tpID)
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
	STATE
*/

// Networks returns all the network types contained within the TransportManager.
func (tm *Manager) Networks() []string {
	return tm.conf.Networks
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

	i, statuses := 0, make([]*Status, len(tm.tps))
	for _, tr := range tm.tps {
		tr.close()
		statuses[i] = &Status{ID: tr.Entry.ID, IsUp: false}
		i++
	}
	if _, err := tm.conf.DiscoveryClient.UpdateStatuses(context.Background(), statuses...); err != nil {
		tm.Logger.Warnf("failed to update transport statuses: %v", err)
	}
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
