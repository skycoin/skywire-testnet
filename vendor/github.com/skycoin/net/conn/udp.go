package conn

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/google/btree"
	"github.com/sirupsen/logrus"
	"github.com/skycoin/net/msg"
	"hash/crc32"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const (
	MAX_UDP_PACKAGE_SIZE = 1200
)

type UDPConn struct {
	*ConnCommonFields
	*UDPPendingMap
	streamQueue
	UdpConn         *net.UDPConn
	UnsharedUdpConn bool
	addr            *net.UDPAddr

	// write loop with ping
	SendPing bool
	rto      time.Duration
	rtt      time.Duration

	rtoResendCount  uint32
	lossResendCount uint32
	ackCount        uint32
	overAckCount    uint32

	lastAck     uint32
	lastCnt     uint32
	lastCnted   uint32
	lastAckCond *sync.Cond
	lastAckMtx  sync.Mutex

	// congestion algorithm
	*ca
	pacingTimer      *time.Timer
	pacingTimerMutex sync.Mutex
	pacingChan       chan struct{}

	// fec
	*fecEncoder
	*fecDecoder

	closed bool
}

const (
	dataShards   = 4
	parityShards = 1
)

// used for server spawn udp conn
func NewUDPConn(c *net.UDPConn, addr *net.UDPAddr) *UDPConn {
	conn := &UDPConn{
		UdpConn:          c,
		addr:             addr,
		ConnCommonFields: NewConnCommonFileds(),
		UDPPendingMap:    NewUDPPendingMap(),
		streamQueue:      newFECStreamQueue(dataShards, parityShards),
		rto:              300 * time.Millisecond,
		fecEncoder:       newFECEncoder(dataShards, parityShards),
		fecDecoder:       newFECDecoder(dataShards, parityShards),
	}
	conn.ca = newCA()
	conn.pacingTimer = time.NewTimer(0)
	if !conn.pacingTimer.Stop() {
		<-conn.pacingTimer.C
	}
	conn.pacingChan = make(chan struct{}, 1)
	conn.lastAckCond = sync.NewCond(&conn.lastAckMtx)
	go conn.ackLoop()
	return conn
}

func (c *UDPConn) ReadLoop() error {
	return nil
}

func (c *UDPConn) WriteLoop() (err error) {
	var pingTicker *time.Ticker
	var pingTickerChan <-chan time.Time
	if c.SendPing {
		pingTicker = time.NewTicker(time.Second * UDP_PING_TICK_PERIOD)
		pingTickerChan = pingTicker.C
	}
	defer func() {
		if pingTicker != nil {
			pingTicker.Stop()
		}
		if err != nil {
			c.SetStatusToError(err)
		}
		c.GetContextLogger().Debugf("udp conn closed %s", c.String())
	}()

	for {
		select {
		case <-pingTickerChan:
			if c.GetCrypto() == nil {
				continue
			}
			nowUnix := time.Now().Unix()
			lastTime := c.GetLastTime()
			if nowUnix-lastTime >= UDP_GC_PERIOD {
				c.Close()
				return errors.New("timeout")
			} else if nowUnix-lastTime < UDP_PING_TICK_PERIOD {
				continue
			}
			err := c.Ping()
			if err != nil {
				return err
			}
		case m, ok := <-c.Out:
			if !ok {
				return nil
			}
			err := c.Write(m)
			if err != nil {
				c.GetContextLogger().Debugf("write msg is failed %v", err)
				return err
			}
		case <-c.pacingChan:
			err := c.writePendingMsgs()
			if err != nil {
				c.SetStatusToError(err)
				c.Close()
			}
		case <-c.pacingTimer.C:
			err := c.writePendingMsgs()
			if err != nil {
				c.SetStatusToError(err)
				c.Close()
			}
		}
	}
}

func (c *UDPConn) ackReady() (seq uint32, ok bool) {
	c.lastAckMtx.Lock()
	seq = c.lastAck
	if c.lastCnt != c.lastCnted {
		ok = true
		c.lastCnted = c.lastCnt
	}
	c.lastAckMtx.Unlock()
	return
}

func (c *UDPConn) ackLoop() (err error) {
	t := time.NewTicker(2 * time.Millisecond)
	defer func() {
		t.Stop()
		if err != nil {
			c.SetStatusToError(err)
		}
	}()

	for {
		select {
		case <-t.C:
			la, ok := c.ackReady()
			if ok {
				err = c.ack(la)
				if err != nil {
					return
				}
			} else {
				t.Stop()
				c.lastAckMtx.Lock()
				c.lastAckCond.Wait()
				c.lastAckMtx.Unlock()
				if c.IsClosed() {
					return
				}
				t = time.NewTicker(2 * time.Millisecond)
			}
		case <-c.disconnected:
			return
		}
	}
}

func (c *UDPConn) Write(bytes []byte) (err error) {
	err = c.WriteToChannel(0, bytes)
	return
}

func (c *UDPConn) WriteToChannel(channel int, bytes []byte) (err error) {
	err = c.writeToChannel(channel, bytes, msg.TYPE_NORMAL)
	return
}

func (c *UDPConn) writeToChannel(channel int, bytes []byte, msgt byte) (err error) {
	if len(bytes) > MAX_UDP_PACKAGE_SIZE {
		for i := 0; i < len(bytes)/MAX_UDP_PACKAGE_SIZE; i++ {
			err = c.addToChannel(channel, bytes[i*MAX_UDP_PACKAGE_SIZE:(i+1)*MAX_UDP_PACKAGE_SIZE], msgt)
			if err != nil {
				return
			}
		}
		i := len(bytes) % MAX_UDP_PACKAGE_SIZE
		if i > 0 {
			err = c.addToChannel(channel, bytes[len(bytes)-i:], msgt)
			if err != nil {
				return
			}
		}
	} else {
		err = c.addToChannel(channel, bytes, msgt)
	}
	return
}

func (c *UDPConn) addToChannel(channel int, bytes []byte, msgt byte) (err error) {
	m := msg.NewUDPWithoutSeq(msgt, bytes)
	c.addToPendingChannel(channel, m)
	c.pacingChan <- struct{}{}
	return
}

func (c *UDPConn) resendCallback(m *msg.UDPMessage) (err error) {
	c.AddRTOResendCount()
	err = c.resendMsg(m)
	if err != nil {
		c.SetStatusToError(err)
		c.Close()
	}
	return
}

func (c *UDPConn) transmitted(m *msg.UDPMessage) {
	seq := m.GetSeq()
	c.ca.updateLastSentSeq(seq)
	c.ca.checkAppLimited(seq)
	c.addMsg(seq, m)
	m.Transmitted()
	m.SetRTO(c.getRTO(), c.resendCallback)
	m.UpdateState(c.getDelivered(), c.getDeliveredTime(), c.getSentTime())
}

func (c *UDPConn) resendMsg(m *msg.UDPMessage) (err error) {
	if m.IsAcked() {
		return
	}
	m.Loss()
	c.GetContextLogger().Debugf("resendMsg %s", m)
	c.addToResendChannel(m)
	c.pacingChan <- struct{}{}
	return
}

func (c *UDPConn) writePendingMsgs() (err error) {
	c.ca.nextPacingMutex.Lock()
	defer c.ca.nextPacingMutex.Unlock()
	for {
		if !c.ca.isPacingTime() {
			return nil
		}
		m := c.ca.popMessage()
		c.GetContextLogger().Debugf("popMessage bif %d, m %v", c.ca.getBytesInFlight(), m)
		if m == nil {
			return nil
		}
		tx := !m.IsTransmitted()
		if tx {
			m.SetSeq(c.GetNextSeq())
			c.GetContextLogger().Debugf("new msg seq %d", m.GetSeq())
		} else {
			c.GetContextLogger().Debugf("resend msg seq %d", m.GetSeq())
		}
		pkgBytes := m.PkgBytes()
		if DEBUG_DATA_HEX {
			c.GetContextLogger().Debugf("before encrypt out %x", pkgBytes)
		}
		switch m.Type {
		case msg.TYPE_NORMAL:
			if tx {
				crypto := c.GetCrypto()
				if crypto != nil {
					err = crypto.Encrypt(pkgBytes[msg.PKG_HEADER_SIZE+msg.UDP_HEADER_END:])
					if err != nil {
						return
					}
				}
				m.SetCache(pkgBytes)
			}
			err = c.WriteBytes(pkgBytes)
		case msg.TYPE_SYN:
			err = c.WriteBytes(pkgBytes)
		}
		if err != nil {
			return err
		}
		d := c.ca.calcPacingTime(m.PkgBytesLen())
		c.pacingTimerMutex.Lock()
		c.pacingTimer.Reset(d)
		c.pacingTimerMutex.Unlock()
		if tx {
			c.transmitted(m)
			ps, err := c.fecEncoder.encode(pkgBytes[msg.PKG_HEADER_SIZE:])
			if err != nil {
				return err
			}
			if len(ps) > 0 {
				for _, v := range ps {
					err = c.WriteBytes(fec(v, c.GetNextSeq()))
					if err != nil {
						return err
					}
				}
			}
		} else {
			m.SetRTO(c.getRTO(), c.resendCallback)
		}
	}
}

func (c *UDPConn) fillAckInfo(m []byte) {
	c.lastAckMtx.Lock()
	seq := c.lastAck
	c.lastCnted = c.lastCnt
	c.lastAckMtx.Unlock()
	nSeq := c.GetNextAckSeq()
	binary.BigEndian.PutUint32(m[msg.UDP_ACK_SEQ_BEGIN:], seq)
	binary.BigEndian.PutUint32(m[msg.UDP_ACK_NEXT_SEQ_BEGIN:], nSeq)
	if seq > nSeq+1 {
		acked := c.GetAckedSeqs(nSeq+1, seq)
		binary.BigEndian.PutUint32(m[msg.UDP_ACK_ACKED_SEQ_BEGIN:], acked)
	} else {
		binary.BigEndian.PutUint32(m[msg.UDP_ACK_ACKED_SEQ_BEGIN:], 0)
	}
}

func fec(b []byte, seq uint32) (result []byte) {
	hz := msg.PKG_HEADER_SIZE + msg.UDP_HEADER_SIZE
	result = make([]byte, hz+len(b))
	l := copy(result[hz:], b)
	m := result[msg.PKG_HEADER_SIZE:]
	m[0] = msg.TYPE_FEC
	binary.BigEndian.PutUint32(m[msg.UDP_SEQ_BEGIN:msg.UDP_SEQ_END], seq)
	binary.BigEndian.PutUint32(m[msg.UDP_LEN_BEGIN:msg.UDP_LEN_END], uint32(l))
	return
}

func (c *UDPConn) Process(t byte, m []byte) (err error) {
	err = c.processAckInfo(m)
	if err != nil {
		return
	}
	seq := binary.BigEndian.Uint32(m[msg.UDP_SEQ_BEGIN:msg.UDP_SEQ_END])
	l := binary.BigEndian.Uint32(m[msg.UDP_LEN_BEGIN:msg.UDP_LEN_END])
	c.GetContextLogger().Debugf("seq %d l %d, len %d", seq, l, len(m))
	if DEBUG_DATA_HEX {
		c.GetContextLogger().Debugf("%x", m)
	}

	if t == msg.TYPE_FEC {
		m = m[msg.UDP_HEADER_END:]
	}
	g, err := c.decode(seq, m)
	if err != nil {
		return
	}
	if g != nil && g.recovered {
		for i, b := range g.dataRecv {
			if !b {
				m := g.datas[i]
				if len(m) <= msg.UDP_HEADER_SIZE {
					c.GetContextLogger().Error("fec recovered len(m) <= msg.UDP_HEADER_SIZE")
					continue
				}
				t := m[msg.UDP_TYPE_BEGIN]
				seq := binary.BigEndian.Uint32(m[msg.UDP_SEQ_BEGIN:msg.UDP_SEQ_END])
				l := binary.BigEndian.Uint32(m[msg.UDP_LEN_BEGIN:msg.UDP_LEN_END])
				c.GetContextLogger().Debugf("fec recovered seq %d l %d len %d", seq, l, len(m))
				if DEBUG_DATA_HEX {
					c.GetContextLogger().Debugf("fec recovered \n%x", m)
				}
				if uint32(len(m)) >= msg.UDP_HEADER_END+l {
					err = c.process(t, seq, m[msg.UDP_HEADER_END:msg.UDP_HEADER_END+l])
					if err != nil {
						return
					}
				}
			}
		}
	}
	if t != msg.TYPE_FEC &&
		uint32(len(m)) >= msg.UDP_HEADER_END+l {
		err = c.process(t, seq, m[msg.UDP_HEADER_END:msg.UDP_HEADER_END+l])
		if err != nil {
			return
		}
	}

	return
}

func (c *UDPConn) processAckInfo(m []byte) (err error) {
	seq := binary.BigEndian.Uint32(m[msg.UDP_ACK_SEQ_BEGIN:])
	ns := binary.BigEndian.Uint32(m[msg.UDP_ACK_NEXT_SEQ_BEGIN:])
	acked := binary.BigEndian.Uint32(m[msg.UDP_ACK_ACKED_SEQ_BEGIN:])
	c.GetContextLogger().Debugf("udp ack %d, next %d, acked %b", seq, ns, acked)
	return c.recvAck(seq, ns, acked)
}

func (c *UDPConn) process(t byte, seq uint32, m []byte) (err error) {
	switch t {
	case msg.TYPE_SYN, msg.TYPE_NORMAL:
		err = c.Ack(seq)
		if err != nil {
			return
		}
	}
	ok, ms := c.Push(seq, msg.NewUDP(t, seq, m))
	if ok {
		for _, m := range ms {
			if m.Type != msg.TYPE_SYN {
				if DEBUG_DATA_HEX {
					c.GetContextLogger().Debugf("MustGetCrypto t %d seq %d \n%x", m.Type, m.GetSeq(), m.Body)
				}
				crypto := c.MustGetCrypto()
				err = crypto.Decrypt(m.Body)
				if DEBUG_DATA_HEX {
					c.GetContextLogger().Debugf("MustGetCrypto out t %d seq %d \n%x", m.Type, m.GetSeq(), m.Body)
				}
				if err != nil {
					return
				}
			}
			c.In <- m.Body
		}
	}
	return
}

func (c *UDPConn) WriteSyn(bytes []byte) (err error) {
	err = c.writeToChannel(0, bytes, msg.TYPE_SYN)
	return
}

func (c *UDPConn) WriteBytes(bytes []byte) (err error) {
	c.fillAckInfo(bytes[msg.PKG_CRC32_END:])
	checksum := crc32.ChecksumIEEE(bytes[msg.PKG_CRC32_END:])
	binary.BigEndian.PutUint32(bytes[msg.PKG_CRC32_BEGIN:], checksum)
	l := len(bytes)
	c.AddSentBytes(l)
	n, err := c.UdpConn.WriteToUDP(bytes, c.addr)
	if DEBUG_DATA_HEX {
		c.GetContextLogger().Debugf("write out %x", bytes)
	}
	if err == nil && n != l {
		return errors.New("nothing was written")
	}
	return
}

func (c *UDPConn) WriteExt(bytes []byte) (err error) {
	l := len(bytes)
	c.AddSentBytes(l)
	n, err := c.UdpConn.WriteToUDP(bytes, c.addr)
	if DEBUG_DATA_HEX {
		c.GetContextLogger().Debugf("write out %x", bytes)
	}
	if err == nil && n != l {
		return errors.New("nothing was written")
	}
	return
}

func (c *UDPConn) Ack(seq uint32) error {
	c.lastAckMtx.Lock()
	c.lastAck = seq
	c.lastCnt++
	c.lastAckMtx.Unlock()
	c.lastAckCond.Broadcast()
	return nil
}

func (c *UDPConn) ack(seq uint32) error {
	nSeq := c.GetNextAckSeq()
	c.GetContextLogger().Debugf("ack %d, next %d", seq, nSeq)
	p := make([]byte, msg.ACK_HEADER_SIZE+msg.PKG_HEADER_SIZE)
	m := p[msg.PKG_HEADER_SIZE:]
	m[msg.ACK_TYPE_BEGIN] = msg.TYPE_ACK
	binary.BigEndian.PutUint32(m[msg.ACK_SEQ_BEGIN:], seq)
	binary.BigEndian.PutUint32(m[msg.ACK_NEXT_SEQ_BEGIN:], nSeq)
	if seq > nSeq+1 {
		acked := c.GetAckedSeqs(nSeq+1, seq)
		binary.BigEndian.PutUint32(m[msg.ACK_ACKED_SEQ_BEGIN:msg.ACK_ACKED_SEQ_END], acked)
	}
	checksum := crc32.ChecksumIEEE(m)
	binary.BigEndian.PutUint32(p[msg.PKG_CRC32_BEGIN:], checksum)
	return c.WriteExt(p)
}

func (c *UDPConn) fin() error {
	p := make([]byte, msg.PKG_HEADER_SIZE+msg.UDP_TYPE_SIZE)
	m := p[msg.PKG_HEADER_SIZE:]
	m[msg.UDP_TYPE_BEGIN] = msg.TYPE_FIN
	checksum := crc32.ChecksumIEEE(m)
	binary.BigEndian.PutUint32(p[msg.PKG_CRC32_BEGIN:], checksum)
	c.GetContextLogger().Debug("fin")
	return c.WriteExt(p)
}

func (c *UDPConn) recvAck(seq, ns, acked uint32) (err error) {
	err = c.delMsg(seq, false)
	if err != nil {
		return
	}
	for n, ok := c.getMinUnAckSeq(); ok && ns > n; n, ok = c.getMinUnAckSeq() {
		c.GetContextLogger().Debugf("ignore ack %d", n)
		err = c.delMsg(n, true)
		if err != nil {
			return
		}
	}

	for i := uint32(0); acked > 0; i++ {
		if acked&1 > 0 {
			n := i + ns + 1
			c.GetContextLogger().Debugf("ignore ack %d", n)
			err = c.delMsg(n, true)
			if err != nil {
				return
			}
		}
		acked >>= 1
	}

	return
}

func (c *UDPConn) RecvAck(m []byte) (err error) {
	if len(m) < msg.ACK_HEADER_SIZE {
		return fmt.Errorf("invalid ack msg %x", m)
	}
	seq := binary.BigEndian.Uint32(m[msg.ACK_SEQ_BEGIN:msg.ACK_SEQ_END])
	ns := binary.BigEndian.Uint32(m[msg.ACK_NEXT_SEQ_BEGIN:msg.ACK_NEXT_SEQ_END])
	acked := binary.BigEndian.Uint32(m[msg.ACK_ACKED_SEQ_BEGIN:msg.ACK_ACKED_SEQ_END])

	c.GetContextLogger().Debugf("recv ack %d, next %d, acked %b", seq, ns, acked)
	return c.recvAck(seq, ns, acked)
}

func (c *UDPConn) Ping() error {
	c.GetContextLogger().Debug("ping")
	p := make([]byte, msg.PING_MSG_HEADER_SIZE+msg.PKG_HEADER_SIZE)
	m := p[msg.PKG_HEADER_SIZE:]
	m[msg.PING_MSG_TYPE_BEGIN] = msg.TYPE_PING
	binary.BigEndian.PutUint64(m[msg.PING_MSG_TIME_BEGIN:], msg.UnixMillisecond())
	checksum := crc32.ChecksumIEEE(m)
	binary.BigEndian.PutUint32(p[msg.PKG_CRC32_BEGIN:], checksum)
	return c.WriteExt(p)
}

func (c *UDPConn) GetNextSeq() uint32 {
	return atomic.AddUint32(&c.seq, 1)
}

func (c *UDPConn) IsClose() (r bool) {
	c.FieldsMutex.RLock()
	r = c.closed
	c.FieldsMutex.RUnlock()
	return
}

func (c *UDPConn) Close() {
	c.FieldsMutex.Lock()
	if c.closed {
		return
	}
	c.closed = true
	c.FieldsMutex.Unlock()
	if c.UDPPendingMap != nil {
		c.UDPPendingMap.Dismiss()
	}
	if c.addr != nil && c.GetStatusError() != ErrFin {
		c.fin()
	}
	c.ConnCommonFields.Close()
	if c.UnsharedUdpConn {
		c.UdpConn.Close()
	}
	if c.lastAckCond != nil {
		c.lastAckCond.Broadcast()
	}
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

func (c *UDPConn) getRTO() (rto time.Duration) {
	c.FieldsMutex.RLock()
	rto = c.rto
	c.FieldsMutex.RUnlock()
	return
}

func (c *UDPConn) setRTO(rto time.Duration) {
	c.GetContextLogger().Debugf("setRTO %d", rto)
	if rto < MIN_RTO {
		rto = MIN_RTO
	}
	c.FieldsMutex.Lock()
	c.rto = rto
	c.FieldsMutex.Unlock()
}

func (c *UDPConn) addMsg(k uint32, v *msg.UDPMessage) {
	c.UDPPendingMap.AddMsg(k, v)
}

func (c *UDPConn) delMsg(seq uint32, ignore bool) error {
	ok, um, msgs := c.DelMsgAndGetLossMsgs(seq)
	if ok {
		c.AddAckCount()
		if !ignore && !um.IsLoss() {
			c.updateRTT(um.GetRTT())
		}
		c.updateDeliveryRate(um)
		if QUICK_LOST_ENABLE {
			if len(msgs) > 1 {
				c.GetContextLogger().Debugf("resend loss msgs %v", msgs)
				for _, msg := range msgs {
					err := c.resendMsg(msg)
					if err != nil {
						c.SetStatusToError(err)
						c.Close()
						return err
					}
					c.AddLossResendCount()
				}
			}
		}
		c.UpdateLastAck(seq)
		c.ca.cwndMtx.Lock()
		c.ca.usedCwnd--
		c.ca.cwndMtx.Unlock()
		c.ca.bifMtx.Lock()
		c.ca.bif -= um.PkgBytesLen()
		c.ca.bifMtx.Unlock()
		return c.writePendingMsgs()
	} else if !ignore {
		c.GetContextLogger().Debugf("over ack %s", c)
		c.AddOverAckCount()
	}
	return nil
}

func (c *UDPConn) AddLossResendCount() {
	atomic.AddUint32(&c.lossResendCount, 1)
}

func (c *UDPConn) AddRTOResendCount() {
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

func (c *UDPConn) getRTT() time.Duration {
	return time.Duration(atomic.LoadInt64((*int64)(&c.rtt)))
}

type rttSampler struct {
	tree  *btree.BTree
	ring  []rtt
	mask  int
	index int
}

type rtt time.Duration

func (a rtt) Less(b btree.Item) bool {
	return a < b.(rtt)
}

// size should be power of 2
func newRttSampler(size int) *rttSampler {
	if size < 2 || (size&(size-1)) > 0 {
		var n uint
		for size > 0 {
			size >>= 1
			n++
		}
		size = 1 << n
	}
	return &rttSampler{
		ring: make([]rtt, size),
		mask: size - 1,
		tree: btree.New(2),
	}
}

func (t *rttSampler) push(r rtt) rtt {
	if r <= 0 {
		panic("push rtt <= 0")
	}
	or := t.ring[t.index]
	if or > 0 {
		t.tree.Delete(or)
	}
	t.ring[t.index] = r
	t.tree.ReplaceOrInsert(r)
	t.index = (t.index + 1) & t.mask
	return t.tree.Min().(rtt)
}

func (t *rttSampler) getMin() rtt {
	item := t.tree.Min()
	if item == nil {
		return 0
	}
	return item.(rtt)
}

func (c *UDPConn) updateRTT(t time.Duration) {
	if t <= 0 {
		panic("updateRTT t <= 0")
	}
	r := c.rttSamples.push(rtt(t))
	if r <= 0 {
		return
	}
	for {
		ot := c.getRTT()
		if time.Duration(r) != ot {
			ok := atomic.CompareAndSwapInt64((*int64)(&c.rtt), int64(ot), int64(r))
			if !ok {
				continue
			}
			c.setRTO(time.Duration(r) * 2)
		}
		break
	}
}

const rttUnit = time.Microsecond

func (c *UDPConn) updateDeliveryRate(m *msg.UDPMessage) {
	c.ca.Lock()
	defer c.ca.Unlock()

	isRoundStart := c.ca.updateRoundTripCounter(m.GetSeq())

	c.delivered++
	c.deliveredTime = time.Now()
	c.sentTime = m.GetTransmittedTime()

	if m.GetSentTime().IsZero() || m.GetDeliveredTime().IsZero() {
		return
	}

	c.tryToCancelAppLimited(m.GetSeq())
	if c.isAppLimited() {
		c.GetContextLogger().Debugf("app limited used:%d max:%d", c.getUsedCwnd(), c.getCwnd())
		return
	}

	sd := c.sentTime.Sub(m.GetSentTime()) / rttUnit
	ad := c.deliveredTime.Sub(m.GetDeliveredTime()) / rttUnit
	interval := ad
	if sd > ad {
		interval = sd
	}
	d := c.delivered - m.GetDelivered()
	drate := rate(d * BW_UNIT / uint64(interval))
	c.GetContextLogger().Debugf("drate(%d) d %d interval %d sd %d ad %d", drate, d, interval, sd, ad)
	if drate <= 0 {
		return
	}
	mr := c.rttSamples.getMin()
	if mr <= 0 {
		return
	}
	rtt := uint64(time.Duration(mr) / rttUnit)
	if uint64(interval) < rtt {
		return
	}

	hm := c.bwFilter.GetBest()
	if drate >= hm {
		c.bwFilter.Update(drate, c.ca.getRoundTripCount())
		hm = c.bwFilter.GetBest()
	}
	if hm <= 0 {
		return
	}
	max := uint64(hm)
	if c.ca.mode == probeBW {
		c.ca.updateGainCyclePhase(max, rtt)
	}
	if isRoundStart {
		c.ca.checkFullBwReached()
	}
	c.ca.checkDrain(max, rtt)
	c.setPacingRate(max, c.pacingGain)
	c.setCwnd(d, max, rtt, c.cwndGain)
	c.GetContextLogger().Debugf("mode %d, max bw %d rtt %d", c.mode, max, rtt)
}

func (c *UDPConn) setPacingRate(bw uint64, gain int) {
	bw *= MAX_UDP_PACKAGE_SIZE
	bw *= uint64(gain)
	bw >>= BBR_SCALE
	bw *= 1000000
	rate := bw >> BW_SCALE
	c.GetContextLogger().Debugf("setPacingRate: rate %d", rate)
	c.ca.setPacingRate(rate)
}

func (c *UDPConn) setCwnd(acked, bw, rtt uint64, gain int) {
	target := c.targetCwnd(bw, rtt, gain)

	cwnd := c.ca.getCwnd()
	if c.fullBwReached() {
		n := cwnd + uint32(acked)
		if n < target {
			cwnd = n
		} else {
			cwnd = target
		}
	} else if cwnd < target {
		cwnd = cwnd + uint32(acked)
	}
	if 10 > cwnd {
		cwnd = 10
	}

	c.GetContextLogger().Debugf("setCwnd %d", cwnd)
	c.ca.setCwnd(cwnd)
}

type ca struct {
	delivered     uint64
	deliveredTime time.Time
	sentTime      time.Time
	rttSamples    *rttSampler
	bwFilter      *maxBandwidthFilter
	cwnd          uint32
	usedCwnd      uint32
	cwndMtx       sync.Mutex
	mode
	pacingGain      int
	pacingRate      uint64
	nextPacingTime  time.Time
	nextPacingMutex sync.RWMutex
	lastCycleStart  time.Time
	cycleOffset     int
	cwndGain        int
	fullBwCnt       uint
	fullBw          rate
	pendingCnt      int32

	bif        int
	bifMtx     sync.RWMutex
	bifPdId    int
	bifPdChans map[int]*pdChan

	resendChan *reChan

	appLimited   bool
	endOfLimited uint32

	lastSentSeq    uint32
	roundTripCount roundTripCount
	currentTripEnd uint32
	roundTripMutex sync.RWMutex

	sync.RWMutex
}

type pdChan struct {
	pd    *btree.BTree
	seq   uint32
	mtx   sync.Mutex
	cond  *sync.Cond
	maxPd int
	end   bool
}

func newPdChan(max int) *pdChan {
	pd := &pdChan{
		pd:    btree.New(2),
		maxPd: max,
	}
	pd.cond = sync.NewCond(&pd.mtx)
	return pd
}

type reChan struct {
	pd  *btree.BTree
	mtx sync.Mutex
}

func newReChan() *reChan {
	return &reChan{
		pd: btree.New(2),
	}
}

func newCA() *ca {
	c := &ca{
		rttSamples: newRttSampler(16),
		bwFilter:   newMaxBandwidthFilter(bandwidthWindowSize, 0, 0),
		cwnd:       10,
		pacingGain: highGain,
		pacingRate: highGain * 10 * BW_UNIT / 1000,
		cwndGain:   highGain,
		bifPdChans: make(map[int]*pdChan),
		resendChan: newReChan(),
	}

	c.bifPdChans[c.bifPdId] = newPdChan(100)
	return c
}

func (ca *ca) getDelivered() (d uint64) {
	ca.RLock()
	d = ca.delivered
	ca.RUnlock()
	return
}

func (ca *ca) getDeliveredTime() (d time.Time) {
	ca.RLock()
	d = ca.deliveredTime
	ca.RUnlock()
	return
}

func (ca *ca) getSentTime() (d time.Time) {
	ca.RLock()
	d = ca.sentTime
	ca.RUnlock()
	return
}

func (ca *ca) newPendingChannel() (channel int) {
	ca.bifMtx.Lock()
	defer ca.bifMtx.Unlock()

	ca.bifPdId++
	channel = ca.bifPdId
	ca.bifPdChans[channel] = newPdChan(3)
	return
}

func (ca *ca) deletePendingChannel(channel int) {
	ca.bifMtx.RLock()
	ch, ok := ca.bifPdChans[channel]
	ca.bifMtx.RUnlock()
	if !ok {
		return
	}

	ch.mtx.Lock()
	ch.end = true
	ch.mtx.Unlock()
}

func (c *UDPConn) DeletePendingChannel(channel int) {
	c.ca.deletePendingChannel(channel)
}

func (c *UDPConn) NewPendingChannel() (channel int) {
	return c.ca.newPendingChannel()
}

func (ca *ca) addToPendingChannel(channel int, m *msg.UDPMessage) {
	ca.bifMtx.RLock()
	ch, ok := ca.bifPdChans[channel]
	ca.bifMtx.RUnlock()
	if !ok {
		panic(fmt.Errorf("no channel %d", channel))
	}

	ch.mtx.Lock()
	for ch.pd.Len() >= ch.maxPd {
		ch.cond.Wait()
	}
	ch.seq++
	m.SetChannelSeq(channel, ch.seq)
	atomic.AddInt32(&ca.pendingCnt, 1)
	ch.pd.ReplaceOrInsert(m)
	ch.mtx.Unlock()
}

func (ca *ca) addToResendChannel(m *msg.UDPMessage) {
	ca.resendChan.mtx.Lock()
	ca.resendChan.pd.ReplaceOrInsert(m)
	ca.resendChan.mtx.Unlock()
}

func (ca *ca) popMessage() (m *msg.UDPMessage) {
	ca.resendChan.mtx.Lock()
	for {
		element := ca.resendChan.pd.Min()
		if element == nil {
			break
		}
		m = element.(*msg.UDPMessage)
		ca.resendChan.pd.DeleteMin()
		if m.IsAcked() {
			m = nil
			continue
		}
		ca.resendChan.mtx.Unlock()
		return
	}
	ca.resendChan.mtx.Unlock()

	ca.cwndMtx.Lock()
	defer ca.cwndMtx.Unlock()
	if ca.cwnd < ca.usedCwnd+1 {
		logrus.Debugf("popMessage cwnd %d used %d", ca.cwnd, ca.usedCwnd)
		return
	}

	ca.bifMtx.Lock()
	defer ca.bifMtx.Unlock()
	defer ca.gcChannel()
OUT:
	for _, v := range ca.bifPdChans {
		v.mtx.Lock()
		pd := v.pd
		if pd.Len() < 1 {
			v.mtx.Unlock()
			continue
		}
		for {
			element := pd.Min()
			if element == nil {
				v.mtx.Unlock()
				continue OUT
			}
			m = element.(*msg.UDPMessage)
			pd.DeleteMin()
			if m.IsAcked() {
				m = nil
				continue
			}
			break
		}

		ca.usedCwnd++
		v.mtx.Unlock()
		v.cond.Broadcast()
		atomic.AddInt32(&ca.pendingCnt, -1)

		ca.bif += m.PkgBytesLen()
		return
	}

	return
}

func (ca *ca) gcChannel() {
	var ids []int
	for id, v := range ca.bifPdChans {
		v.mtx.Lock()
		pd := v.pd
		if pd.Len() == 0 && v.end {
			ids = append(ids, id)
		}
		v.mtx.Unlock()
	}
	for _, id := range ids {
		delete(ca.bifPdChans, id)
	}
}

func (ca *ca) getBytesInFlight() (r int) {
	ca.bifMtx.RLock()
	r = ca.bif
	ca.bifMtx.RUnlock()
	return
}

func (ca *ca) getCwnd() (cwnd uint32) {
	ca.cwndMtx.Lock()
	cwnd = ca.cwnd
	ca.cwndMtx.Unlock()
	return
}

func (ca *ca) getUsedCwnd() (cwnd uint32) {
	ca.cwndMtx.Lock()
	cwnd = ca.usedCwnd
	ca.cwndMtx.Unlock()
	return
}

func (ca *ca) isCwndFull() (r bool) {
	ca.cwndMtx.Lock()
	r = ca.usedCwnd >= ca.cwnd
	ca.cwndMtx.Unlock()
	return
}

func (ca *ca) setCwnd(cwnd uint32) {
	if cwnd < 4 {
		cwnd = 4
	} else if cwnd > 200 {
		cwnd = 200
	}

	ca.cwndMtx.Lock()
	ca.cwnd = cwnd
	ca.cwndMtx.Unlock()
}

func (ca *ca) getPacingRate() uint64 {
	return atomic.LoadUint64(&ca.pacingRate)
}

func (ca *ca) setPacingRate(rate uint64) {
	atomic.StoreUint64(&ca.pacingRate, rate)
}

func (ca *ca) calcPacingTime(len int) (d time.Duration) {
	d = time.Duration(uint64(len*1000000000) / ca.getPacingRate())
	r := time.Now().Add(d)
	logrus.Debugf("calcPacingTime %s", d)
	ca.nextPacingTime = r
	return
}

func (ca *ca) isPacingTime() (r bool) {
	r = !time.Now().Before(ca.nextPacingTime)
	logrus.Debugf("nextPacingTime %s %t", ca.nextPacingTime, r)
	return
}

func (ca *ca) checkAppLimited(seq uint32) {
	pd := atomic.LoadInt32(&ca.pendingCnt)
	if pd > 0 {
		return
	}
	if ca.isCwndFull() {
		return
	}
	ca.setAppLimited(seq)
}

func (ca *ca) tryToCancelAppLimited(seq uint32) {
	if ca.appLimited && seq > ca.endOfLimited {
		ca.appLimited = false
	}
}

func (ca *ca) setAppLimited(seq uint32) {
	ca.Lock()
	ca.appLimited = true
	ca.endOfLimited = seq
	ca.Unlock()
}

func (ca *ca) isAppLimited() (r bool) {
	r = ca.appLimited
	return
}

func (ca *ca) fullBwReached() bool {
	return ca.fullBwCnt >= fullBwCnt
}

func (ca *ca) checkFullBwReached() {
	if ca.fullBwReached() || ca.isAppLimited() {
		return
	}

	bwt := ca.fullBw * fullBwThresh >> BBR_SCALE
	max := ca.bwFilter.GetBest()
	if max >= bwt {
		ca.fullBw = max
		ca.fullBwCnt = 0
		return
	}
	ca.fullBwCnt++
}

func (ca *ca) checkDrain(bw, rtt uint64) {
	if ca.mode == startup && ca.fullBwReached() {
		ca.mode = drain
		ca.pacingGain = drainGain
		ca.cwndGain = highGain
	}
	if ca.mode == drain {
		pcwnd := ca.targetCwnd(bw, rtt, BBR_UNIT)
		if ca.getUsedCwnd() <= pcwnd {
			ca.mode = probeBW
			ca.cwndGain = cwndGain
			ca.pacingGain = BBR_UNIT
		}
	}
}

func (ca *ca) updateRoundTripCounter(seq uint32) bool {
	ca.roundTripMutex.Lock()
	defer ca.roundTripMutex.Unlock()
	if seq > ca.currentTripEnd {
		ca.roundTripCount++
		ca.currentTripEnd = ca.lastSentSeq
		return true
	}
	return false
}

func (ca *ca) updateLastSentSeq(seq uint32) {
	ca.roundTripMutex.Lock()
	ca.lastSentSeq = seq
	ca.roundTripMutex.Unlock()
}

func (ca *ca) getRoundTripCount() (c roundTripCount) {
	ca.roundTripMutex.RLock()
	c = ca.roundTripCount
	ca.roundTripMutex.RUnlock()
	return
}

func (ca *ca) targetCwnd(bw, rtt uint64, gain int) uint32 {
	cwnd := uint32((((bw * rtt * uint64(ca.cwndGain)) >> BBR_SCALE) + BW_UNIT - 1) / BW_UNIT)
	cwnd = (cwnd + 1) & ^uint32(1)
	return cwnd
}

func (ca *ca) updateGainCyclePhase(bw, rtt uint64) {
	b := time.Now().Sub(ca.lastCycleStart) > time.Duration(ca.rttSamples.getMin())
	if ca.pacingGain > BBR_UNIT && ca.getUsedCwnd() < ca.targetCwnd(bw, rtt, ca.pacingGain) {
		b = false
	}

	if ca.pacingGain < BBR_UNIT && ca.getUsedCwnd() <= ca.targetCwnd(bw, rtt, BBR_UNIT) {
		b = true
	}

	if b {
		ca.cycleOffset = (ca.cycleOffset + 1) % gainCycleLength
		ca.lastCycleStart = time.Now()
		ca.pacingGain = pacingGain[ca.cycleOffset]
	}
}
