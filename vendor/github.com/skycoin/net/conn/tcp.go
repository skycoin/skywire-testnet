package conn

import (
	"bufio"
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
	*ConnCommonFields
	TcpConn net.Conn
}

func (c *TCPConn) ReadLoop() (err error) {
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
	header := make([]byte, msg.MSG_HEADER_SIZE)
	reader := bufio.NewReader(NewCryptoReader(c.TcpConn, c))

	for {
		t, err := reader.Peek(msg.MSG_TYPE_SIZE)
		if err != nil {
			return err
		}
		msg_t := t[msg.MSG_TYPE_BEGIN]
		switch msg_t {
		case msg.TYPE_PONG:
			n := msg.PING_MSG_HEADER_END
			reader.Discard(n)
			c.AddReceivedBytes(n)
		case msg.TYPE_SYN, msg.TYPE_NORMAL:
			err = c.ReadBytes(reader, header, msg.MSG_HEADER_SIZE)
			if err != nil {
				return err
			}

			m := msg.NewByHeader(header)
			err = c.ReadBytes(reader, m.Body, int(m.Len))
			if err != nil {
				return err
			}
			c.In <- m.Body
		default:
			c.GetContextLogger().Debugf("not implemented msg type %d", t)
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
				c.GetContextLogger().Debug("conn closed")
				return nil
			}
			c.GetContextLogger().Debugf("msg Out %x", m)
			err := c.Write(m)
			if err != nil {
				c.GetContextLogger().Debugf("write msg is failed %v", err)
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
	return c.WriteBytes(m.Bytes())
}

func (c *TCPConn) WriteSyn(bytes []byte) error {
	s := atomic.AddUint32(&c.seq, 1)
	m := msg.New(msg.TYPE_SYN, s, bytes)
	return c.writeDirectly(m.Bytes())
}

func (c *TCPConn) writeDirectly(bytes []byte) (err error) {
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
	return
}

func (c *TCPConn) WriteBytes(bytes []byte) (err error) {
	crypto := c.GetCrypto()
	if crypto != nil {
		err = crypto.Encrypt(bytes)
		if err != nil {
			return
		}
	}
	err = c.writeDirectly(bytes)
	return
}

func (c *TCPConn) Ping() error {
	return c.WriteBytes(msg.GenPingMsg())
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
