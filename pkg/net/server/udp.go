package server

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"net"
	"time"

	"github.com/skycoin/skywire/pkg/net/conn"
	"github.com/skycoin/skywire/pkg/net/msg"
)

type ServerUDPConn struct {
	conn.UDPConn
}

func NewServerUDPConn(c *net.UDPConn) *ServerUDPConn {
	return &ServerUDPConn{
		UDPConn: conn.UDPConn{
			UdpConn:          c,
			ConnCommonFields: conn.NewConnCommonFileds(),
			UnsharedUdpConn:  true,
		},
	}
}

func (c *ServerUDPConn) ReadLoop(fn func(c *net.UDPConn, addr *net.UDPAddr) *conn.UDPConn) (err error) {
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
	var lst = time.Time{}
	var rt = time.Time{}
	var at = time.Time{}
	var nt = time.Time{}
	maxBuf := make([]byte, conn.MTU)
	for {
		if conn.DEV {
			if !lst.IsZero() {
				c.GetContextLogger().Debugf("process udp d %s", time.Now().Sub(lst))
			}
			rt = time.Now()
		}
		n, addr, err := c.UdpConn.ReadFromUDP(maxBuf)
		if conn.DEV {
			c.GetContextLogger().Debugf("process read udp d %s", time.Now().Sub(rt))
			lst = time.Now()
		}
		if err != nil {
			if e, ok := err.(net.Error); ok {
				if e.Timeout() {
					cc := fn(c.UdpConn, addr)
					cc.GetContextLogger().Debug("close in")
					close(cc.In)
					continue
				}
			}
			return err
		}
		c.AddReceivedBytes(n)
		pkg := maxBuf[:n]
		cc := fn(c.UdpConn, addr)
		if cc.IsClosed() {
			c.GetContextLogger().Infof("udp server conn closed")
			continue
		}
		m := pkg[msg.PKG_HEADER_SIZE:]
		checksum := binary.BigEndian.Uint32(pkg[msg.PKG_CRC32_BEGIN:])
		if checksum != crc32.ChecksumIEEE(m) {
			c.GetContextLogger().Infof("checksum !=")
			continue
		}

		t := m[msg.MSG_TYPE_BEGIN]
		switch t {
		case msg.TYPE_ACK:
			if conn.DEV {
				at = time.Now()
			}
			wrapForClient(cc, func() error {
				return cc.RecvAck(m)
			})
			if conn.DEV {
				c.GetContextLogger().Debugf("process ack d %s", time.Now().Sub(at))
			}
		case msg.TYPE_PONG:
		case msg.TYPE_PING:
			wrapForClient(cc, func() error {
				m[msg.PING_MSG_TYPE_BEGIN] = msg.TYPE_PONG
				checksum := crc32.ChecksumIEEE(m)
				binary.BigEndian.PutUint32(pkg[msg.PKG_CRC32_BEGIN:], checksum)
				cc.GetContextLogger().Debugf("pong")
				return cc.WriteExt(pkg)
			})
		case msg.TYPE_NORMAL, msg.TYPE_FEC, msg.TYPE_SYN:
			if conn.DEV {
				nt = time.Now()
			}
			wrapForClient(cc, func() error {
				return cc.Process(t, m)
			})
			if conn.DEV {
				c.GetContextLogger().Debugf("process normal d %s", time.Now().Sub(nt))
			}
		case msg.TYPE_FIN:
			wrapForClient(cc, func() error {
				cc.GetContextLogger().Debug("process fin")
				return conn.ErrFin
			})
		default:
			cc.GetContextLogger().Debugf("not implemented msg type %d", t)
			cc.SetStatusToError(fmt.Errorf("not implemented msg type %d", t))
			cc.Close()
			continue
		}

		cc.UpdateLastTime()
	}
}

func wrapForClient(cc *conn.UDPConn, fn func() error) {
	var err error
	defer func() {
		if !conn.DEV {
			if e := recover(); e != nil {
				cc.GetContextLogger().Debug(e)
				err = fmt.Errorf("wrapForClient panic err:%v", e)
			}
		}
		if err != nil {
			cc.SetStatusToError(err)
			cc.Close()
		}
	}()
	err = fn()
}
