package app2

import (
	"net"

	"github.com/skycoin/skywire/pkg/routing"
)

type Conn struct {
	id     uint16
	rpc    ConnRPCClient
	local  routing.Addr
	remote routing.Addr
}

func (c *Conn) Read(b []byte) (int, error) {
	return c.rpc.Read(c.id, b)
}

func (c *Conn) Write(b []byte) (int, error) {
	return c.rpc.Write(c.id, b)
}

func (c *Conn) Close() error {
	return c.rpc.CloseConn(c.id)
}

func (c *Conn) LocalAddr() net.Addr {
	return c.local
}

func (c *Conn) RemoteAddr() net.Addr {
	return c.remote
}
