package stcp

import (
	"context"
	"sync"
)

const (
	PorterMinEphemeral = uint16(49152)
)

type Porter struct {
	eph    uint16 // current ephemeral value
	minEph uint16 // minimal ephemeral port value
	ports  map[uint16]struct{}
	mx     sync.Mutex
}

func newPorter(minEph uint16) *Porter {
	ports := make(map[uint16]struct{})
	ports[0] = struct{}{} // port 0 is invalid

	return &Porter{
		eph:    minEph,
		minEph: minEph,
		ports:  ports,
	}
}

func (p *Porter) Reserve(port uint16) (bool, func()) {
	p.mx.Lock()
	defer p.mx.Unlock()

	if _, ok := p.ports[port]; ok {
		return false, nil
	}
	p.ports[port] = struct{}{}
	return true, p.portFreer(port)
}

func (p *Porter) ReserveEphemeral(ctx context.Context) (uint16, func(), error) {
	p.mx.Lock()
	defer p.mx.Unlock()

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
		return p.eph, p.portFreer(p.eph), nil
	}
}

func (p *Porter) portFreer(port uint16) func() {
	once := new(sync.Once)
	return func() {
		once.Do(func() {
			p.mx.Lock()
			delete(p.ports, port)
			p.mx.Unlock()
		})
	}
}
