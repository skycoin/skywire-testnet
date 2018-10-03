package websocket

import (
	"encoding/binary"
	"encoding/json"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/skycoin/skycoin/src/cipher"
	net "github.com/skycoin/skywire/pkg/net/skycoin-messenger/factory"
	"github.com/skycoin/skywire/pkg/net/skycoin-messenger/msg"
	_ "github.com/skycoin/skywire/pkg/net/skycoin-messenger/op"
)

type Client struct {
	sync.RWMutex
	factory *net.MessengerFactory

	push   chan interface{}
	Logger *log.Entry

	seq uint32
	PendingMap

	conn *websocket.Conn
}

func (c *Client) GetFactory() *net.MessengerFactory {
	c.RLock()
	defer c.RUnlock()
	return c.factory
}

func (c *Client) SetFactory(factory *net.MessengerFactory) {
	c.Lock()
	if c.factory != nil {
		c.factory.Close()
	}
	c.factory = factory
	c.Unlock()
}

type pushMsg struct {
	op   byte
	data interface{}
}

var pushMsgPool = &sync.Pool{
	New: func() interface{} {
		return new(pushMsg)
	},
}

func (c *Client) Push(op byte, d interface{}) {
	p := pushMsgPool.Get().(*pushMsg)
	p.op = op
	p.data = d
	c.push <- p
}

func (c *Client) PushLoop(conn *net.Connection) {
	defer func() {
		if err := recover(); err != nil {
			c.Logger.Errorf("PushLoop recovered err %v", err)
		}
	}()
	key := conn.GetKey()
	c.Push(msg.OP_LOGIN, &msg.Reg{PublicKey: key.Hex()})
	for {
		select {
		case m, ok := <-conn.GetChanIn():
			if !ok || len(m) < net.MSG_HEADER_END {
				return
			}
			op := m[net.MSG_OP_BEGIN]
			switch op {
			case net.OP_SEND:
				if len(m) < net.SEND_MSG_META_END {
					continue
				}
				key := cipher.NewPubKey(m[net.SEND_MSG_PUBLIC_KEY_BEGIN:net.SEND_MSG_PUBLIC_KEY_END])
				c.Push(msg.OP_SEND, msg.GetPushMsg(key.Hex(), string(m[net.SEND_MSG_META_END:])))
			}
		}
	}
}

func (c *Client) readLoop() {
	defer func() {
		if err := recover(); err != nil {
			c.Logger.Errorf("readLoop recovered err %v", err)
		}
		c.conn.Close()
		close(c.push)
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, m, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				c.Logger.Errorf("error: %v", err)
			}
			c.Logger.Errorf("error: %v", err)
			return
		}
		if len(m) < msg.MSG_HEADER_END {
			return
		}
		c.Logger.Debugf("recv %x", m)
		opn := int(m[msg.MSG_OP_BEGIN])
		if opn == msg.OP_ACK {
			c.DelMsg(binary.BigEndian.Uint32(m[msg.MSG_SEQ_BEGIN:msg.MSG_SEQ_END]))
			continue
		}
		op := msg.GetOP(opn)
		if op == nil {
			c.Logger.Errorf("op not found, %d", opn)
			return
		}

		c.ack(m[msg.MSG_OP_BEGIN:msg.MSG_SEQ_END])

		err = json.Unmarshal(m[msg.MSG_HEADER_END:], op)
		if err == nil {
			err = op.Execute(c)
			if err != nil {
				c.Logger.Errorf("websocket readLoop execute err: %v", err)
			}
		} else {
			c.Logger.Errorf("websocket readLoop json Unmarshal err: %v", err)
		}
		msg.PutOP(opn, op)
	}
}

func (c *Client) writeLoop() (err error) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		if err := recover(); err != nil {
			c.Logger.Errorf("writeLoop recovered err %v", err)
		}
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.push:
			c.Logger.Debug("Push", message)
			if !ok {
				c.Logger.Debug("closed c.Push")
				err = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				if err != nil {
					c.Logger.Error(err)
					return
				}
			}

			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			w, err := c.conn.NextWriter(websocket.BinaryMessage)
			if err != nil {
				c.Logger.Error(err)
				return err
			}
			switch m := message.(type) {
			case *pushMsg:
				err = c.write(w, m.op, m.data)
				if _, ok := m.data.(*msg.PushMsg); ok {
					msg.PutPushMsg(m.data)
				}
				pushMsgPool.Put(m)
			default:
				c.Logger.Errorf("not implemented msg %v", m)
			}
			if err != nil {
				c.Logger.Error(err)
				return err
			}
			if err = w.Close(); err != nil {
				c.Logger.Error(err)
				return err
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				c.Logger.Error(err)
				return err
			}
		}
	}
}

func (c *Client) write(w io.WriteCloser, op byte, m interface{}) (err error) {
	_, err = w.Write([]byte{op})
	c.Logger.Debugf("op %d", op)
	if err != nil {
		return
	}
	ss := make([]byte, 4)
	nseq := atomic.AddUint32(&c.seq, 1)
	c.AddMsg(nseq, m)
	binary.BigEndian.PutUint32(ss, nseq)
	_, err = w.Write(ss)
	c.Logger.Debugf("seq %x", ss)
	if err != nil {
		return
	}
	jbs, err := json.Marshal(m)
	if err != nil {
		return
	}
	_, err = w.Write(jbs)
	c.Logger.Debugf("json %x", jbs)
	if err != nil {
		return
	}

	return nil
}

func (c *Client) ack(data []byte) error {
	data[msg.MSG_OP_BEGIN] = msg.OP_ACK
	c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	return c.conn.WriteMessage(websocket.BinaryMessage, data)
}
