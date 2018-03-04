package factory

import (
	"github.com/pkg/errors"
	"github.com/skycoin/net/util/producer"
	"sync"
)

func init() {
	ops[OP_POW] = &sync.Pool{
		New: func() interface{} {
			return new(workTicket)
		},
	}
}

type workTicket struct {
	Seq   uint32
	Code  []byte
	Codes [][]byte
	Last  bool
}

func (wt *workTicket) Execute(f *MessengerFactory, conn *Connection) (r resp, err error) {
	if f.Proxy {
		return
	}
	pair := conn.GetTransportPair()
	if pair == nil {
		err = errors.New("GetTransportPair == nil")
		return
	}

	ok, err := pair.submitTicket(wt)
	conn.GetContextLogger().Debugf("pow ticket %#v valid %t", wt, err == nil)
	if ok == 0 || err != nil {
		return
	}

	producer.Send(&producer.MqBody{
		Uid:          pair.uid,
		FromApp:      pair.fromApp.Hex(),
		FromNode:     pair.fromNode.Hex(),
		ToNode:       pair.fromNode.Hex(),
		ToApp:        pair.fromApp.Hex(),
		FromHostPort: pair.fromHostPort,
		ToHostPort:   pair.toHostPort,
		FromIp:       pair.fromIp,
		ToIp:         pair.toIp,
		Count:        uint64(ok),
		IsEnd:        wt.Last,
	})
	return
}
