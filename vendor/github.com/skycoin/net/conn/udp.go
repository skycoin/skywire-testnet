package conn

import (
	"encoding/binary"
	"hash/crc32"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"fmt"

	"errors"

	"github.com/skycoin/net/msg"
)

const (
	MAX_UDP_PACKAGE_SIZE = 1200
)

type UDPConn struct {
	ConnCommonFields
	*UDPPendingMap
	StreamQueue
	UdpConn *net.UDPConn
	addr    *net.UDPAddr

	// write loop with ping
	SendPing bool
	wmx      sync.Mutex
	rto      time.Duration

	rtoResendCount  uint32
	lossResendCount uint32
	ackCount        uint32
	overAckCount    uint32
}

// used for server spawn udp conn
func NewUDPConn(c *net.UDPConn, addr *net.UDPAddr) *UDPConn {
	return &UDPConn{
		UdpConn:          c,
		addr:             addr,
		ConnCommonFields: NewConnCommonFileds(),
		UDPPendingMap:    NewUDPPendingMap(),
		rto:              200 * time.Millisecond,
	}
}

func (c *UDPConn) ReadLoop() error {
	return nil
}

func (c *UDPConn) WriteLoop() (err error) {
	if c.SendPing {
		err = c.writeLoopWithPing()
	} else {
		err = c.writeLoop()
	}
	c.GetContextLogger().Debugf("%s", c.String())
	return
}

func (c *UDPConn) writeLoop() (err error) {
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
			err := c.Write(m)
			if err != nil {
				c.CTXLogger.Debugf("write msg is failed %v", err)
				return err
			}
		}
	}
}

func (c *UDPConn) writeLoopWithPing() (err error) {
	ticker := time.NewTicker(time.Second * UDP_PING_TICK_PERIOD)
	defer func() {
		ticker.Stop()
		if err != nil {
			c.SetStatusToError(err)
		}
	}()

	for {
		select {
		case <-ticker.C:
			nowUnix := time.Now().Unix()
			if nowUnix-c.GetLastTime() >= UDP_GC_PERIOD {
				c.Close()
				return errors.New("timeout")
			}
			err := c.Ping()
			if err != nil {
				return err
			}
		case m, ok := <-c.Out:
			if !ok {
				c.CTXLogger.Debug("udp conn closed")
				return nil
			}
			err := c.Write(m)
			if err != nil {
				c.CTXLogger.Debugf("write msg is failed %v", err)
				return err
			}
		}
	}
}

func (c *UDPConn) Write(bytes []byte) (err error) {
	c.wmx.Lock()
	defer c.wmx.Unlock()
	s := c.GetNextSeq()
	m := msg.NewUDP(msg.TYPE_NORMAL, s, bytes)
	c.AddMsg(s, m)
	return c.WriteBytes(m.PkgBytes())
}

func (c *UDPConn) WriteBytes(bytes []byte) error {
	//c.CTXLogger.Debugf("write %x", bytes)
	c.AddSentBytes(len(bytes))
	c.WriteMutex.Lock()
	defer c.WriteMutex.Unlock()
	n, err := c.UdpConn.WriteToUDP(bytes, c.addr)
	if err == nil && n != len(bytes) {
		return errors.New("nothing was written")
	}
	return err
}

func (c *UDPConn) Ack(seq uint32) error {
	p := make([]byte, msg.MSG_SEQ_END+msg.PKG_HEADER_SIZE)
	m := p[msg.PKG_HEADER_SIZE:]
	m[msg.MSG_TYPE_BEGIN] = msg.TYPE_ACK
	binary.BigEndian.PutUint32(m[msg.MSG_SEQ_BEGIN:], seq)
	checksum := crc32.ChecksumIEEE(m)
	binary.BigEndian.PutUint32(p[msg.PKG_CRC32_BEGIN:], checksum)
	return c.WriteBytes(p)
}

func (c *UDPConn) Ping() error {
	p := make([]byte, msg.PING_MSG_HEADER_SIZE+msg.PKG_HEADER_SIZE)
	m := p[msg.PKG_HEADER_SIZE:]
	m[msg.PING_MSG_TYPE_BEGIN] = msg.TYPE_PING
	binary.BigEndian.PutUint64(m[msg.PING_MSG_TIME_BEGIN:], msg.UnixMillisecond())
	checksum := crc32.ChecksumIEEE(m)
	binary.BigEndian.PutUint32(p[msg.PKG_CRC32_BEGIN:], checksum)
	return c.WriteBytes(p)
}

func (c *UDPConn) GetChanOut() chan<- []byte {
	return c.Out
}

func (c *UDPConn) GetChanIn() <-chan []byte {
	return c.In
}

func (c *UDPConn) GetNextSeq() uint32 {
	return atomic.AddUint32(&c.seq, 1)
}

func (c *UDPConn) Close() {
	c.ConnCommonFields.Close()
}

func (c *UDPConn) String() string {
	return fmt.Sprintf(
		`udp connection(%s):
			rtoResend:%d,
			lossResend:%d,
			ack:%d,
			overAck:%d,`,
		c.GetRemoteAddr().String(),
		atomic.LoadUint32(&c.rtoResendCount),
		atomic.LoadUint32(&c.lossResendCount),
		atomic.LoadUint32(&c.ackCount),
		atomic.LoadUint32(&c.overAckCount),
	)
}

func (c *UDPConn) GetRemoteAddr() net.Addr {
	return c.addr
}

func (c *UDPConn) GetRTO() (rto time.Duration) {
	c.FieldsMutex.RLock()
	rto = c.rto
	c.FieldsMutex.RUnlock()
	return
}

func (c *UDPConn) SetRTO(rto time.Duration) {
	c.FieldsMutex.Lock()
	c.rto = rto
	c.FieldsMutex.Unlock()
}

func (c *UDPConn) AddMsg(k uint32, v *msg.UDPMessage) {
	v.SetRTO(c.GetRTO(), func() (err error) {
		c.AddLossResendCount()
		err = c.WriteBytes(v.PkgBytes())
		if err != nil {
			c.SetStatusToError(err)
			c.Close()
		}
		return
	})
	c.UDPPendingMap.AddMsg(k, v)
}

func (c *UDPConn) DelMsg(seq uint32) error {
	c.AddAckCount()
	ok, msgs := c.DelMsgAndGetLossMsgs(seq)
	if ok {
		if len(msgs) > 1 {
			c.CTXLogger.Debugf("resend loss msgs %v", msgs)
			for _, msg := range msgs {
				err := c.WriteBytes(msg.PkgBytes())
				if err != nil {
					c.SetStatusToError(err)
					c.Close()
					return err
				}
				c.AddLossResendCount()
			}
		}
		c.UpdateLastAck(seq)
	} else {
		c.CTXLogger.Debugf("over ack %s", c)
		c.AddOverAckCount()
	}
	return nil
}

func (c *UDPConn) AddLossResendCount() {
	atomic.AddUint32(&c.lossResendCount, 1)
}

func (c *UDPConn) AddResendCount() {
	atomic.AddUint32(&c.rtoResendCount, 1)
}

func (c *UDPConn) AddAckCount() {
	atomic.AddUint32(&c.ackCount, 1)
}

func (c *UDPConn) AddOverAckCount() {
	atomic.AddUint32(&c.overAckCount, 1)
}

func (c *UDPConn) IsTCP() bool {
	return false
}

func (c *UDPConn) IsUDP() bool {
	return true
}
