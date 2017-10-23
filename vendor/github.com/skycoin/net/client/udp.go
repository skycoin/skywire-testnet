package client

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"net"

	"github.com/skycoin/net/conn"
	"github.com/skycoin/net/msg"
)

type ClientUDPConn struct {
	*conn.UDPConn
}

func NewClientUDPConn(c *net.UDPConn, addr *net.UDPAddr) *ClientUDPConn {
	uc := conn.NewUDPConn(c, addr)
	uc.SendPing = true
	return &ClientUDPConn{UDPConn: uc}
}

func (c *ClientUDPConn) ReadLoop() (err error) {
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
	for {
		maxBuf := make([]byte, conn.MAX_UDP_PACKAGE_SIZE)
		n, err := c.UdpConn.Read(maxBuf)
		if err != nil {
			return err
		}
		c.AddReceivedBytes(n)
		maxBuf = maxBuf[:n]
		m := maxBuf[msg.PKG_HEADER_SIZE:]
		checksum := binary.BigEndian.Uint32(maxBuf[msg.PKG_CRC32_BEGIN:])
		if checksum != crc32.ChecksumIEEE(m) {
			c.GetContextLogger().Infof("checksum !=")
			continue
		}

		t := m[msg.MSG_TYPE_BEGIN]
		switch t {
		case msg.TYPE_PONG:
		case msg.TYPE_ACK:
			seq := binary.BigEndian.Uint32(m[msg.MSG_SEQ_BEGIN:msg.MSG_SEQ_END])
			err = c.DelMsg(seq)
			if err != nil {
				return err
			}
		case msg.TYPE_NORMAL:
			seq := binary.BigEndian.Uint32(m[msg.MSG_SEQ_BEGIN:msg.MSG_SEQ_END])
			err := c.Ack(seq)
			if err != nil {
				return err
			}
			if ok, ms := c.Push(seq, m[msg.MSG_HEADER_END:]); ok {
				for _, m := range ms {
					c.In <- m
				}
			}
		default:
			c.CTXLogger.Debugf("not implemented msg type %d", t)
			return fmt.Errorf("not implemented msg type %d", t)
		}
	}
}

func (c *ClientUDPConn) Close() {
	c.FieldsMutex.RLock()
	if c.UdpConn != nil {
		c.UdpConn.Close()
	}
	c.FieldsMutex.RUnlock()
	c.ConnCommonFields.Close()
}
