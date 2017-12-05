package conn

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/google/btree"
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
	ConnCommonFields
	*UDPPendingMap
	*streamQueue
	UdpConn *net.UDPConn
	addr    *net.UDPAddr

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
}

type mode int

const (
	startup mode = iota
	drain
	probeBW
)

// used for server spawn udp conn
func NewUDPConn(c *net.UDPConn, addr *net.UDPAddr) *UDPConn {
	conn := &UDPConn{
		UdpConn:          c,
		addr:             addr,
		ConnCommonFields: NewConnCommonFileds(),
		UDPPendingMap:    NewUDPPendingMap(),
		streamQueue:      newStreamQueue(),
		rto:              300 * time.Millisecond,

		ca: newCA(),
	}
	conn.lastAckCond = sync.NewCond(&conn.lastAckMtx)
	go conn.ackLoop()
	return conn
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
				c.GetContextLogger().Debug("udp conn closed")
				return nil
			}
			err := c.Write(m)
			if err != nil {
				c.GetContextLogger().Debugf("write msg is failed %v", err)
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
				c.GetContextLogger().Debug("udp conn closed")
				return nil
			}
			err := c.Write(m)
			if err != nil {
				c.GetContextLogger().Debugf("write msg is failed %v", err)
				return err
			}
		}
	}
}

func (c *UDPConn) ackLoop() (err error) {
	t := time.NewTimer(5 * time.Millisecond)
	defer func() {
		if !t.Stop() {
			<-t.C
		}
		if err != nil {
			c.SetStatusToError(err)
		}
	}()

	for {
		select {
		case <-t.C:
			la := atomic.LoadUint32(&c.lastAck)
			lt := atomic.LoadUint32(&c.lastCnt)
			if lt != c.lastCnted {
				err = c.ack(la)
				if err != nil {
					return
				}
				c.lastCnted = lt
			} else {
				if !t.Stop() {
					<-t.C
				}
				c.lastAckMtx.Lock()
				c.lastAckCond.Wait()
				c.lastAckMtx.Unlock()
				t.Reset(5 * time.Millisecond)
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
	m := msg.NewUDPWithoutSeq(msg.TYPE_NORMAL, bytes)
	ok := c.addToPendingChannel(channel, m)
	c.GetContextLogger().Debugf("bif %d, ok %t", c.ca.getBytesInFlight(), ok)
	if !ok {
		return nil
	}
	m.SetSeq(c.GetNextSeq())
	c.GetContextLogger().Debugf("new msg seq %d", m.GetSeq())
	err = c.WriteBytes(m.PkgBytes())
	c.transmitted(m)
	return
}

func (c *UDPConn) transmitted(m *msg.UDPMessage) {
	seq := m.GetSeq()
	c.ca.checkAppLimited(seq)
	c.addMsg(seq, m)
	m.Transmitted()
	m.SetRTO(c.getRTO(), func() (err error) {
		c.AddRTOResendCount()
		err = c.resendMsg(m)
		if err != nil {
			c.SetStatusToError(err)
			c.Close()
		}
		return
	})
	m.UpdateState(c.getDelivered(), c.getDeliveredTime(), c.getSentTime())
}

func (c *UDPConn) resendMsg(m *msg.UDPMessage) (err error) {
	if !c.UDPPendingMap.exists(m.GetSeq()) {
		return
	}
	c.GetContextLogger().Debugf("resendMsg %s", m)
	err = c.WriteBytes(m.PkgBytes())
	return
}

func (c *UDPConn) writePendingMsgs(s int) error {
	for {
		c.GetContextLogger().Debugf("popMessage bif %d, s %d", c.ca.getBytesInFlight(), s)
		m := c.ca.popMessage(s)
		s = 0
		c.GetContextLogger().Debugf("popMessage bif %d, m %v", c.ca.getBytesInFlight(), m)
		if m == nil {
			return nil
		}
		m.SetSeq(c.GetNextSeq())
		c.GetContextLogger().Debugf("new msg seq %d", m.GetSeq())
		err := c.WriteBytes(m.PkgBytes())
		if err != nil {
			return err
		}
		c.transmitted(m)
	}
}

func (c *UDPConn) WriteBytes(bytes []byte) error {
	l := len(bytes)
	c.AddSentBytes(l)
	c.WriteMutex.Lock()
	defer c.WriteMutex.Unlock()
	n, err := c.UdpConn.WriteToUDP(bytes, c.addr)
	c.GetContextLogger().Debugf("write out %x", bytes)
	if err == nil && n != l {
		return errors.New("nothing was written")
	}
	return err
}

func (c *UDPConn) Ack(seq uint32) error {
	atomic.StoreUint32(&c.lastAck, seq)
	atomic.AddUint32(&c.lastCnt, 1)
	c.lastAckCond.Broadcast()
	return nil
}

func (c *UDPConn) ack(seq uint32) error {
	nSeq := c.getNextAckSeq()
	c.GetContextLogger().Debugf("ack %d, next %d", seq, nSeq)
	var missing []uint32
	var ml int
	if seq > nSeq+1 {
		missing = c.getMissingSeqs(nSeq+1, seq)
		c.GetContextLogger().Debugf("missing %v", missing)
		ml = len(missing)
	}
	p := make([]byte, msg.ACK_HEADER_SIZE+msg.PKG_HEADER_SIZE+4*ml)
	m := p[msg.PKG_HEADER_SIZE:]
	m[msg.ACK_TYPE_BEGIN] = msg.TYPE_ACK
	binary.BigEndian.PutUint32(m[msg.ACK_SEQ_BEGIN:], seq)
	binary.BigEndian.PutUint32(m[msg.ACK_NEXT_SEQ_BEGIN:], nSeq)

	for i, v := range missing {
		binary.BigEndian.PutUint32(m[msg.ACK_NEXT_SEQ_END+i*4:], v)
	}

	checksum := crc32.ChecksumIEEE(m)
	binary.BigEndian.PutUint32(p[msg.PKG_CRC32_BEGIN:], checksum)
	return c.WriteBytes(p)
}

func (c *UDPConn) RecvAck(m []byte) (err error) {
	if len(m) < msg.ACK_HEADER_SIZE {
		return fmt.Errorf("invalid ack msg %x", m)
	}
	seq := binary.BigEndian.Uint32(m[msg.ACK_SEQ_BEGIN:msg.ACK_SEQ_END])
	ns := binary.BigEndian.Uint32(m[msg.ACK_NEXT_SEQ_BEGIN:msg.ACK_NEXT_SEQ_END])

	c.GetContextLogger().Debugf("recv ack %d, next %d", seq, ns)
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

	if seq > ns+1 {
		i := msg.ACK_NEXT_SEQ_END
		mm := make(map[uint32]struct{})
		for len(m)-i >= 4 {
			v := binary.BigEndian.Uint32(m[i:])
			mm[v] = struct{}{}
			i = i + 4
		}
		c.GetContextLogger().Debugf("recover ack [%d-%d) missing %v", ns+1, seq, mm)

		for j := ns + 1; j < seq; j++ {
			if _, ok := mm[j]; !ok {
				err = c.delMsg(j, true)
				if err != nil {
					return
				}
			}
		}
	}

	return
}

func (c *UDPConn) Ping() error {
	c.GetContextLogger().Debug("ping")
	p := make([]byte, msg.PING_MSG_HEADER_SIZE+msg.PKG_HEADER_SIZE)
	m := p[msg.PKG_HEADER_SIZE:]
	m[msg.PING_MSG_TYPE_BEGIN] = msg.TYPE_PING
	binary.BigEndian.PutUint64(m[msg.PING_MSG_TIME_BEGIN:], msg.UnixMillisecond())
	checksum := crc32.ChecksumIEEE(m)
	binary.BigEndian.PutUint32(p[msg.PKG_CRC32_BEGIN:], checksum)
	return c.WriteBytes(p)
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

func (c *UDPConn) getRTO() (rto time.Duration) {
	c.FieldsMutex.RLock()
	rto = c.rto
	c.FieldsMutex.RUnlock()
	return
}

func (c *UDPConn) setRTO(rto time.Duration) {
	c.GetContextLogger().Debugf("setRTO %d", rto)
	if rto < 100*time.Millisecond {
		rto = 100 * time.Millisecond
	}
	c.FieldsMutex.Lock()
	c.rto = rto
	c.FieldsMutex.Unlock()
}

func (c *UDPConn) addMsg(k uint32, v *msg.UDPMessage) {
	c.UDPPendingMap.AddMsg(k, v)
}

func (c *UDPConn) delMsg(seq uint32, ignore bool) error {
	ok, um, msgs := c.DelMsgAndGetLossMsgs(seq, 3)
	if ok {
		c.AddAckCount()
		if !ignore {
			c.updateRTT(um.GetRTT())
		}
		c.updateDeliveryRate(um)
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
		c.UpdateLastAck(seq)
		return c.writePendingMsgs(um.PkgBytesLen())
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
			c.setRTO(t * 3)
		}
		break
	}
}

type rateSampler struct {
	tree  *btree.BTree
	ring  []rate
	mask  int
	index int
}

type rate uint64

func (a rate) Less(b btree.Item) bool {
	return a < b.(rate)
}

// size should be power of 2
func newRateSampler(size int) *rateSampler {
	if size < 2 || (size&(size-1)) > 0 {
		var n uint
		for size > 0 {
			size >>= 1
			n++
		}
		size = 1 << n
	}
	return &rateSampler{
		ring: make([]rate, size),
		mask: size - 1,
		tree: btree.New(2),
	}
}

func (t *rateSampler) push(r rate) rate {
	if r <= 0 {
		panic("push rate <= 0")
	}
	or := t.ring[t.index]
	if or > 0 {
		t.tree.Delete(or)
	}
	t.ring[t.index] = r
	t.tree.ReplaceOrInsert(r)
	t.index = (t.index + 1) & t.mask
	return t.tree.Max().(rate)
}

func (t *rateSampler) getMax() rate {
	item := t.tree.Max()
	if item == nil {
		return 0
	}
	return item.(rate)
}

func (c *UDPConn) updateDeliveryRate(m *msg.UDPMessage) {
	c.ca.Lock()
	defer c.ca.Unlock()
	c.delivered += uint64(m.TotalSize())
	c.deliveredTime = time.Now()
	c.sentTime = m.GetTransmittedTime()

	if m.GetSentTime().IsZero() || m.GetDeliveredTime().IsZero() {
		return
	}

	c.tryToCancelAppLimited(m.GetSeq())

	sd := c.sentTime.Sub(m.GetSentTime()) / time.Millisecond
	ad := c.deliveredTime.Sub(m.GetDeliveredTime()) / time.Millisecond
	interval := ad
	if sd > ad {
		interval = sd
	}
	d := c.delivered - m.GetDelivered()
	drate := rate(d / uint64(interval))
	c.GetContextLogger().Debugf("drate(%d) d %d interval %d sd %d ad %d", drate, d, interval, sd, ad)
	mr := c.rttSamples.getMin()
	if mr <= 0 {
		return
	}
	rtt := uint64(time.Duration(mr) / time.Millisecond)
	if uint64(interval) < rtt {
		return
	}
	if drate <= 0 {
		return
	}

	if c.isAppLimited() {
		return
	}
	max := uint64(c.rateSamples.push(drate))
	hm := c.rateSamples.getMax()
	if hm <= 0 {
		return
	}
	cwnd := uint32(float64(max*rtt) * c.pacingGain)
	if c.ca.mode == startup {
		if hm > drate {
			c.fullCnt++
			if c.fullCnt > 3 {
				c.ca.mode = drain
				c.ca.pacingGain = drainGain
			}
		}
	}
	c.GetContextLogger().Debugf("mode %d, max bw %d rtt %d: cwnd %d", c.mode, max, rtt, cwnd)
	if c.ca.mode == startup && cwnd > c.cwnd {
		//c.cwnd = cwnd
	} else if c.ca.mode == drain {
		pcwnd := uint32(max * rtt)
		if c.ca.bif < int(pcwnd) {
			c.ca.mode = probeBW
			c.ca.pacingGain = 1
			cwnd = pcwnd
		}
		//c.cwnd = cwnd
	}
}

type ca struct {
	delivered     uint64
	deliveredTime time.Time
	sentTime      time.Time
	rttSamples    *rttSampler
	rateSamples   *rateSampler
	cwnd          uint32
	mode
	pacingGain float64
	fullCnt    uint
	pendingCnt int32

	bif        int
	bifMtx     sync.RWMutex
	bifPdId    int
	bifPdChans map[int]*pdChan

	appLimited   bool
	endOfLimited uint32

	sync.RWMutex
}

type pdChan struct {
	pd    *btree.BTree
	seq   uint32
	mtx   sync.Mutex
	cond  *sync.Cond
	maxPd int
}

func newPdChan(max int) *pdChan {
	pd := &pdChan{
		pd:    btree.New(2),
		maxPd: max,
	}
	pd.cond = sync.NewCond(&pd.mtx)
	return pd
}

func newCA() *ca {
	c := &ca{
		rttSamples:  newRttSampler(16),
		rateSamples: newRateSampler(16),
		pacingGain:  highGain,
		cwnd:        20480,
		bifPdChans:  make(map[int]*pdChan),
	}

	c.bifPdChans[c.bifPdId] = newPdChan(1000)
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
	ca.bifPdChans[channel] = newPdChan(100)
	return
}

func (ca *ca) deletePendingChannel(channel int) {
	ca.bifMtx.Lock()
	defer ca.bifMtx.Unlock()

	delete(ca.bifPdChans, channel)
}

func (c *UDPConn) DeletePendingChannel(channel int) {
	c.ca.deletePendingChannel(channel)
}

func (c *UDPConn) NewPendingChannel() (channel int) {
	return c.ca.newPendingChannel()
}

func (ca *ca) addToPendingChannel(channel int, m *msg.UDPMessage) bool {
	ca.bifMtx.RLock()
	ch, ok := ca.bifPdChans[channel]
	ca.bifMtx.RUnlock()
	if !ok {
		panic(fmt.Errorf("no channel %d", channel))
	}

	ch.mtx.Lock()
	if ch.pd.Len()+1 > ch.maxPd && channel != 0 {
		ch.cond.Wait()
	}
	ch.seq++
	m.SetSeq(ch.seq)
	min := ch.pd.Min()
	if (min != nil && min.Less(m)) || int(ca.cwnd) < ca.bif+m.PkgBytesLen() {
		atomic.AddInt32(&ca.pendingCnt, 1)
		ch.pd.ReplaceOrInsert(m)
		ch.mtx.Unlock()
		return false
	}
	ch.mtx.Unlock()

	ca.bifMtx.Lock()
	ca.bif += m.PkgBytesLen()
	ca.bifMtx.Unlock()
	return true
}

func (ca *ca) popMessage(s int) (m *msg.UDPMessage) {
	ca.bifMtx.Lock()
	if ca.bif < s {
		panic(fmt.Errorf("popMessage ca.bif(%d) < s(%d)", ca.bif, s))
	}
	ca.bif -= s
	ca.bifMtx.Unlock()

	for _, v := range ca.bifPdChans {
		v.mtx.Lock()
		pd := v.pd
		if pd.Len() < 1 {
			v.mtx.Unlock()
			continue
		}
		element := pd.Min()
		if element == nil {
			v.mtx.Unlock()
			continue
		}
		m = element.(*msg.UDPMessage)

		if int(ca.cwnd) < ca.bif+m.PkgBytesLen() {
			m = nil
			v.mtx.Unlock()
			continue
		}
		pd.DeleteMin()
		v.mtx.Unlock()
		v.cond.Broadcast()
		atomic.AddInt32(&ca.pendingCnt, -1)

		ca.bifMtx.Lock()
		ca.bif += m.PkgBytesLen()
		ca.bifMtx.Unlock()
		return
	}
	return
}

func (ca *ca) getBytesInFlight() (r int) {
	ca.bifMtx.RLock()
	r = ca.bif
	ca.bifMtx.RUnlock()
	return
}

func (ca *ca) getCwnd() (r uint32) {
	ca.RLock()
	r = ca.cwnd
	ca.RUnlock()
	return
}

func (ca *ca) setCwnd(cwnd uint32) {
	if cwnd < MAX_UDP_PACKAGE_SIZE {
		return
	}
	ca.cwnd = cwnd
}

func (ca *ca) checkAppLimited(seq uint32) {
	pd := atomic.LoadInt32(&ca.pendingCnt)
	if pd == 0 {
		ca.setAppLimited(seq)
	}
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
