package network

import (
	"net"
	"time"

	"github.com/skycoin/skywire/pkg/routing"

	"github.com/skycoin/dmsg"
)

type DMSGConn struct {
	tp *dmsg.Transport
}

func (c *DMSGConn) Read(b []byte) (n int, err error) {
	return c.tp.Read(b)
}

func (c *DMSGConn) Write(b []byte) (n int, err error) {
	return c.tp.Write(b)
}

func (c *DMSGConn) Close() error {
	return c.tp.Close()
}

func (c *DMSGConn) LocalAddr() net.Addr {
	dmsgAddr, ok := c.tp.LocalAddr().(dmsg.Addr)
	if !ok {
		return c.tp.LocalAddr()
	}

	return Addr{
		Net:    TypeDMSG,
		PubKey: dmsgAddr.PK,
		Port:   routing.Port(dmsgAddr.Port),
	}
}

func (c *DMSGConn) RemoteAddr() net.Addr {
	dmsgAddr, ok := c.tp.RemoteAddr().(dmsg.Addr)
	if !ok {
		return c.tp.RemoteAddr()
	}

	return Addr{
		Net:    TypeDMSG,
		PubKey: dmsgAddr.PK,
		Port:   routing.Port(dmsgAddr.Port),
	}
}

func (c *DMSGConn) SetDeadline(t time.Time) error {
	return c.tp.SetDeadline(t)
}

func (c *DMSGConn) SetReadDeadline(t time.Time) error {
	return c.tp.SetReadDeadline(t)
}

func (c *DMSGConn) SetWriteDeadline(t time.Time) error {
	return c.tp.SetWriteDeadline(t)
}
