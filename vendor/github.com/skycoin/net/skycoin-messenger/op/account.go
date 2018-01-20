package op

import (
	"sync"

	"github.com/skycoin/net/skycoin-messenger/msg"
	"github.com/skycoin/net/skycoin-messenger/websocket/data"
)

func init() {
	msg.OP_POOL[msg.OP_ACCOUNT] = &sync.Pool{
		New: func() interface{} {
			return new(Account)
		},
	}
	msg.OP_POOL[msg.OP_REG] = &sync.Pool{
		New: func() interface{} {
			return new(Reg)
		},
	}
}

type Account struct {
}

func (r *Account) Execute(c msg.OPer) (err error) {
	sc, err := data.GetData()
	if err != nil {
		return
	}
	keys := make([]string, 0, len(sc))
	for k := range sc {
		keys = append(keys, k)
	}
	c.Push(msg.OP_ACCOUNT, keys)
	return
}

type Reg struct {
}

func (r *Reg) Execute(c msg.OPer) (err error) {
	sc, err := data.AddKey()
	if err != nil {
		return
	}
	keys := make([]string, 0, len(sc))
	for k := range sc {
		keys = append(keys, k)
	}
	c.Push(msg.OP_ACCOUNT, keys)
	return
}
