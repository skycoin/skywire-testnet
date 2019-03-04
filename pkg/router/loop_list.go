package router

import (
	"sync"

	"github.com/google/uuid"

	"github.com/skycoin/skywire/internal/noise"
	"github.com/skycoin/skywire/pkg/app"
	"github.com/skycoin/skywire/pkg/routing"
)

type loop struct {
	trID    uuid.UUID
	routeID routing.RouteID
	noise   *noise.Noise
}

type loopList struct {
	sync.Mutex

	loops map[app.Addr]*loop
}

func newLoopList() *loopList {
	return &loopList{loops: make(map[app.Addr]*loop)}
}

func (ll *loopList) get(addr *app.Addr) *loop {
	ll.Lock()
	l := ll.loops[*addr]
	ll.Unlock()

	return l
}

func (ll *loopList) set(addr *app.Addr, l *loop) {
	ll.Lock()
	ll.loops[*addr] = l
	ll.Unlock()
}

func (ll *loopList) remove(addr *app.Addr) {
	ll.Lock()
	delete(ll.loops, *addr)
	ll.Unlock()
}

func (ll *loopList) dropAll() []app.Addr {
	r := make([]app.Addr, 0)
	ll.Lock()
	for addr := range ll.loops {
		r = append(r, addr)
	}
	ll.Unlock()
	return r
}
