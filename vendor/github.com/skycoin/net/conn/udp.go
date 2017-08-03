package conn

import (
	"encoding/binary"
	"net"
	"sync/atomic"
	"time"

	"github.com/skycoin/net/msg"
)

const (
	MAX_UDP_PACKAGE_SIZE = msg.MAX_MESSAGE_SIZE
)

type UDPConn struct {
	ConnCommonFields
	UdpConn *net.UDPConn
	addr    *net.UDPAddr

	lastTime int64
}

func NewUDPConn(c *net.UDPConn, addr *net.UDPAddr) *UDPConn {
	return &UDPConn{UdpConn: c, addr: addr, lastTime: time.Now().Unix(), ConnCommonFields: NewConnCommonFileds()}
}

func (c *UDPConn) ReadLoop() error {
	return nil
}

func (c *UDPConn) WriteLoop() (err error) {
	defer func() {
		if err != nil {
			c.SetStatusToError(err)
		}
	}()
	for {
		select {
		case m, ok := <-c.Out:
			if !ok {
				c.CTXLogger.Debug("udp conn closed")
				return nil
			}
			c.CTXLogger.Debugf("msg out %x", m)
			err := c.Write(m)
			if err != nil {
				c.CTXLogger.Debugf("write msg is failed %v", err)
				return err
			}
		}
	}
}

func (c *UDPConn) Write(bytes []byte) error {
	s := atomic.AddUint32(&c.seq, 1)
	m := msg.New(msg.TYPE_NORMAL, s, bytes)
	c.AddMsg(s, m)
	return c.WriteBytes(m.Bytes())
}

func (c *UDPConn) WriteBytes(bytes []byte) error {
	c.writeMutex.Lock()
	defer c.writeMutex.Unlock()
	_, err := c.UdpConn.WriteToUDP(bytes, c.addr)
	return err
}

func (c *UDPConn) Ack(seq uint32) error {
	resp := make([]byte, msg.MSG_SEQ_END)
	resp[msg.MSG_TYPE_BEGIN] = msg.TYPE_ACK
	binary.BigEndian.PutUint32(resp[msg.MSG_SEQ_BEGIN:], seq)
	return c.WriteBytes(resp)
}

func (c *UDPConn) GetChanOut() chan<- []byte {
	return c.Out
}

func (c *UDPConn) GetChanIn() <-chan []byte {
	return c.In
}

func (c *UDPConn) GetLastTime() int64 {
	c.fieldsMutex.RLock()
	defer c.fieldsMutex.RUnlock()
	return c.lastTime
}

func (c *UDPConn) UpdateLastTime() {
	c.fieldsMutex.Lock()
	c.lastTime = time.Now().Unix()
	c.fieldsMutex.Unlock()
}

func (c *UDPConn) GetNextSeq() uint32 {
	return atomic.AddUint32(&c.seq, 1)
}

func (c *UDPConn) Close() {
	c.fieldsMutex.Lock()
	if c.UdpConn != nil {
		c.UdpConn.Close()
	}
	c.fieldsMutex.Unlock()
	c.ConnCommonFields.Close()
}