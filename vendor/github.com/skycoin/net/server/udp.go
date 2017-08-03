package server

import (
	"encoding/binary"
	"fmt"
	"net"

	"github.com/skycoin/net/conn"
	"github.com/skycoin/net/msg"
)

type ServerUDPConn struct {
	conn.UDPConn
}

func NewServerUDPConn(c *net.UDPConn) *ServerUDPConn {
	return &ServerUDPConn{UDPConn: conn.UDPConn{UdpConn: c, ConnCommonFields: conn.NewConnCommonFileds()}}
}

func (c *ServerUDPConn) ReadLoop(fn func(c *net.UDPConn, addr *net.UDPAddr) *conn.UDPConn) (err error) {
	defer func() {
		if e := recover(); e != nil {
			c.CTXLogger.Debug(e)
			err = fmt.Errorf("readloop panic err:%v", e)
		}
		if err != nil {
			c.SetStatusToError(err)
		}
		c.Close()
	}()
	maxBuf := make([]byte, conn.MAX_UDP_PACKAGE_SIZE)
	for {
		n, addr, err := c.UdpConn.ReadFromUDP(maxBuf)
		if err != nil {
			if e, ok := err.(net.Error); ok {
				if e.Timeout() {
					cc := fn(c.UdpConn, addr)
					cc.CTXLogger.Debug("close in")
					close(cc.In)
					continue
				}
			}
			return err
		}
		m := maxBuf[:n]
		cc := fn(c.UdpConn, addr)

		t := m[msg.MSG_TYPE_BEGIN]
		switch t {
		case msg.TYPE_ACK:
			seq := binary.BigEndian.Uint32(m[msg.MSG_SEQ_BEGIN:msg.MSG_SEQ_END])
			cc.DelMsg(seq)
			cc.UpdateLastAck(seq)
		case msg.TYPE_PING:
			cc.CTXLogger.Debug("recv ping")
			m[msg.PING_MSG_TYPE_BEGIN] = msg.TYPE_PONG
			err = cc.WriteBytes(m)
			if err != nil {
				return err
			}
		case msg.TYPE_NORMAL:
			seq := binary.BigEndian.Uint32(m[msg.MSG_SEQ_BEGIN:msg.MSG_SEQ_END])
			err = cc.Ack(seq)
			if err != nil {
				return err
			}
			func() {
				defer func() {
					if e := recover(); e != nil {
						cc.CTXLogger.Debug(e)
						err = fmt.Errorf("readloop panic err:%v", e)
					}
					if err != nil {
						cc.SetStatusToError(err)
					}
					cc.Close()
				}()
				cc.CTXLogger.Debugf("c.In <- m.Body %x", m[msg.MSG_HEADER_END:])
				cc.In <- m[msg.MSG_HEADER_END:]
			}()
		default:
			cc.CTXLogger.Debugf("not implemented msg type %d", t)
			return fmt.Errorf("not implemented msg type %d", t)
		}

		cc.UpdateLastTime()
	}
}
