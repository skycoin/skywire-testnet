package conn

import (
	"sync"

	"github.com/google/btree"
	"github.com/sirupsen/logrus"
	"github.com/skycoin/skywire/pkg/net/msg"
)

type streamQueue interface {
	Push(k uint32, m *msg.UDPMessage) (ok bool, msgs []*msg.UDPMessage)
	Len() (s int)
	GetNextAckSeq() (s uint32)
	GetAckedSeqs(start, end uint32) (mask uint32)
}

type packet struct {
	seq  uint32
	data *msg.UDPMessage
}

func (a packet) Less(b btree.Item) bool {
	return a.seq < b.(packet).seq
}

type fecStreamQueue struct {
	dataShards   uint32
	parityShards uint32
	shardSize    uint32

	ackedSeq uint32
	msgs     *btree.BTree
	mutex    sync.RWMutex
}

func newFECStreamQueue(dataShards, parityShards uint32) *fecStreamQueue {
	return &fecStreamQueue{
		dataShards:   dataShards,
		parityShards: parityShards,
		shardSize:    dataShards + parityShards,
		msgs:         btree.New(2),
	}
}

func (q *fecStreamQueue) _getDataShardSeq(seq uint32) (s uint32) {
	ss := seq - 1
	i := ss / q.shardSize
	j := ss % q.shardSize
	if j >= q.dataShards {
		s = (i+1)*q.shardSize + 1
	} else {
		s = seq
	}
	return
}

func (q *fecStreamQueue) _getNextAckSeq() (s uint32) {
	return q._getDataShardSeq(q.ackedSeq + 1)
}

func (q *fecStreamQueue) Push(k uint32, m *msg.UDPMessage) (ok bool, msgs []*msg.UDPMessage) {
	if (k-1)%q.shardSize >= q.dataShards {
		return
	}
	defer func() {
		logrus.Debugf("fecStreamQueue return %t, len %d, push k %d, next %d ", ok, len(msgs), k, q._getNextAckSeq())
	}()
	q.mutex.Lock()
	defer q.mutex.Unlock()
	if k <= q.ackedSeq {
		return
	}
	if k == q._getNextAckSeq() {
		ok = true
		if q.msgs.Len() < 1 {
			msgs = []*msg.UDPMessage{m}
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

func (q *fecStreamQueue) pop() (msgs []*msg.UDPMessage) {
	for i := q._getNextAckSeq(); ; i = q._getNextAckSeq() {
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

func (q *fecStreamQueue) push(k uint32, m *msg.UDPMessage) {
	q.msgs.ReplaceOrInsert(packet{
		seq:  k,
		data: m,
	})
}

func (q *fecStreamQueue) Len() (s int) {
	q.mutex.RLock()
	s = q.msgs.Len()
	q.mutex.RUnlock()
	return
}

func (q *fecStreamQueue) GetNextAckSeq() (s uint32) {
	q.mutex.RLock()
	s = q._getNextAckSeq()
	q.mutex.RUnlock()
	return
}

func (q *fecStreamQueue) GetAckedSeqs(start, end uint32) (mask uint32) {
	if end-start > 32 {
		end = start + 32
	}
	q.mutex.RLock()
	defer q.mutex.RUnlock()
	q.msgs.AscendRange(packet{seq: start}, packet{seq: end}, func(i btree.Item) bool {
		p, ok := i.(packet)
		if !ok {
			return true
		}
		mask |= 1 << (p.seq - start)
		return true
	})
	return
}
