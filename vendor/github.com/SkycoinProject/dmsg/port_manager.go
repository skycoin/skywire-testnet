package dmsg

import (
	"math/rand"
	"sync"
	"time"

	"github.com/SkycoinProject/dmsg/cipher"
)

const (
	firstEphemeralPort = 49152
	lastEphemeralPort  = 65535
)

// PortManager manages ports of nodes.
type PortManager struct {
	mu        sync.RWMutex
	rand      *rand.Rand
	listeners map[uint16]*Listener
}

func newPortManager() *PortManager {
	return &PortManager{
		rand:      rand.New(rand.NewSource(time.Now().UnixNano())),
		listeners: make(map[uint16]*Listener),
	}
}

// Listener returns a listener assigned to a given port.
func (pm *PortManager) Listener(port uint16) (*Listener, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	l, ok := pm.listeners[port]
	return l, ok
}

// NewListener assigns listener to port if port is available.
func (pm *PortManager) NewListener(pk cipher.PubKey, port uint16) (*Listener, bool) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if _, ok := pm.listeners[port]; ok {
		return nil, false
	}
	l := newListener(pk, port)
	pm.listeners[port] = l
	return l, true
}

// RemoveListener removes listener assigned to port.
func (pm *PortManager) RemoveListener(port uint16) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	delete(pm.listeners, port)
}

// NextEmptyEphemeralPort returns next random ephemeral port.
// It has a value between firstEphemeralPort and lastEphemeralPort.
func (pm *PortManager) NextEmptyEphemeralPort() uint16 {
	for {
		port := pm.randomEphemeralPort()
		if _, ok := pm.Listener(port); !ok {
			return port
		}
	}
}

func (pm *PortManager) randomEphemeralPort() uint16 {
	return uint16(firstEphemeralPort + pm.rand.Intn(lastEphemeralPort-firstEphemeralPort))
}
