package conn

import (
	"testing"

	"github.com/skycoin/net/msg"
)

func TestNewUDPPendingMap(t *testing.T) {
	m := NewUDPPendingMap()
	m.AddMsg(1, &msg.Message{Seq: 1, Body: []byte{0x1}})
	m.AddMsg(2, &msg.Message{Seq: 2, Body: []byte{0x2}})
	m.AddMsg(3, &msg.Message{Seq: 3, Body: []byte{0x3}})
	m.AddMsg(4, &msg.Message{Seq: 4, Body: []byte{0x4}})
	m.AddMsg(5, &msg.Message{Seq: 5, Body: []byte{0x5}})

	t.Log(m.DelMsgAndGetLossMsgs(1))
	//t.Log(m.DelMsgAndGetLossMsgs(3))
	t.Log(m.DelMsgAndGetLossMsgs(4))
	t.Log(m.DelMsgAndGetLossMsgs(5))
	m.AddMsg(6, &msg.Message{Seq: 6, Body: []byte{0x5}})
	t.Log(m.DelMsgAndGetLossMsgs(3))
	m.AddMsg(7, &msg.Message{Seq: 7, Body: []byte{0x5}})
	t.Log(m.DelMsgAndGetLossMsgs(6))
	m.AddMsg(8, &msg.Message{Seq: 8, Body: []byte{0x5}})
	m.AddMsg(9, &msg.Message{Seq: 9, Body: []byte{0x5}})
	t.Log(m.DelMsgAndGetLossMsgs(8))
	t.Log(m.DelMsgAndGetLossMsgs(9))
}

func TestStreamQueue_Push(t *testing.T) {
	q := &StreamQueue{}
	t.Log(q.Push(1, []byte{0x60}))
	t.Log(q.Push(1, []byte{0x60}))
	t.Log(q.Push(2, []byte{0x61}))
	t.Log(q.Push(4, []byte{0x63}))
	t.Log(q.Push(3, []byte{0x62}))
	t.Log(q.Push(7, []byte{0x66}))
	t.Log(q.Push(5, []byte{0x64}))
	t.Log(q.Push(6, []byte{0x65}))
}
