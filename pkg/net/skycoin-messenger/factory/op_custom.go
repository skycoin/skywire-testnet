package factory

import (
	"sync"
)

func init() {
	ops[OP_CUSTOM] = &sync.Pool{
		New: func() interface{} {
			return new(Custom)
		},
	}
}

type Custom struct {
}

func (custom *Custom) RawExecute(f *MessengerFactory, conn *Connection, m []byte) (rb []byte, err error) {
	if f.CustomMsgHandler != nil {
		f.CustomMsgHandler(conn, m[MSG_HEADER_END:])
	}
	return
}
