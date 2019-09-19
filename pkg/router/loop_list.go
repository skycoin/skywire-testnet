package router

import (
	"sync"

	"github.com/google/uuid"

	"github.com/SkycoinProject/skywire-mainnet/pkg/routing"
)

type loop struct {
	trID    uuid.UUID
	routeID routing.RouteID
}

type loopList struct {
	sync.Mutex

	loops map[routing.Addr]*loop // key: remote address (pk+port), value: forwarding transport and route ID.
}

func newLoopList() *loopList {
	return &loopList{loops: make(map[routing.Addr]*loop)}
}

func (ll *loopList) get(addr routing.Addr) *loop {
	ll.Lock()
	l := ll.loops[addr]
	ll.Unlock()

	return l
}

func (ll *loopList) set(addr routing.Addr, l *loop) {
	ll.Lock()
	ll.loops[addr] = l
	ll.Unlock()
}

func (ll *loopList) remove(addr routing.Addr) {
	ll.Lock()
	delete(ll.loops, addr)
	ll.Unlock()
}

func (ll *loopList) dropAll() []routing.Addr {
	ll.Lock()
	r := make([]routing.Addr, 0, len(ll.loops))
	for addr := range ll.loops {
		r = append(r, addr)
	}
	ll.Unlock()
	return r
}
