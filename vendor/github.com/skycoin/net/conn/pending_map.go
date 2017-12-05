package conn

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"math/big"
	"sync"
	"time"

	"github.com/google/btree"
	"github.com/skycoin/net/msg"
)

type PendingMap struct {
	Pending map[uint32]msg.Interface
	sync.RWMutex
	ackedMessages        map[uint32]msg.Interface
	ackedMessagesMutex   sync.RWMutex
	lastMinuteAcked      map[uint32]msg.Interface
	lastMinuteAckedMutex sync.RWMutex

	statistics string
}

func NewPendingMap() *PendingMap {
	pendingMap := &PendingMap{Pending: make(map[uint32]msg.Interface), ackedMessages: make(map[uint32]msg.Interface)}
	go pendingMap.analyse()
	return pendingMap
}

func (m *PendingMap) AddMsg(k uint32, v *msg.Message) {
	m.Lock()
	m.Pending[k] = v
	m.Unlock()
	v.Transmitted()
}

func (m *PendingMap) DelMsg(k uint32) (ok bool) {
	m.RLock()
	v, ok := m.Pending[k]
	m.RUnlock()

	if !ok {
		return
	}

	v.Acked()

	m.ackedMessagesMutex.Lock()
	m.ackedMessages[k] = v
	m.ackedMessagesMutex.Unlock()

	m.Lock()
	delete(m.Pending, k)
	m.Unlock()
	return
}

func (m *PendingMap) analyse() {
	ticker := time.NewTicker(time.Minute)
	for {
		select {
		case <-ticker.C:
			m.ackedMessagesMutex.Lock()
			m.lastMinuteAckedMutex.Lock()
			m.lastMinuteAcked = m.ackedMessages
			m.lastMinuteAckedMutex.Unlock()
			m.ackedMessages = make(map[uint32]msg.Interface)
			m.ackedMessagesMutex.Unlock()

			m.lastMinuteAckedMutex.RLock()
			if len(m.lastMinuteAcked) < 1 {
				m.lastMinuteAckedMutex.RUnlock()
				continue
			}
			var max, min int64
			sum := new(big.Int)
			bytesSent := 0
			for _, v := range m.lastMinuteAcked {
				latency := v.GetRTT().Nanoseconds()
				if max < latency {
					max = latency
				}
				if min == 0 || min > latency {
					min = latency
				}
				y := new(big.Int)
				y.SetInt64(latency)
				sum.Add(sum, y)

				bytesSent += v.TotalSize()
			}
			n := new(big.Int)
			n.SetInt64(int64(len(m.lastMinuteAcked)))
			avg := new(big.Int)
			avg.Div(sum, n)
			m.lastMinuteAckedMutex.RUnlock()

			m.statistics = fmt.Sprintf("sent: %d bytes, latency: max %d ns, min %d ns, avg %s ns, count %s", bytesSent, max, min, avg, n)
		}
	}
}

type UDPPendingMap struct {
	*PendingMap
	seqs *btree.BTree
}

type seq uint32

func (a seq) Less(b btree.Item) bool {
	return a < b.(seq)
}

func NewUDPPendingMap() *UDPPendingMap {
	m := &UDPPendingMap{
		PendingMap: NewPendingMap(),
		seqs:       btree.New(2),
	}
	return m
}

func (m *UDPPendingMap) AddMsg(k uint32, v msg.Interface) {
	m.Lock()
	m.Pending[k] = v
	m.seqs.ReplaceOrInsert(seq(k))
	m.Unlock()
}

func (m *UDPPendingMap) getMinUnAckSeq() (s uint32, ok bool) {
	m.RLock()
	r, ok := m.seqs.Min().(seq)
	if !ok {
		m.RUnlock()
		return
	}
	s = uint32(r)
	m.RUnlock()
	return
}

func (m *UDPPendingMap) exists(k uint32) (ok bool) {
	m.RLock()
	_, ok = m.Pending[k]
	m.RUnlock()
	return
}

func (m *UDPPendingMap) DelMsgAndGetLossMsgs(k uint32, resend uint32) (ok bool, um *msg.UDPMessage, loss []*msg.UDPMessage) {
	m.Lock()
	v, ok := m.Pending[k]
	if !ok {
		m.Unlock()
		return
	}
	um = v.(*msg.UDPMessage)
	um.Acked()
	delete(m.Pending, k)

	m.seqs.AscendLessThan(seq(k), func(i btree.Item) bool {
		v, ok := m.Pending[uint32(i.(seq))]
		if ok {
			v, ok := v.(*msg.UDPMessage)
			if ok {
				if v.AddMiss() >= resend {
					v.ResetMiss()
					loss = append(loss, v)
				}
			}
		}
		return true
	})
	m.seqs.Delete(seq(k))
	m.Unlock()

	m.ackedMessagesMutex.Lock()
	m.ackedMessages[k] = um
	m.ackedMessagesMutex.Unlock()

	return
}

type streamQueue struct {
	ackedSeq uint32
	msgs     *btree.BTree
	mutex    sync.RWMutex
}

func newStreamQueue() *streamQueue {
	return &streamQueue{
		msgs: btree.New(2),
	}
}

type packet struct {
	seq  uint32
	data []byte
}

func (a packet) Less(b btree.Item) bool {
	return a.seq < b.(packet).seq
}

func (q *streamQueue) Push(k uint32, m []byte) (ok bool, msgs [][]byte) {
	defer func() {
		logrus.Debugf("streamQueue push k %d return %t, len %d", k, ok, len(msgs))
	}()
	q.mutex.Lock()
	defer q.mutex.Unlock()
	if k <= q.ackedSeq {
		return
	}
	if k == q.ackedSeq+1 {
		ok = true
		if q.msgs.Len() < 1 {
			msgs = [][]byte{m}
			q.ackedSeq = k
			return
		}
		q.push(k, m)
		msgs = q.pop()
		return
	}
	q.push(k, m)
	return
}

func (q *streamQueue) pop() (msgs [][]byte) {
	for i := q.ackedSeq + 1; ; i++ {
		min, ok := q.msgs.Min().(packet)
		if !ok {
			break
		}
		if min.seq == i {
			msgs = append(msgs, min.data)
			q.msgs.DeleteMin()
			q.ackedSeq = i
		} else {
			break
		}
	}
	if len(msgs) < 1 {
		panic("streamQueue pop return 0 msg")
	}
	return
}

func (q *streamQueue) push(k uint32, m []byte) {
	q.msgs.ReplaceOrInsert(packet{
		seq:  k,
		data: m,
	})
}

func (q *streamQueue) getAckedSeq() (s uint32) {
	q.mutex.RLock()
	s = q.ackedSeq
	q.mutex.RUnlock()
	return
}

func (q *streamQueue) getNextAckSeq() (s uint32) {
	q.mutex.RLock()
	s = q.ackedSeq + 1
	q.mutex.RUnlock()
	return
}

func (q *streamQueue) getMissingSeqs(start, end uint32) (seqs []uint32) {
	q.mutex.RLock()
	defer q.mutex.RUnlock()
	e := make(map[uint32]struct{})
	q.msgs.AscendRange(packet{seq: start}, packet{seq: end}, func(i btree.Item) bool {
		p, ok := i.(packet)
		if !ok {
			return true
		}
		e[p.seq] = struct{}{}
		return true
	})

	for i := start; i < end; i++ {
		if _, ok := e[i]; !ok {
			seqs = append(seqs, i)
		}
	}
	return
}
