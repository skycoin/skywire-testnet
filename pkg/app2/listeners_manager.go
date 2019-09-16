package app2

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"
	"github.com/skycoin/dmsg"
)

// listenersManager manages listeners within the app server.
type listenersManager struct {
	listeners map[uint16]*dmsg.Listener
	mx        sync.RWMutex
	lstID     uint16
}

// newListenersManager constructs new `listenersManager`.
func newListenersManager() *listenersManager {
	return &listenersManager{
		listeners: make(map[uint16]*dmsg.Listener),
	}
}

// `nextID` reserves slot for the next listener and returns its id.
func (m *listenersManager) nextID() (*uint16, error) {
	m.mx.Lock()

	lisID := m.lstID + 1
	for ; lisID < m.lstID; lisID++ {
		if _, ok := m.listeners[lisID]; !ok {
			break
		}
	}

	if lisID == m.lstID {
		m.mx.Unlock()
		return nil, errors.New("no more available listeners")
	}

	m.listeners[lisID] = nil
	m.lstID = lisID

	m.mx.Unlock()
	return &lisID, nil
}

// getAndRemove removes listener specified by `lisID` from the manager instance and
// returns it.
func (m *listenersManager) getAndRemove(lisID uint16) (*dmsg.Listener, error) {
	m.mx.Lock()
	lis, ok := m.listeners[lisID]
	if !ok {
		m.mx.Unlock()
		return nil, fmt.Errorf("no listener with id %d", lisID)
	}

	if lis == nil {
		m.mx.Unlock()
		return nil, fmt.Errorf("listener with id %d is not set", lisID)
	}

	delete(m.listeners, lisID)

	m.mx.Unlock()
	return lis, nil
}

// set sets `lis` associated with `lisID`.
func (m *listenersManager) set(lisID uint16, lis *dmsg.Listener) error {
	m.mx.Lock()

	if l, ok := m.listeners[lisID]; ok && l != nil {
		m.mx.Unlock()
		return errors.New("listener already exists")
	}

	m.listeners[lisID] = lis

	m.mx.Unlock()
	return nil
}

// get gets the listener associated with the `lisID`.
func (m *listenersManager) get(lisID uint16) (*dmsg.Listener, bool) {
	m.mx.RLock()
	lis, ok := m.listeners[lisID]
	m.mx.RUnlock()
	return lis, ok
}
