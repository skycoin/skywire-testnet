package router

import (
	"math"
	"sync"

	"github.com/skycoin/skywire/internal/appnet"
)

type portBind struct {
	conn  *appnet.Protocol
	loops *loopList
}

type portList struct {
	sync.Mutex

	minPort uint16
	ports   map[uint16]*portBind
}

func newPortList(minPort uint16) *portList {
	return &portList{minPort: minPort, ports: make(map[uint16]*portBind, minPort)}
}

func (pl *portList) all() map[uint16]*portBind {
	r := make(map[uint16]*portBind)
	pl.Lock()
	for port, bind := range pl.ports {
		r[port] = bind
	}
	pl.Unlock()

	return r
}

func (pl *portList) add(b *portBind) uint16 {
	pl.Lock()
	defer pl.Unlock()

	for i := uint16(pl.minPort); i < math.MaxUint16; i++ {
		if pl.ports[i] == nil {
			pl.ports[i] = b
			return i
		}
	}

	panic("no free ports")
}

func (pl *portList) set(port uint16, b *portBind) {
	pl.Lock()
	pl.ports[port] = b
	pl.Unlock()
}

func (pl *portList) get(port uint16) *portBind {
	pl.Lock()
	l := pl.ports[port]
	pl.Unlock()

	return l
}

func (pl *portList) remove(port uint16) *portBind {
	pl.Lock()
	b := pl.ports[port]
	delete(pl.ports, port)
	pl.Unlock()

	return b
}
