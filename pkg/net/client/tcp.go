package client

import (
	"net"
	"time"

	"github.com/skycoin/skywire/pkg/net/conn"
)

type ClientTCPConn struct {
	conn.TCPConn
}

func NewClientTCPConn(c net.Conn) *ClientTCPConn {
	return &ClientTCPConn{
		TCPConn: conn.TCPConn{
			TcpConn:          c,
			ConnCommonFields: conn.NewConnCommonFileds(),
		},
	}
}

func (c *ClientTCPConn) WriteLoop() (err error) {
	ticker := time.NewTicker(time.Second * conn.TCP_PING_TICK_PERIOD)
	defer func() {
		ticker.Stop()
		if err != nil {
			c.SetStatusToError(err)
		}
	}()
	for {
		select {
		case <-ticker.C:
			err := c.Ping()
			if err != nil {
				return err
			}
		case m, ok := <-c.Out:
			if !ok {
				c.GetContextLogger().Debug("conn closed")
				return nil
			}
			//c.GetContextLogger().Debugf("msg Out %x", m)
			err := c.Write(m)
			if err != nil {
				c.GetContextLogger().Debugf("write msg is failed %v", err)
				return err
			}
		}
	}
}
