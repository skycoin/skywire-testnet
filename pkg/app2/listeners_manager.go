package app2

import (
	"net"
	"sync"

	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/pkg/errors"
	"github.com/skycoin/skywire/pkg/routing"
)

var (
	ErrPortAlreadyBound               = errors.New("port is already bound")
	ErrNoListenerOnPort               = errors.New("no listener on port")
	ErrListenersManagerAlreadyServing = errors.New("listeners manager already serving")
)

type listenersManager struct {
	listeners map[routing.Port]*Listener
	mx        sync.RWMutex
	isServing int32
}

func newListenersManager() *listenersManager {
	return &listenersManager{
		listeners: make(map[routing.Port]*Listener),
	}
}

func (lm *listenersManager) portIsBound(port routing.Port) bool {
	lm.mx.RLock()
	_, ok := lm.listeners[port]
	lm.mx.RUnlock()
	return ok
}

func (lm *listenersManager) add(addr routing.Addr, stopListening func(port routing.Port) error, logger *logging.Logger) (*Listener, error) {
	lm.mx.Lock()
	if _, ok := lm.listeners[addr.Port]; ok {
		lm.mx.Unlock()
		return nil, ErrPortAlreadyBound
	}
	l := NewListener(addr, lm, stopListening, logger)
	lm.listeners[addr.Port] = l
	lm.mx.Unlock()
	return l, nil
}

func (lm *listenersManager) remove(port routing.Port) error {
	lm.mx.Lock()
	if _, ok := lm.listeners[port]; !ok {
		lm.mx.Unlock()
		return ErrNoListenerOnPort
	}
	delete(lm.listeners, port)
	lm.mx.Unlock()
	return nil
}

func (lm *listenersManager) addConn(port routing.Port, conn net.Conn) error {
	lm.mx.RLock()
	if _, ok := lm.listeners[port]; !ok {
		lm.mx.RUnlock()
		return ErrNoListenerOnPort
	}
	lm.listeners[port].addConn(conn)
	lm.mx.RUnlock()
	return nil
}
