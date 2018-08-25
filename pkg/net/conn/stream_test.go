package conn

import "testing"

func TestFecStreamQueue_Push(t *testing.T) {
	q := newFECStreamQueue(10, 3)
	t.Log(q.Push(1, []byte{0x60}))
	t.Log(q.Push(1, []byte{0x60}))
	t.Log(q.Push(2, []byte{0x61}))
	t.Log(q.Push(4, []byte{0x63}))
	t.Log(q.Push(3, []byte{0x62}))
	t.Log(q.Push(7, []byte{0x66}))
	t.Log(q.Push(5, []byte{0x64}))
	t.Log(q.Push(6, []byte{0x65}))
	t.Log(q.Push(11, []byte{0xb}))
	t.Log(q.Push(10, []byte{0xa}))
	t.Log(q.Push(9, []byte{0x9}))
	t.Log(q.Push(8, []byte{0x8}))
	t.Log(q.Push(12, []byte{0xc}))
	t.Log(q.Push(13, []byte{0xd}))
	t.Log(q.Push(14, []byte{0xe}))
	t.Log(q.Len())
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
