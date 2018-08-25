package op

import (
	"sync"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skywire/pkg/net/skycoin-messenger/factory"
	"github.com/skycoin/skywire/pkg/net/skycoin-messenger/msg"
)

type Send struct {
	PublicKey string
	Msg       string
}

func init() {
	msg.OP_POOL[msg.OP_SEND] = &sync.Pool{
		New: func() interface{} {
			return new(Send)
		},
	}
}

func (s *Send) Execute(c msg.OPer) error {
	key, err := cipher.PubKeyFromHex(s.PublicKey)
	if err != nil {
		return err
	}
	c.GetFactory().ForEachConn(func(connection *factory.Connection) {
		connection.Send(key, []byte(s.Msg))
	})
	return nil
}
