package app2

import (
	"net"

	"github.com/skycoin/skywire/pkg/routing"
)

// clientConn serves as a wrapper for `net.Conn` being returned to the
// app client side from `Accept` func
type clientConn struct {
	remote routing.Addr
	local  routing.Addr
	net.Conn
}

func (c *clientConn) RemoteAddr() net.Addr {
	return c.remote
}

func (c *clientConn) LocalAddr() net.Addr {
	return c.local
}
