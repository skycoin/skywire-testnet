package factory

import (
	"sync"

	"encoding/json"
	log "github.com/sirupsen/logrus"
	"github.com/skycoin/net/factory"
	"github.com/skycoin/skycoin/src/cipher"
)

func init() {
	log.SetLevel(log.DebugLevel)
}

type MessengerFactory struct {
	factory             factory.Factory
	regConnections      map[cipher.PubKey]*Connection
	regConnectionsMutex sync.RWMutex
	CustomMsgHandler    func(*Connection, []byte)

	serviceDiscovery
}

func NewMessengerFactory() *MessengerFactory {
	return &MessengerFactory{regConnections: make(map[cipher.PubKey]*Connection), serviceDiscovery: newServiceDiscovery()}
}

func (f *MessengerFactory) Listen(address string) error {
	tcpFactory := factory.NewTCPFactory()
	f.factory = tcpFactory
	tcpFactory.AcceptedCallback = f.acceptedCallback
	return tcpFactory.Listen(address)
}

var EMPTY_KEY = cipher.PubKey{}

func (f *MessengerFactory) acceptedCallback(connection *factory.Connection) {
	var err error
	conn := newConnection(connection)
	conn.SetContextLogger(conn.GetContextLogger().WithField("app", "messenger"))
	defer func() {
		if e := recover(); e != nil {
			conn.GetContextLogger().Errorf("acceptedCallback recover err %v", e)
		}
		if err != nil {
			conn.GetContextLogger().Errorf("acceptedCallback err %v", err)
		}
		f.unregister(conn.GetKey(), conn)
		f.serviceDiscovery.unregister(conn)
		conn.Close()
	}()
	for {
		select {
		case m, ok := <-conn.GetChanIn():
			if !ok {
				return
			}
			if len(m) < MSG_HEADER_END {
				return
			}
			op := m[MSG_OP_BEGIN]
			switch op {
			case OP_REG:
				if conn.IsKeySet() {
					conn.GetContextLogger().Infof("reg %s already", conn.key.Hex())
					continue
				}
				key, _ := cipher.GenerateKeyPair()
				conn.SetKey(key)
				conn.SetContextLogger(conn.GetContextLogger().WithField("pubkey", key.Hex()))
				f.register(key, conn)
				err = conn.Write(GenRegRespMsg(key))
				if err != nil {
					return
				}
			case OP_SEND:
				if len(m) < SEND_MSG_TO_PUBLIC_KEY_END {
					return
				}
				key := cipher.NewPubKey(m[SEND_MSG_TO_PUBLIC_KEY_BEGIN:SEND_MSG_TO_PUBLIC_KEY_END])
				f.regConnectionsMutex.RLock()
				c, ok := f.regConnections[key]
				f.regConnectionsMutex.RUnlock()
				if !ok {
					conn.GetContextLogger().Infof("Key %s not found", key.Hex())
					continue
				}
				err = c.Write(m)
				if err != nil {
					conn.GetContextLogger().Errorf("forward to Key %s err %v", key.Hex(), err)
					c.GetContextLogger().Errorf("write %x err %v", m, err)
					c.Close()
				}
			case OP_CUSTOM:
				if f.CustomMsgHandler != nil {
					f.CustomMsgHandler(conn, m[MSG_HEADER_END:])
				}
			case OP_OFFER_SERVICE:
				var services []*Service
				err = json.Unmarshal(m[MSG_HEADER_END:], services)
				if err != nil {
					return
				}
				f.serviceDiscovery.register(conn, services)
			case OP_GET_SERVICE_NODES:
				var service *Service
				err = json.Unmarshal(m[MSG_HEADER_END:], service)
				if err != nil {
					return
				}
				if len(service.Attributes) > 0 {
					err = conn.Write(GenGetServiceNodesRespMsg(f.serviceDiscovery.findByAttributes(service.Attributes)))
				} else {
					err = conn.Write(GenGetServiceNodesRespMsg(f.serviceDiscovery.find(service.Key)))
				}
				if err != nil {
					return
				}
			default:
				conn.GetContextLogger().Errorf("not implemented op %d", op)
			}
		}
	}
}

func (f *MessengerFactory) register(key cipher.PubKey, connection *Connection) {
	f.regConnectionsMutex.Lock()
	defer f.regConnectionsMutex.Unlock()
	c, ok := f.regConnections[key]
	if ok {
		if c == connection {
			log.Debugf("reg %s %p already", key.Hex(), connection)
			return
		}
		log.Debugf("reg close %s %p for %p", key.Hex(), c, connection)
		c.Close()
	}
	f.regConnections[key] = connection
	log.Debugf("reg %s %p", key.Hex(), connection)
}

func (f *MessengerFactory) GetConnection(key cipher.PubKey) (c *Connection, ok bool) {
	f.regConnectionsMutex.RLock()
	c, ok = f.regConnections[key]
	f.regConnectionsMutex.RUnlock()
	return
}

func (f *MessengerFactory) unregister(key cipher.PubKey, connection *Connection) {
	f.regConnectionsMutex.Lock()
	defer f.regConnectionsMutex.Unlock()
	c, ok := f.regConnections[key]
	if ok && c == connection {
		delete(f.regConnections, key)
		log.Debugf("unreg %s %p", key.Hex(), c)
	} else {
		log.Debugf("unreg %s %p != new %p", key.Hex(), connection, c)
	}
}

func (f *MessengerFactory) Connect(address string) (conn *Connection, err error) {
	tcpFactory := factory.NewTCPFactory()
	c, err := tcpFactory.Connect(address)
	if err != nil {
		return nil, err
	}
	conn = newClientConnection(c)
	conn.SetContextLogger(conn.GetContextLogger().WithField("app", "messenger"))
	err = conn.Reg()
	return
}

func (f *MessengerFactory) Close() error {
	return f.factory.Close()
}
