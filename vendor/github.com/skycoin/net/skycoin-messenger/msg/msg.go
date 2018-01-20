package msg

import (
	"sync"

	"github.com/skycoin/net/skycoin-messenger/factory"
)

var (
	OP_POOL = make([]*sync.Pool, OP_SIZE)
)

type OP interface {
	Execute(OPer) error
}

type OPer interface {
	GetFactory() *factory.MessengerFactory
	SetFactory(factory *factory.MessengerFactory)
	PushLoop(*factory.Connection)
	Push(op byte, d interface{})
}

func GetOP(opn int) (op OP) {
	if opn < 0 || opn > OP_SIZE {
		return
	}

	pool := OP_POOL[opn]
	if pool == nil {
		return
	}
	op, ok := pool.Get().(OP)
	if !ok {
		return
	}
	return
}

func PutOP(opn int, op OP) {
	if opn < 0 || opn > OP_SIZE {
		return
	}
	pool := OP_POOL[opn]
	if pool == nil {
		return
	}
	pool.Put(op)
}
