package router

import (
	"math"
	"sync"

	"github.com/skycoin/skywire/pkg/app"
	"github.com/skycoin/skywire/pkg/routing"
)

type portBind struct {
	conn  *app.Protocol
	loops *loopList
}

type portList struct {
	sync.Mutex

	minPort routing.Port
	ports   map[routing.Port]*portBind
}

func newPortList(minPort routing.Port) *portList {
	return &portList{minPort: minPort, ports: make(map[routing.Port]*portBind, minPort)}
}

func (pl *portList) all() map[routing.Port]*portBind {
	r := make(map[routing.Port]*portBind)
	pl.Lock()
	for port, bind := range pl.ports {
		r[port] = bind
	}
	pl.Unlock()

	return r
}

func (pl *portList) add(b *portBind) routing.Port {
	pl.Lock()
	defer pl.Unlock()

	for i := pl.minPort; i < math.MaxUint16; i++ {
		if pl.ports[i] == nil {
			pl.ports[i] = b
			return i
		}
	}

	panic("no free ports")
}

func (pl *portList) set(port routing.Port, b *portBind) {
	pl.Lock()
	pl.ports[port] = b
	pl.Unlock()
}

func (pl *portList) get(port routing.Port) *portBind {
	pl.Lock()
	l := pl.ports[port]
	pl.Unlock()

	return l
}

func (pl *portList) remove(port routing.Port) *portBind {
	pl.Lock()
	b := pl.ports[port]
	delete(pl.ports, port)
	pl.Unlock()

	return b
}
