package factory

import (
	"sync"

	"github.com/SkycoinProject/skycoin/src/cipher"
)

func init() {
	ops[OP_SEND] = &sync.Pool{
		New: func() interface{} {
			return new(send)
		},
	}
}

type send struct {
}

func (send *send) RawExecute(f *MessengerFactory, conn *Connection, m []byte) (rb []byte, err error) {
	if len(m) < SEND_MSG_TO_PUBLIC_KEY_END {
		return
	}
	key := cipher.NewPubKey(m[SEND_MSG_TO_PUBLIC_KEY_BEGIN:SEND_MSG_TO_PUBLIC_KEY_END])
	f.regConnectionsMutex.RLock()
	c, ok := f.regConnections[key]
	f.regConnectionsMutex.RUnlock()
	if !ok {
		conn.GetContextLogger().Infof("Key %s not found", key.Hex())
		return
	}
	err = c.Write(m)
	if err != nil {
		conn.GetContextLogger().Errorf("forward to Key %s err %v", key.Hex(), err)
		c.GetContextLogger().Errorf("write %x err %v", m, err)
		c.Close()
	}
	return
}
