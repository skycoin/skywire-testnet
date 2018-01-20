package conn

import (
	"fmt"
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

func (m *UDPPendingMap) DelMsgAndGetLossMsgs(k uint32) (ok bool, um *msg.UDPMessage, loss []*msg.UDPMessage) {
	m.Lock()
	v, ok := m.Pending[k]
	if !ok {
		m.Unlock()
		return
	}
	um = v.(*msg.UDPMessage)
	um.Acked()
	delete(m.Pending, k)

	m.seqs.Delete(seq(k))
	m.seqs.AscendLessThan(seq(k), func(i btree.Item) bool {
		v, ok := m.Pending[uint32(i.(seq))]
		if ok {
			v, ok := v.(*msg.UDPMessage)
			if ok {
				miss := v.AddMiss()
				x := miss / QUICK_LOST_THRESH
				y := miss % QUICK_LOST_THRESH
				if x > 0 && x < QUICK_LOST_RESEND_COUNT && y == 0 {
					loss = append(loss, v)
				}
			}
		}
		return true
	})
	m.Unlock()

	m.ackedMessagesMutex.Lock()
	m.ackedMessages[k] = um
	m.ackedMessagesMutex.Unlock()

	return
}
