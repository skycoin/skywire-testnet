package router

import (
	"github.com/skycoin/skywire/pkg/app"
	"sync"

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

	loops map[app.LoopAddr]*loop // key: remote addr (pk+port), value: forwarding transport and route ID.
}

func newLoopList() *loopList {
	return &loopList{loops: make(map[app.LoopAddr]*loop)}
}

func (ll *loopList) get(addr *app.LoopAddr) *loop {
	ll.Lock()
	l := ll.loops[*addr]
	ll.Unlock()

	return l
}

func (ll *loopList) set(addr *app.LoopAddr, l *loop) {
	ll.Lock()
	ll.loops[*addr] = l
	ll.Unlock()
}

func (ll *loopList) remove(addr *app.LoopAddr) {
	ll.Lock()
	delete(ll.loops, *addr)
	ll.Unlock()
}

func (ll *loopList) dropAll() []app.LoopAddr {
	r := make([]app.LoopAddr, 0)
	ll.Lock()
	for addr := range ll.loops {
		r = append(r, addr)
	}
	ll.Unlock()
	return r
}
