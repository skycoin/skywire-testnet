package app2

import (
	"fmt"
	"net"
	"sync"

	"github.com/pkg/errors"
)

type connsManager struct {
	conns map[uint16]net.Conn
	mx    sync.RWMutex
	lstID uint16
}

func newConnsManager() *connsManager {
	return &connsManager{
		conns: make(map[uint16]net.Conn),
	}
}

func (m *connsManager) nextID() (*uint16, error) {
	m.mx.Lock()

	connID := m.lstID + 1
	for ; connID < m.lstID; connID++ {
		if _, ok := m.conns[connID]; !ok {
			break
		}
	}

	if connID == m.lstID {
		m.mx.Unlock()
		return nil, errors.New("no more available conns")
	}

	m.conns[connID] = nil
	m.lstID = connID

	m.mx.Unlock()
	return &connID, nil
}

func (m *connsManager) getAndRemove(connID uint16) (net.Conn, error) {
	m.mx.Lock()
	conn, ok := m.conns[connID]
	if !ok {
		m.mx.Unlock()
		return nil, fmt.Errorf("no conn with id %d", connID)
	}

	if conn == nil {
		m.mx.Unlock()
		return nil, fmt.Errorf("conn with id %d is not set", connID)
	}

	delete(m.conns, connID)

	m.mx.Unlock()
	return conn, nil
}

func (m *connsManager) set(connID uint16, conn net.Conn) error {
	m.mx.Lock()

	if c, ok := m.conns[connID]; ok && c != nil {
		m.mx.Unlock()
		return errors.New("conn already exists")
	}

	m.conns[connID] = conn

	m.mx.Unlock()
	return nil
}

func (m *connsManager) get(connID uint16) (net.Conn, bool) {
	m.mx.RLock()
	conn, ok := m.conns[connID]
	m.mx.RUnlock()
	return conn, ok
}
