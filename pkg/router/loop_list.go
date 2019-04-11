package router

import (
	"sync"

	"github.com/skycoin/skywire/internal/appnet"

	"github.com/google/uuid"

	"github.com/skycoin/skywire/internal/noise"
	"github.com/skycoin/skywire/pkg/routing"
)

type loop struct {
	trID    uuid.UUID
	routeID routing.RouteID
	noise   *noise.Noise
}

type loopList struct {
	sync.Mutex

	loops map[appnet.LoopAddr]*loop // key: remote address (pk+port), value: forwarding transport and route ID.
}

func newLoopList() *loopList {
	return &loopList{loops: make(map[appnet.LoopAddr]*loop)}
}

func (ll *loopList) get(addr *appnet.LoopAddr) *loop {
	ll.Lock()
	l := ll.loops[*addr]
	ll.Unlock()

	return l
}

func (ll *loopList) set(addr *appnet.LoopAddr, l *loop) {
	ll.Lock()
	ll.loops[*addr] = l
	ll.Unlock()
}

func (ll *loopList) remove(addr *appnet.LoopAddr) {
	ll.Lock()
	delete(ll.loops, *addr)
	ll.Unlock()
}

func (ll *loopList) dropAll() []appnet.LoopAddr {
	r := make([]appnet.LoopAddr, 0)
	ll.Lock()
	for addr := range ll.loops {
		r = append(r, addr)
	}
	ll.Unlock()
	return r
}
