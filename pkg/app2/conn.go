package app2

import (
	"net"
	"time"

	"github.com/pkg/errors"
	"github.com/skycoin/skywire/pkg/routing"
)

// Conn is a connection from app client to the server.
type Conn struct {
	id            uint16
	rpc           ServerRPCClient
	local         routing.Addr
	remote        routing.Addr
	freeLocalPort func()
}

func (c *Conn) Read(b []byte) (int, error) {
	return c.rpc.Read(c.id, b)
}

func (c *Conn) Write(b []byte) (int, error) {
	return c.rpc.Write(c.id, b)
}

func (c *Conn) Close() error {
	defer c.freeLocalPort()

	return c.rpc.CloseConn(c.id)
}

func (c *Conn) LocalAddr() net.Addr {
	return c.local
}

func (c *Conn) RemoteAddr() net.Addr {
	return c.remote
}

func (c *Conn) SetDeadline(t time.Time) error {
	return errors.New("method not implemented")
}

func (c *Conn) SetReadDeadline(t time.Time) error {
	return errors.New("method not implemented")
}

func (c *Conn) SetWriteDeadline(t time.Time) error {
	return errors.New("method not implemented")
}
