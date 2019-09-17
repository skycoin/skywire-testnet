package app2

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"
)

var (
	errNoMoreAvailableValues = errors.New("no more available values")
)

// manager manages allows to store and retrieve arbitrary values
// associated with the `uint16` key in a thread-safe manner.
// Provides method to generate key.
type manager struct {
	values map[uint16]interface{}
	mx     sync.RWMutex
	lstKey uint16
}

// newManager constructs new `manager`.
func newManager() *manager {
	return &manager{
		values: make(map[uint16]interface{}),
	}
}

// `nextKey` reserves next free slot for the value and returns the key for it.
func (m *manager) nextKey() (*uint16, error) {
	m.mx.Lock()

	nxtKey := m.lstKey + 1
	for ; nxtKey != m.lstKey; nxtKey++ {
		if _, ok := m.values[nxtKey]; !ok {
			break
		}
	}

	if nxtKey == m.lstKey {
		m.mx.Unlock()
		return nil, errNoMoreAvailableValues
	}

	m.values[nxtKey] = nil
	m.lstKey = nxtKey

	m.mx.Unlock()
	return &nxtKey, nil
}

// getAndRemove removes value specified by `key` from the manager instance and
// returns it.
func (m *manager) getAndRemove(key uint16) (interface{}, error) {
	m.mx.Lock()
	v, ok := m.values[key]
	if !ok {
		m.mx.Unlock()
		return nil, fmt.Errorf("no value with key %d", key)
	}

	if v == nil {
		m.mx.Unlock()
		return nil, fmt.Errorf("value with key %d is not set", key)
	}

	delete(m.values, key)

	m.mx.Unlock()
	return v, nil
}

// set sets value `v` associated with `key`.
func (m *manager) set(key uint16, v interface{}) error {
	m.mx.Lock()

	l, ok := m.values[key]
	if !ok {
		m.mx.Unlock()
		return errors.New("key is not reserved")
	} else {
		if l != nil {
			m.mx.Unlock()
			return errors.New("value already exists")
		}
	}

	m.values[key] = v

	m.mx.Unlock()
	return nil
}

// get gets the value associated with the `key`.
func (m *manager) get(key uint16) (interface{}, bool) {
	m.mx.RLock()
	lis, ok := m.values[key]
	m.mx.RUnlock()
	return lis, ok
}
