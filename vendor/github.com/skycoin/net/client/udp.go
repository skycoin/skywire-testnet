package client

import (
	"encoding/binary"
	"fmt"
	"github.com/skycoin/net/conn"
	"github.com/skycoin/net/msg"
	"net"
	"time"
)

type ClientUDPConn struct {
	conn.UDPConn
}

func NewClientUDPConn(c *net.UDPConn) *ClientUDPConn {
	return &ClientUDPConn{conn.UDPConn{UdpConn: c, ConnCommonFields: conn.NewConnCommonFileds()}}
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
		maxBuf = maxBuf[:n]

		t := maxBuf[msg.MSG_TYPE_BEGIN]
		switch t {
		case msg.TYPE_PONG:
		case msg.TYPE_ACK:
			seq := binary.BigEndian.Uint32(maxBuf[msg.MSG_SEQ_BEGIN:msg.MSG_SEQ_END])
			c.DelMsg(seq)
			c.UpdateLastAck(seq)
		case msg.TYPE_NORMAL:
			seq := binary.BigEndian.Uint32(maxBuf[msg.MSG_SEQ_BEGIN:msg.MSG_SEQ_END])
			err = c.Ack(seq)
			if err != nil {
				return err
			}
			c.In <- maxBuf[msg.MSG_HEADER_END:]
		default:
			c.CTXLogger.Debugf("not implemented msg type %d", t)
			return fmt.Errorf("not implemented msg type %d", t)
		}
	}
}

const (
	TICK_PERIOD = 60
)

func (c *ClientUDPConn) ping() error {
	return c.WriteBytes(msg.GenPingMsg())
}

func (c *ClientUDPConn) WriteLoop() (err error) {
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
			c.CTXLogger.Debug("Ping out")
			err := c.ping()
			if err != nil {
				return err
			}
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

func (c *ClientUDPConn) Write(bytes []byte) error {
	new := c.GetNextSeq()
	m := msg.New(msg.TYPE_NORMAL, new, bytes)
	c.AddMsg(new, m)
	return c.WriteBytes(m.Bytes())
}

func (c *ClientUDPConn) WriteBytes(bytes []byte) error {
	_, err := c.UdpConn.Write(bytes)
	return err
}

func (c *ClientUDPConn) Ack(seq uint32) error {
	resp := make([]byte, msg.MSG_SEQ_END)
	resp[msg.MSG_TYPE_BEGIN] = msg.TYPE_ACK
	binary.BigEndian.PutUint32(resp[msg.MSG_SEQ_BEGIN:], seq)
	return c.WriteBytes(resp)
}
