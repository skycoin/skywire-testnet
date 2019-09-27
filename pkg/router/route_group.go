package router

import (
	"bytes"
	"sync"

	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/transport"
)

// RouteGroup should implement 'io.ReadWriteCloser'.
type RouteGroup struct {
	mu sync.RWMutex

	desc routing.RouteDescriptor // describes the route group
	fwd  []routing.Rule          // forward rules (for writing)
	rvs  []routing.Rule          // reverse rules (for reading)

	// The following fields are used for writing:
	// - fwd/tps should have the same number of elements.
	// - the corresponding element of tps should have tpID of the corresponding rule in fwd.
	// - rg.fwd references 'ForwardRule' rules for writes.

	// 'tps' is transports used for writing/forward rules.
	// It should have the same number of elements as 'fwd'
	// where each element corresponds with the adjacent element in 'fwd'.
	tps []*transport.ManagedTransport

	// 'readCh' reads in incoming packets of this route group.
	// - Router should serve call '(*transport.Manager).ReadPacket' in a loop,
	//      and push to the appropriate '(RouteGroup).readCh'.
	readCh  chan routing.Packet // push reads from Router
	readBuf bytes.Buffer        // for read overflow
}
