package router

import (
	"context"
	"net"
	"sync"

	"github.com/skycoin/skywire/pkg/app"
	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/transport"
)

// MockRouter - test implementation of Router
type MockRouter struct {
	sync.Mutex

	ports []routing.Port

	didStart bool
	DidClose bool

	errChan chan error

	inPacket *app.Packet
	inLoop   routing.Loop
}

var n *MockRouter

// Ports implements PacketRouter.Ports
func (r *MockRouter) Ports() []routing.Port {
	r.Lock()
	p := r.ports
	r.Unlock()
	return p
}

// Serve implements PacketRouter.Serve
func (r *MockRouter) Serve(_ context.Context) error {
	r.didStart = true
	return nil
}

// ServeApp implements PacketRouter.ServeApp
func (r *MockRouter) ServeApp(conn net.Conn, port routing.Port, appConf *app.Config) error {
	r.Lock()
	if r.ports == nil {
		r.ports = []routing.Port{}
	}

	r.ports = append(r.ports, port)
	r.Unlock()

	if r.errChan == nil {
		r.Lock()
		r.errChan = make(chan error)
		r.Unlock()
	}

	return <-r.errChan
}

// Close implements PacketRouter.Close
func (r *MockRouter) Close() error {
	if r == nil {
		return nil
	}
	r.DidClose = true
	r.Lock()
	if r.errChan != nil {
		close(r.errChan)
	}
	r.Unlock()
	return nil
}

// IsSetupTransport implements PacketRouter.IsSetupTransport
func (r *MockRouter) IsSetupTransport(tr *transport.ManagedTransport) bool {
	return false
}

// CloseLoop  implements PacketRouter.CloseLoop
func (r *MockRouter) CloseLoop(conn *app.Protocol, loop routing.Loop) error {
	r.inLoop = loop
	return nil
}

// ForwardAppPacket  implements PacketRouter.ForwardAppPacket
func (r *MockRouter) ForwardAppPacket(conn *app.Protocol, packet *app.Packet) error {
	r.inPacket = packet
	return nil
}

// CreateLoop  implements PacketRouter.CreateLoop
func (r *MockRouter) CreateLoop(conn *app.Protocol, raddr routing.Addr) (laddr routing.Addr, err error) {
	return raddr, nil
}
