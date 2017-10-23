package conn

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync/atomic"
	"time"

	"github.com/skycoin/net/msg"
)

const (
	TCP_READ_TIMEOUT = 90
)

type TCPConn struct {
	ConnCommonFields
	*PendingMap
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
			err = c.ReadBytes(reader, header[:msg.MSG_SEQ_END], msg.MSG_SEQ_END)
			if err != nil {
				return err
			}
			seq := binary.BigEndian.Uint32(header[msg.MSG_SEQ_BEGIN:msg.MSG_SEQ_END])
			c.DelMsg(seq)
			c.UpdateLastAck(seq)
		case msg.TYPE_PONG:
			n := msg.PING_MSG_HEADER_END
			reader.Discard(n)
			c.AddReceivedBytes(n)
		case msg.TYPE_NORMAL:
			err = c.ReadBytes(reader, header, msg.MSG_HEADER_SIZE)
			if err != nil {
				return err
			}

			m := msg.NewByHeader(header)
			err = c.ReadBytes(reader, m.Body, int(m.Len))
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

func (c *TCPConn) ReadBytes(r io.Reader, buf []byte, min int) (err error) {
	n, err := io.ReadAtLeast(r, buf, min)
	if err != nil {
		return
	}
	c.AddReceivedBytes(n)
	return
}

func (c *TCPConn) Write(bytes []byte) error {
	s := atomic.AddUint32(&c.seq, 1)
	m := msg.New(msg.TYPE_NORMAL, s, bytes)
	c.AddMsg(s, m)
	return c.WriteBytes(m.Bytes())
}

func (c *TCPConn) WriteBytes(bytes []byte) error {
	c.CTXLogger.Debugf("write %x", bytes)
	c.WriteMutex.Lock()
	defer c.WriteMutex.Unlock()
	for index := 0; index != len(bytes); {
		n, err := c.TcpConn.Write(bytes[index:])
		if err != nil {
			return err
		}
		index += n
		c.AddSentBytes(n)
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
	c.ConnCommonFields.UpdateLastTime()
}

func (c *TCPConn) Close() {
	c.FieldsMutex.Lock()
	if c.TcpConn != nil {
		c.TcpConn.Close()
	}
	c.FieldsMutex.Unlock()
	c.ConnCommonFields.Close()
}

func (c *TCPConn) GetRemoteAddr() net.Addr {
	return c.TcpConn.RemoteAddr()
}

func (c *TCPConn) IsTCP() bool {
	return true
}

func (c *TCPConn) IsUDP() bool {
	return false
}
