package app2

import (
	"net"

	"github.com/skycoin/skywire/pkg/app2/network"
)

// Listener is a listener for app server connections.
// Implements `net.Listener`.
type Listener struct {
	id       uint16
	rpc      RPCClient
	addr     network.Addr
	freePort func()
}

func (l *Listener) Accept() (net.Conn, error) {
	connID, remote, err := l.rpc.Accept(l.id)
	if err != nil {
		return nil, err
	}

	conn := &Conn{
		id:     connID,
		rpc:    l.rpc,
		local:  l.addr,
		remote: remote,
	}

	return conn, nil
}

func (l *Listener) Close() error {
	defer l.freePort()

	return l.rpc.CloseListener(l.id)
}

func (l *Listener) Addr() net.Addr {
	return l.addr
}
