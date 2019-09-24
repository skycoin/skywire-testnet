package network

import (
	"context"
	"net"

	"github.com/skycoin/dmsg"
)

// DMSGNetworker implements `Networker` for dmsg network.
type DMSGNetworker struct {
	dmsgC *dmsg.Client
}

// NewDMSGNetworker constructs new `DMSGNetworker`.
func NewDMSGNetworker(dmsgC *dmsg.Client) Networker {
	return &DMSGNetworker{
		dmsgC: dmsgC,
	}
}

// Dial dials remote `addr` via dmsg network.
func (n *DMSGNetworker) Dial(addr Addr) (net.Conn, error) {
	return n.DialContext(context.Background(), addr)
}

// DialContext dials remote `addr` via dmsg network with context.
func (n *DMSGNetworker) DialContext(ctx context.Context, addr Addr) (net.Conn, error) {
	tp, err := n.dmsgC.Dial(ctx, addr.PubKey, uint16(addr.Port))
	if err != nil {
		return nil, err
	}

	return WrapConn(tp)
}

// Listen starts listening on local `addr` in the dmsg network.
func (n *DMSGNetworker) Listen(addr Addr) (net.Listener, error) {
	return n.ListenContext(context.Background(), addr)
}

// ListenContext starts listening on local `addr` in the dmsg network with context.
func (n *DMSGNetworker) ListenContext(ctx context.Context, addr Addr) (net.Listener, error) {
	return n.dmsgC.Listen(uint16(addr.Port))
}
