package op

import (
	"errors"
	"sync"
	"time"

	"github.com/skycoin/net/skycoin-messenger/factory"
	"github.com/skycoin/net/skycoin-messenger/msg"
	"github.com/skycoin/net/skycoin-messenger/websocket/data"
)

type Login struct {
	Address   string
	PublicKey string
}

func init() {
	msg.OP_POOL[msg.OP_LOGIN] = &sync.Pool{
		New: func() interface{} {
			return new(Login)
		},
	}
}

func (r *Login) Execute(c msg.OPer) (err error) {
	keys, err := data.GetKeys()
	if err != nil {
		return
	}
	if len(keys) < 1 {
		return errors.New("no public key found")
	}
	sc, ok := keys[r.PublicKey]
	if !ok {
		return errors.New("public key not found")
	}
	f := factory.NewMessengerFactory()
	_, err = f.ConnectWithConfig(r.Address, &factory.ConnConfig{
		SeedConfig:    sc,
		Reconnect:     true,
		ReconnectWait: 2 * time.Second,
		OnConnected: func(connection *factory.Connection) {
			go c.PushLoop(connection)
		},
	})
	if err != nil {
		return
	}
	c.SetFactory(f)
	return
}
