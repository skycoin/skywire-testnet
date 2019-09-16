package netutil

import (
	"context"
	"sync"
)

const (
	// PorterMinEphemeral is the default minimum ephemeral port.
	PorterMinEphemeral = uint16(49152)
)

// Porter reserves ports.
type Porter struct {
	sync.RWMutex
	eph    uint16 // current ephemeral value
	minEph uint16 // minimal ephemeral port value
	ports  map[uint16]interface{}
}

// NewPorter creates a new Porter with a given minimum ephemeral port value.
func NewPorter(minEph uint16) *Porter {
	ports := make(map[uint16]interface{})
	ports[0] = struct{}{} // port 0 is invalid

	return &Porter{
		eph:    minEph,
		minEph: minEph,
		ports:  ports,
	}
}

// Reserve a given port.
// It returns a boolean informing whether the port is reserved, and a function to clear the reservation.
func (p *Porter) Reserve(port uint16, v interface{}) (bool, func()) {
	p.Lock()
	defer p.Unlock()

	if _, ok := p.ports[port]; ok {
		return false, nil
	}
	p.ports[port] = v
	return true, p.makePortFreer(port)
}

// ReserveEphemeral reserves a new ephemeral port.
// It returns the reserved ephemeral port, a function to clear the reservation and an error (if any).
func (p *Porter) ReserveEphemeral(ctx context.Context, v interface{}) (uint16, func(), error) {
	p.Lock()
	defer p.Unlock()

	for {
		p.eph++
		if p.eph < p.minEph {
			p.eph = p.minEph
		}
		if _, ok := p.ports[p.eph]; ok {
			select {
			case <-ctx.Done():
				return 0, nil, ctx.Err()
			default:
				continue
			}
		}
		p.ports[p.eph] = v
		return p.eph, p.makePortFreer(p.eph), nil
	}
}

// PortValue returns the value stored under a given port.
func (p *Porter) PortValue(port uint16) (interface{}, bool) {
	p.RLock()
	defer p.RUnlock()

	v, ok := p.ports[port]
	return v, ok
}

// RangePortValues ranges all ports that are currently reserved.
func (p *Porter) RangePortValues(fn func(port uint16, v interface{}) (next bool)) {
	p.RLock()
	defer p.RUnlock()

	for port, v := range p.ports {
		if next := fn(port, v); !next {
			return
		}
	}
}

// This returns a function that frees a given port.
// It is ensured that the function's action is only performed once.
func (p *Porter) makePortFreer(port uint16) func() {
	once := new(sync.Once)
	return func() {
		once.Do(func() {
			p.Lock()
			delete(p.ports, port)
			p.Unlock()
		})
	}
}
