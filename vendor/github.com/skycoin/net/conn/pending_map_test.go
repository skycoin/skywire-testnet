package conn

import (
	"github.com/skycoin/net/msg"
	"testing"
)

func newUdp(seq uint32) *msg.UDPMessage {
	return msg.NewUDP(1, seq, []byte{byte(seq)})
}

func TestNewUDPPendingMap(t *testing.T) {
	m := NewUDPPendingMap()
	m.AddMsg(1, newUdp(1))
	m.AddMsg(2, newUdp(2))
	m.AddMsg(3, newUdp(3))
	m.AddMsg(4, newUdp(4))
	m.AddMsg(5, newUdp(5))

	t.Log(m.DelMsgAndGetLossMsgs(1, 3))
	//t.Log(m.DelMsgAndGetLossMsgs(3))
	t.Log(m.DelMsgAndGetLossMsgs(4, 3))
	t.Log(m.DelMsgAndGetLossMsgs(5, 3))
	m.AddMsg(6, newUdp(6))
	t.Log(m.DelMsgAndGetLossMsgs(3, 3))
	m.AddMsg(7, newUdp(7))
	t.Log(m.DelMsgAndGetLossMsgs(6, 3))
	m.AddMsg(8, newUdp(8))
	m.AddMsg(9, newUdp(9))
	t.Log(m.DelMsgAndGetLossMsgs(8, 3))
	t.Log(m.DelMsgAndGetLossMsgs(9, 3))
}

func TestStreamQueue_Push(t *testing.T) {
	q := newStreamQueue()
	t.Log(q.Push(1, []byte{0x60}))
	t.Log(q.Push(1, []byte{0x60}))
	t.Log(q.Push(2, []byte{0x61}))
	t.Log(q.Push(4, []byte{0x63}))
	t.Log(q.Push(3, []byte{0x62}))
	t.Log(q.Push(7, []byte{0x66}))
	t.Log(q.Push(5, []byte{0x64}))
	t.Log(q.Push(6, []byte{0x65}))
}
