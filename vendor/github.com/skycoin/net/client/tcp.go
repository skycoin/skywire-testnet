package client

import (
	"github.com/skycoin/net/conn"
	"net"
	"time"
)

type ClientTCPConn struct {
	conn.TCPConn
}

func NewClientTCPConn(c net.Conn) *ClientTCPConn {
	return &ClientTCPConn{conn.TCPConn{TcpConn: c, ConnCommonFields: conn.NewConnCommonFileds()}}
}

func (c *ClientTCPConn) WriteLoop() (err error) {
	ticker := time.NewTicker(time.Second * TICK_PERIOD)
	defer func() {
		ticker.Stop()
		if err != nil {
			c.SetStatusToError(err)
		}
	}()
	for {
		select {
		case <-ticker.C:
			c.CTXLogger.Debug("ping out")
			err := c.Ping()
			if err != nil {
				return err
			}
		case m, ok := <-c.Out:
			if !ok {
				c.CTXLogger.Debug("conn closed")
				return nil
			}
			c.CTXLogger.Debugf("msg Out %x", m)
			err := c.Write(m)
			if err != nil {
				c.CTXLogger.Debugf("write msg is failed %v", err)
				return err
			}
		}
	}
}
