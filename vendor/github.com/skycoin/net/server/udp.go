package server

import (
	"encoding/binary"
	"fmt"
	"github.com/skycoin/net/conn"
	"github.com/skycoin/net/msg"
	"hash/crc32"
	"net"
	"time"
)

type ServerUDPConn struct {
	conn.UDPConn
}

func NewServerUDPConn(c *net.UDPConn) *ServerUDPConn {
	return &ServerUDPConn{
		UDPConn: conn.UDPConn{
			UdpConn:          c,
			ConnCommonFields: conn.NewConnCommonFileds(),
		},
	}
}

func (c *ServerUDPConn) ReadLoop(fn func(c *net.UDPConn, addr *net.UDPAddr) *conn.UDPConn) (err error) {
	defer func() {
		if e := recover(); e != nil {
			c.GetContextLogger().Debug(e)
			err = fmt.Errorf("readloop panic err:%v", e)
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
	for {
		maxBuf := make([]byte, conn.MAX_UDP_PACKAGE_SIZE)
		rt = time.Now()
		n, addr, err := c.UdpConn.ReadFromUDP(maxBuf)
		c.GetContextLogger().Debugf("process read udp d %s", time.Now().Sub(rt))
		if !lst.IsZero() {
			c.GetContextLogger().Debugf("read udp d %s", time.Now().Sub(lst))
		}
		lst = time.Now()
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
		maxBuf = maxBuf[:n]
		m := maxBuf[msg.PKG_HEADER_SIZE:]
		checksum := binary.BigEndian.Uint32(maxBuf[msg.PKG_CRC32_BEGIN:])
		if checksum != crc32.ChecksumIEEE(m) {
			c.GetContextLogger().Infof("checksum !=")
			continue
		}
		cc := fn(c.UdpConn, addr)

		t := m[msg.MSG_TYPE_BEGIN]
		switch t {
		case msg.TYPE_ACK:
			at = time.Now()
			func() {
				var err error
				defer func() {
					if e := recover(); e != nil {
						cc.GetContextLogger().Debug(e)
						err = fmt.Errorf("readloop panic err:%v", e)
					}
					if err != nil {
						cc.SetStatusToError(err)
						cc.Close()
					}
				}()
				err = cc.RecvAck(m)
			}()
			c.GetContextLogger().Debugf("process ack d %s", time.Now().Sub(at))
		case msg.TYPE_PONG:
		case msg.TYPE_PING:
			func() {
				var err error
				defer func() {
					if e := recover(); e != nil {
						cc.GetContextLogger().Debug(e)
						err = fmt.Errorf("readloop panic err:%v", e)
					}
					if err != nil {
						cc.SetStatusToError(err)
						cc.Close()
					}
				}()
				m[msg.PING_MSG_TYPE_BEGIN] = msg.TYPE_PONG
				checksum := crc32.ChecksumIEEE(m)
				binary.BigEndian.PutUint32(maxBuf[msg.PKG_CRC32_BEGIN:], checksum)
				err = cc.WriteBytes(maxBuf)
				if err != nil {
					return
				}
				cc.GetContextLogger().Debugf("pong")
			}()
		case msg.TYPE_NORMAL:
			nt = time.Now()
			func() {
				var err error
				defer func() {
					if e := recover(); e != nil {
						cc.GetContextLogger().Debug(e)
						err = fmt.Errorf("readloop panic err:%v", e)
					}
					if err != nil {
						cc.SetStatusToError(err)
						cc.Close()
					}
				}()
				seq := binary.BigEndian.Uint32(m[msg.MSG_SEQ_BEGIN:msg.MSG_SEQ_END])

				ok, ms := cc.Push(seq, m[msg.MSG_HEADER_END:])
				err = cc.Ack(seq)
				if err != nil {
					return
				}
				if ok {
					for _, m := range ms {
						cc.GetContextLogger().Debugf("msg in")
						cc.In <- m
						cc.GetContextLogger().Debugf("msg out")
					}
				}
			}()
			c.GetContextLogger().Debugf("process normal d %s", time.Now().Sub(nt))
		default:
			cc.GetContextLogger().Debugf("not implemented msg type %d", t)
			cc.SetStatusToError(fmt.Errorf("not implemented msg type %d", t))
			cc.Close()
			continue
		}

		cc.UpdateLastTime()
	}
}

func (c *ServerUDPConn) Close() {
	c.FieldsMutex.RLock()
	if c.UdpConn != nil {
		c.UdpConn.Close()
	}
	c.FieldsMutex.RUnlock()
	c.ConnCommonFields.Close()
}
