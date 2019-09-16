package app2

import (
	"net"

	"github.com/skycoin/skywire/pkg/routing"
)

// Listener is a listener for app server connections.
type Listener struct {
	id   uint16
	rpc  ServerRPCClient
	addr routing.Addr
}

func (l *Listener) Accept() (net.Conn, error) {
	connID, err := l.rpc.Accept(l.id)
	if err != nil {
		return nil, err
	}

	conn := &Conn{
		id:    connID,
		rpc:   l.rpc,
		local: l.addr,
		// TODO: probably pass with response
		remote: routing.Addr{},
	}

	return conn, nil
}

// TODO: should unblock all called `Accept`s with errors
func (l *Listener) Close() error {
	return l.rpc.CloseListener(l.id)
}

func (l *Listener) Addr() net.Addr {
	return l.addr
}
