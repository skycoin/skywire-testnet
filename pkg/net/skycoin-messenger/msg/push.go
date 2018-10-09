package msg

import "sync"

type Reg struct {
	PublicKey string
}

type PushMsg struct {
	From string
	Msg  string
}

var pool = &sync.Pool{
	New: func() interface{} {
		return new(PushMsg)
	},
}

// Get `PushMsg` from pool
func GetPushMsg(from, msg string) (p *PushMsg) {
	p = pool.Get().(*PushMsg)
	p.From = from
	p.Msg = msg
	return
}

// Put `PushMsg` back to the pool
func PutPushMsg(p interface{}) {
	pool.Put(p)
}