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
	uc.UnsharedUdpConn = true
	return &ClientUDPConn{UDPConn: uc}
}

func (c *ClientUDPConn) ReadLoop() (err error) {
	defer func() {
		if !conn.DEV {
			if e := recover(); e != nil {
				c.GetContextLogger().Debug(e)
				err = fmt.Errorf("readloop panic err:%v", e)
			}
		}
		if err != nil {
			c.SetStatusToError(err)
		}
		c.Close()
	}()
	maxBuf := make([]byte, conn.MTU)
	for {
		n, err := c.UdpConn.Read(maxBuf)
		if err != nil {
			return err
		}
		c.AddReceivedBytes(n)
		pkg := maxBuf[:n]
		m := pkg[msg.PKG_HEADER_SIZE:]
		checksum := binary.BigEndian.Uint32(pkg[msg.PKG_CRC32_BEGIN:])
		if checksum != crc32.ChecksumIEEE(m) {
			c.GetContextLogger().Infof("checksum !=")
			continue
		}

		t := m[msg.MSG_TYPE_BEGIN]
		switch t {
		case msg.TYPE_PONG:
		case msg.TYPE_ACK:
			err = c.RecvAck(m)
			if err != nil {
				return err
			}
		case msg.TYPE_NORMAL, msg.TYPE_FEC, msg.TYPE_SYN:
			err = c.Process(t, m)
			if err != nil {
				return err
			}
		case msg.TYPE_FIN:
			err = conn.ErrFin
			break
		default:
			c.GetContextLogger().Debugf("not implemented msg type %d", t)
			return fmt.Errorf("not implemented msg type %d", t)
		}
		c.UpdateLastTime()
	}
}
