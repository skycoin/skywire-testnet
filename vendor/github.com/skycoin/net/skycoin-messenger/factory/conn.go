package factory

import (
	"sync"

	"encoding/json"
	"errors"

	"github.com/skycoin/net/factory"
	"github.com/skycoin/skycoin/src/cipher"
)

type Connection struct {
	*factory.Connection
	key         cipher.PubKey
	keySetCond  *sync.Cond
	keySet      bool
	services    []*Service
	fieldsMutex sync.RWMutex

	in chan []byte

	getServicesChan chan []cipher.PubKey
}

// Used by factory to spawn connections for server side
func newConnection(c *factory.Connection) *Connection {
	connection := &Connection{Connection: c}
	connection.keySetCond = sync.NewCond(connection.fieldsMutex.RLocker())
	return connection
}

// Used by factory to spawn connections for client side
func newClientConnection(c *factory.Connection) *Connection {
	connection := &Connection{Connection: c, in: make(chan []byte), getServicesChan: make(chan []cipher.PubKey)}
	connection.keySetCond = sync.NewCond(connection.fieldsMutex.RLocker())
	go connection.preprocessor()
	return connection
}

func (c *Connection) SetKey(key cipher.PubKey) {
	c.fieldsMutex.Lock()
	c.key = key
	c.keySet = true
	c.fieldsMutex.Unlock()
	c.keySetCond.Broadcast()
}

func (c *Connection) IsKeySet() bool {
	c.fieldsMutex.Lock()
	defer c.fieldsMutex.Unlock()
	return c.keySet
}

func (c *Connection) GetKey() cipher.PubKey {
	c.fieldsMutex.RLock()
	defer c.fieldsMutex.RUnlock()
	if !c.keySet {
		c.keySetCond.Wait()
	}
	return c.key
}

func (c *Connection) setServices(s []*Service) {
	c.fieldsMutex.Lock()
	defer c.fieldsMutex.Unlock()
	c.services = s
}

func (c *Connection) GetServices() []*Service {
	c.fieldsMutex.RLock()
	defer c.fieldsMutex.RUnlock()
	return c.services
}

func (c *Connection) Reg() error {
	return c.Write(GenRegMsg())
}

func (c *Connection) OfferService(attr ...string) error {
	return c.UpdateServices([]*Service{{Key: c.GetKey(), Attributes: attr}})
}

func (c *Connection) UpdateServices(services []*Service) error {
	if len(services) < 1 {
		return errors.New("len(services) < 1")
	}
	js, err := json.Marshal(services)
	if err != nil {
		return err
	}
	err = c.Write(GenOfferServiceMsg(js))
	if err != nil {
		return err
	}
	c.setServices(services)
	return nil
}

func (c *Connection) GetServiceNodes(attr ...string) (result []cipher.PubKey, err error) {
	return c.getServiceNodes(&Service{Attributes: attr})
}

func (c *Connection) GetServiceNodesByKey(key cipher.PubKey) (result []cipher.PubKey, err error) {
	return c.getServiceNodes(&Service{Key: key})
}

func (c *Connection) getServiceNodes(service *Service) (result []cipher.PubKey, err error) {
	js, err := json.Marshal(service)
	if err != nil {
		return
	}
	err = c.Write(GenGetServiceNodesMsg(js))
	if err != nil {
		return
	}
	result = <-c.getServicesChan
	return
}

func (c *Connection) Send(to cipher.PubKey, msg []byte) error {
	return c.Write(GenSendMsg(c.key, to, msg))
}

func (c *Connection) SendCustom(msg []byte) error {
	return c.Write(GenCustomMsg(msg))
}

func (c *Connection) preprocessor() error {
	defer func() {
		if e := recover(); e != nil {
			c.GetContextLogger().Debugf("panic in preprocessor %v", e)
		}
	}()
	for {
		select {
		case m, ok := <-c.Connection.GetChanIn():
			if !ok {
				return nil
			}
			c.GetContextLogger().Debugf("read %x", m)
			if len(m) >= MSG_HEADER_END {
				switch m[MSG_OP_BEGIN] {
				case OP_REG:
					reg := m[MSG_HEADER_END:]
					if len(reg) < MSG_PUBLIC_KEY_SIZE {
						continue
					}
					key := cipher.NewPubKey(reg[:MSG_PUBLIC_KEY_SIZE])
					c.SetKey(key)
					c.SetContextLogger(c.GetContextLogger().WithField("pubkey", key.Hex()))
				case OP_GET_SERVICE_NODES:
					ks := m[MSG_HEADER_END:]
					kc := len(ks) / MSG_PUBLIC_KEY_SIZE
					if len(ks)%MSG_PUBLIC_KEY_SIZE != 0 || kc < 1 {
						continue
					}
					keys := make([]cipher.PubKey, kc)
					for i := 0; i < kc; i++ {
						key := cipher.NewPubKey(ks[i*MSG_PUBLIC_KEY_SIZE : (i+1)*MSG_PUBLIC_KEY_SIZE])
						keys[i] = key
					}
					c.getServicesChan <- keys
					continue
				}
			}
			c.in <- m
		}
	}
}

func (c *Connection) GetChanIn() <-chan []byte {
	if c.in == nil {
		return c.Connection.GetChanIn()
	}
	return c.in
}

func (c *Connection) Close() {
	c.fieldsMutex.Lock()
	defer c.fieldsMutex.Unlock()
	if c.IsClosed() {
		return
	}
	if c.in != nil {
		close(c.in)
	}
	if c.getServicesChan != nil {
		close(c.getServicesChan)
	}
	c.Connection.Close()
}
