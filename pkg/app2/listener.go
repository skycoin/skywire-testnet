package app2

import (
	"net"

	"github.com/skycoin/skywire/pkg/routing"
)

type Listener struct {
	id   uint16
	rpc  ListenerRPCClient
	addr routing.Addr
}

func (l *Listener) Accept() (*Conn, error) {
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

func (l *Listener) Close() error {
	return l.rpc.CloseListener(l.id)
}

func (l *Listener) Addr() net.Addr {
	return l.addr
}
