package conn

import (
	"github.com/google/btree"
	"github.com/sirupsen/logrus"
	"github.com/skycoin/net/msg"
	"sync"
)

type streamQueue interface {
	Push(k uint32, m *msg.UDPMessage) (ok bool, msgs []*msg.UDPMessage)
	Len() (s int)
	GetNextAckSeq() (s uint32)
	GetMissingSeqs(start, end uint32) (seqs []uint32)
}

type defaultStreamQueue struct {
	ackedSeq uint32
	msgs     *btree.BTree
	mutex    sync.RWMutex
}

func newStreamQueue() *defaultStreamQueue {
	return &defaultStreamQueue{
		msgs: btree.New(2),
	}
}

type packet struct {
	seq  uint32
	data *msg.UDPMessage
}

func (a packet) Less(b btree.Item) bool {
	return a.seq < b.(packet).seq
}

func (q *defaultStreamQueue) Push(k uint32, m *msg.UDPMessage) (ok bool, msgs []*msg.UDPMessage) {
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

func (q *defaultStreamQueue) pop() (msgs []*msg.UDPMessage) {
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

func (q *defaultStreamQueue) push(k uint32, m *msg.UDPMessage) {
	q.msgs.ReplaceOrInsert(packet{
		seq:  k,
		data: m,
	})
}

func (q *defaultStreamQueue) Len() (s int) {
	q.mutex.RLock()
	s = q.msgs.Len()
	q.mutex.RUnlock()
	return
}

func (q *defaultStreamQueue) GetNextAckSeq() (s uint32) {
	q.mutex.RLock()
	s = q.ackedSeq + 1
	q.mutex.RUnlock()
	return
}

func (q *defaultStreamQueue) GetMissingSeqs(start, end uint32) (seqs []uint32) {
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
		logrus.Debugf("fecStreamQueue push k %d return %t, len %d", k, ok, len(msgs))
	}()
	q.mutex.Lock()
	defer q.mutex.Unlock()
	if k <= q.ackedSeq {
		return
	}
	ok = true
	if k == q._getNextAckSeq() {
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

func (q *fecStreamQueue) GetMissingSeqs(start, end uint32) (seqs []uint32) {
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

	for i := q._getDataShardSeq(start); i < end; i = q._getDataShardSeq(i + 1) {
		if _, ok := e[i]; !ok {
			seqs = append(seqs, i)
		}
	}
	return
}
