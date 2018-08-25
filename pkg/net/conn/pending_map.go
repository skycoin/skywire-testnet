package conn

import (
	"sync"

	"github.com/google/btree"
	"github.com/skycoin/skywire/pkg/net/msg"
)

type UDPPendingMap struct {
	pendings map[uint32]*msg.UDPMessage
	sync.RWMutex
	seqs *btree.BTree
}

type seq uint32

func (a seq) Less(b btree.Item) bool {
	return a < b.(seq)
}

func NewUDPPendingMap() *UDPPendingMap {
	m := &UDPPendingMap{
		pendings: make(map[uint32]*msg.UDPMessage),
		seqs:     btree.New(2),
	}
	return m
}

func (m *UDPPendingMap) AddMsg(k uint32, v *msg.UDPMessage) {
	m.Lock()
	m.pendings[k] = v
	m.seqs.ReplaceOrInsert(seq(k))
	m.Unlock()
}

func (m *UDPPendingMap) Dismiss() {
	m.RLock()
	for _, m := range m.pendings {
		if m == nil {
			continue
		}
		m.Cancel()
	}
	m.RUnlock()
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
	_, ok = m.pendings[k]
	m.RUnlock()
	return
}

func (m *UDPPendingMap) DelMsgAndGetLossMsgs(k uint32) (ok bool, um *msg.UDPMessage, loss []*msg.UDPMessage) {
	m.Lock()
	um, ok = m.pendings[k]
	if !ok {
		m.Unlock()
		return
	}
	um.Acked()
	delete(m.pendings, k)

	m.seqs.Delete(seq(k))
	if QUICK_LOST_ENABLE {
		m.seqs.AscendLessThan(seq(k), func(i btree.Item) bool {
			v, ok := m.pendings[uint32(i.(seq))]
			if ok {
				miss := v.AddMiss()
				x := miss / QUICK_LOST_THRESH
				y := miss % QUICK_LOST_THRESH
				if x > 0 && x <= QUICK_LOST_RESEND_COUNT && y == 0 {
					loss = append(loss, v)
				}
			}
			return true
		})
	}
	m.Unlock()
	return
}
