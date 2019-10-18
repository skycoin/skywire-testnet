package server

import (
	"bufio"
	"fmt"
	"net"

	"github.com/SkycoinProject/skywire/pkg/net/conn"
	"github.com/SkycoinProject/skywire/pkg/net/msg"
)

type ServerTCPConn struct {
	conn.TCPConn
}

func NewServerTCPConn(c *net.TCPConn) *ServerTCPConn {
	return &ServerTCPConn{
		TCPConn: conn.TCPConn{
			TcpConn:          c,
			ConnCommonFields: conn.NewConnCommonFileds(),
		},
	}
}

func (c *ServerTCPConn) ReadLoop() (err error) {
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
	pingHeader := make([]byte, msg.PING_MSG_HEADER_SIZE)
	reader := bufio.NewReader(conn.NewCryptoReader(c.TcpConn, c))

	for {
		t, err := reader.Peek(msg.MSG_TYPE_SIZE)
		if err != nil {
			return err
		}
		msg_t := t[msg.MSG_TYPE_BEGIN]
		switch msg_t {
		case msg.TYPE_PING:
			err = c.ReadBytes(reader, pingHeader, msg.PING_MSG_HEADER_SIZE)
			if err != nil {
				return err
			}
			pingHeader[msg.PING_MSG_TYPE_BEGIN] = msg.TYPE_PONG
			err = c.WriteBytes(pingHeader)
			if err != nil {
				return err
			}
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
