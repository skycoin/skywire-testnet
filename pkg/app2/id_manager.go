package app2

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"
)

var (
	errNoMoreAvailableValues = errors.New("no more available values")
)

// idManager manages allows to store and retrieve arbitrary values
// associated with the `uint16` key in a thread-safe manner.
// Provides method to generate key.
type idManager struct {
	values map[uint16]interface{}
	mx     sync.RWMutex
	lstID  uint16
}

// newIDManager constructs new `idManager`.
func newIDManager() *idManager {
	return &idManager{
		values: make(map[uint16]interface{}),
	}
}

// `reserveNextID` reserves next free slot for the value and returns the id for it.
func (m *idManager) reserveNextID() (id *uint16, free func(), err error) {
	m.mx.Lock()

	nxtID := m.lstID + 1
	for ; nxtID != m.lstID; nxtID++ {
		if _, ok := m.values[nxtID]; !ok {
			break
		}
	}

	if nxtID == m.lstID {
		m.mx.Unlock()
		return nil, nil, errNoMoreAvailableValues
	}

	m.values[nxtID] = nil
	m.lstID = nxtID

	m.mx.Unlock()
	return &nxtID, m.constructFreeFunc(nxtID), nil
}

// pop removes value specified by `id` from the idManager instance and
// returns it.
func (m *idManager) pop(id uint16) (interface{}, error) {
	m.mx.Lock()
	v, ok := m.values[id]
	if !ok {
		m.mx.Unlock()
		return nil, fmt.Errorf("no value with id %d", id)
	}

	if v == nil {
		m.mx.Unlock()
		return nil, fmt.Errorf("value with id %d is not set", id)
	}

	delete(m.values, id)

	m.mx.Unlock()
	return v, nil
}

// set sets value `v` associated with `id`.
func (m *idManager) set(id uint16, v interface{}) error {
	m.mx.Lock()

	l, ok := m.values[id]
	if !ok {
		m.mx.Unlock()
		return errors.New("id is not reserved")
	} else {
		if l != nil {
			m.mx.Unlock()
			return errors.New("value already exists")
		}
	}

	m.values[id] = v

	m.mx.Unlock()
	return nil
}

// get gets the value associated with the `id`.
func (m *idManager) get(id uint16) (interface{}, bool) {
	m.mx.RLock()
	lis, ok := m.values[id]
	m.mx.RUnlock()
	if lis == nil {
		return nil, false
	}
	return lis, ok
}

// constructFreeFunc constructs new func responsible for clearing
// a slot with the specified `id`.
func (m *idManager) constructFreeFunc(id uint16) func() {
	return func() {
		m.mx.Lock()
		delete(m.values, id)
		m.mx.Unlock()
	}
}
