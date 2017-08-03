package server

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"github.com/skycoin/net/conn"
	"github.com/skycoin/net/msg"
	"io"
	"net"
)

type ServerTCPConn struct {
	conn.TCPConn
}

func NewServerTCPConn(c *net.TCPConn) *ServerTCPConn {
	return &ServerTCPConn{TCPConn: conn.TCPConn{TcpConn: c, ConnCommonFields: conn.NewConnCommonFileds()}}
}

func (c *ServerTCPConn) ReadLoop() (err error) {
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
	pingHeader := make([]byte, msg.PING_MSG_HEADER_SIZE)
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
		case msg.TYPE_PING:
			_, err = io.ReadAtLeast(reader, pingHeader, msg.PING_MSG_HEADER_SIZE)
			if err != nil {
				return err
			}
			pingHeader[msg.PING_MSG_TYPE_BEGIN] = msg.TYPE_PONG
			err = c.WriteBytes(pingHeader)
			if err != nil {
				return err
			}
			c.CTXLogger.Debug("recv ping")
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

			seq := binary.BigEndian.Uint32(header[msg.MSG_TYPE_END:msg.MSG_SEQ_END])
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
