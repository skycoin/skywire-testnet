package conn

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"github.com/skycoin/net/msg"
	"io"
	"net"
	"sync/atomic"
	"time"
)

const (
	TCP_READ_TIMEOUT = 90
)

type TCPConn struct {
	ConnCommonFields
	TcpConn net.Conn
}

func (c *TCPConn) ReadLoop() (err error) {
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
	header := make([]byte, msg.MSG_HEADER_SIZE)
	reader := bufio.NewReader(c.TcpConn)

	for {
		t, err := reader.Peek(msg.MSG_TYPE_SIZE)
		if err != nil {
			return err
		}
		msg_t := t[msg.MSG_TYPE_BEGIN]
		switch msg_t {
		case msg.TYPE_ACK:
			_, err = io.ReadAtLeast(reader, header[:msg.MSG_SEQ_END], msg.MSG_SEQ_END)
			if err != nil {
				return err
			}
			seq := binary.BigEndian.Uint32(header[msg.MSG_SEQ_BEGIN:msg.MSG_SEQ_END])
			c.DelMsg(seq)
			c.UpdateLastAck(seq)
		case msg.TYPE_PONG:
			reader.Discard(msg.PING_MSG_HEADER_END)
			c.CTXLogger.Debug("recv pong")
		case msg.TYPE_NORMAL:
			_, err = io.ReadAtLeast(reader, header, msg.MSG_HEADER_SIZE)
			if err != nil {
				return err
			}

			m := msg.NewByHeader(header)
			_, err = io.ReadAtLeast(reader, m.Body, int(m.Len))
			if err != nil {
				return err
			}

			seq := binary.BigEndian.Uint32(header[msg.MSG_SEQ_BEGIN:msg.MSG_SEQ_END])
			c.Ack(seq)
			c.CTXLogger.Debugf("c.In <- m.Body %x", m.Body)
			c.In <- m.Body
		default:
			c.CTXLogger.Debugf("not implemented msg type %d", t)
			return fmt.Errorf("not implemented msg type %d", msg_t)
		}
		c.UpdateLastTime()
	}
}

func (c *TCPConn) WriteLoop() (err error) {
	defer func() {
		if err != nil {
			c.SetStatusToError(err)
		}
	}()
	for {
		select {
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

func getTCPReadDeadline() time.Time {
	return time.Now().Add(time.Second * TCP_READ_TIMEOUT)
}

func (c *TCPConn) Write(bytes []byte) error {
	s := atomic.AddUint32(&c.seq, 1)
	m := msg.New(msg.TYPE_NORMAL, s, bytes)
	c.AddMsg(s, m)
	return c.WriteBytes(m.Bytes())
}

func (c *TCPConn) WriteBytes(bytes []byte) error {
	c.CTXLogger.Debugf("write %x", bytes)
	c.writeMutex.Lock()
	defer c.writeMutex.Unlock()
	index := 0
	for n, err := c.TcpConn.Write(bytes[index:]); index != len(bytes); index += n {
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *TCPConn) Ack(seq uint32) error {
	resp := make([]byte, msg.MSG_SEQ_END)
	resp[msg.MSG_TYPE_BEGIN] = msg.TYPE_ACK
	binary.BigEndian.PutUint32(resp[msg.MSG_SEQ_BEGIN:], seq)
	return c.WriteBytes(resp)
}

func (c *TCPConn) Ping() error {
	return c.WriteBytes(msg.GenPingMsg())
}

func (c *TCPConn) GetChanOut() chan<- []byte {
	return c.Out
}

func (c *TCPConn) GetChanIn() <-chan []byte {
	return c.In
}

func (c *TCPConn) UpdateLastTime() {
	c.TcpConn.SetReadDeadline(getTCPReadDeadline())
}

func (c *TCPConn) Close() {
	c.fieldsMutex.Lock()
	if c.TcpConn != nil {
		c.TcpConn.Close()
	}
	c.fieldsMutex.Unlock()
	c.ConnCommonFields.Close()
}