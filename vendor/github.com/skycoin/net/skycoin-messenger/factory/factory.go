package factory

import (
	"sync"

	"encoding/json"

	"time"

	"errors"

	"io/ioutil"

	"os"

	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/skycoin/net/factory"
	"github.com/skycoin/skycoin/src/cipher"
)

type MessengerFactory struct {
	factory             factory.Factory
	udp                 *factory.UDPFactory
	regConnections      map[cipher.PubKey]*Connection
	regConnectionsMutex sync.RWMutex

	// custom msg callback
	CustomMsgHandler func(*Connection, []byte)

	// will deliver the services data to server if true
	Proxy bool
	serviceDiscovery

	fieldsMutex sync.RWMutex
}

func NewMessengerFactory() *MessengerFactory {
	return &MessengerFactory{regConnections: make(map[cipher.PubKey]*Connection), serviceDiscovery: newServiceDiscovery()}
}

func (f *MessengerFactory) Listen(address string) (err error) {
	tcp := factory.NewTCPFactory()
	tcp.AcceptedCallback = f.acceptedCallback
	f.fieldsMutex.Lock()
	f.factory = tcp
	f.fieldsMutex.Unlock()
	err = tcp.Listen(address)
	if err != nil {
		return
	}
	if !f.Proxy {
		udp := factory.NewUDPFactory()
		udp.AcceptedCallback = f.acceptedUDPCallback
		f.fieldsMutex.Lock()
		f.udp = udp
		f.fieldsMutex.Unlock()
		err = udp.Listen(address)
	}
	return
}

func (f *MessengerFactory) acceptedUDPCallback(connection *factory.Connection) {
	var err error
	conn := newConnection(connection, f)
	conn.SetContextLogger(conn.GetContextLogger().WithField("app", "messenger"))
	defer func() {
		if e := recover(); e != nil {
			conn.GetContextLogger().Errorf("acceptedUDPCallback recover err %v", e)
		}
		if err != nil {
			conn.GetContextLogger().Errorf("acceptedUDPCallback err %v", err)
		}
		conn.Close()
	}()
	err = f.callbackLoop(conn)
	if err == ErrDetach {
		err = nil
		conn.WaitForDisconnected()
	}
}

func (f *MessengerFactory) callbackLoop(conn *Connection) (err error) {
	var m []byte
	var ok bool
	defer func() {
		if err != nil && err != ErrDetach {
			conn.GetContextLogger().Debugf("err in %x", m)
		}
	}()
	for {
		select {
		case m, ok = <-conn.GetChanIn():
			if !ok {
				return
			}
			if len(m) < MSG_HEADER_END {
				return
			}
			opn := m[MSG_OP_BEGIN]
			op := getOP(int(opn))
			if op == nil {
				continue
			}
			var rb []byte
			if sop, ok := op.(simpleOP); ok {
				body := m[MSG_HEADER_END:]
				if len(body) > 0 {
					err = json.Unmarshal(m[MSG_HEADER_END:], sop)
					if err != nil {
						return
					}
				}
				var r resp
				r, err = sop.Execute(f, conn)
				if err != nil {
					return
				}
				if r != nil {
					rb, err = json.Marshal(r)
				}
			} else if rop, ok := op.(rawOP); ok {
				rb, err = rop.RawExecute(f, conn, m)
			} else {
				err = errors.New("not implement op type")
				return
			}
			if err != nil {
				return
			}
			if rb != nil {
				err = conn.writeOPBytes(opn|RESP_PREFIX, rb)
				if err != nil {
					return
				}
			}
			putOP(int(opn), op)
		}
	}
}

func (f *MessengerFactory) acceptedCallback(connection *factory.Connection) {
	var err error
	conn := newConnection(connection, f)
	conn.SetContextLogger(conn.GetContextLogger().WithField("app", "messenger"))
	defer func() {
		if e := recover(); e != nil {
			conn.GetContextLogger().Errorf("acceptedCallback recover err %v", e)
		}
		if err != nil {
			conn.GetContextLogger().Errorf("acceptedCallback err %v", err)
		}
		f.discoveryUnregister(conn)
		conn.Close()
	}()
	err = f.callbackLoop(conn)
}

func (f *MessengerFactory) register(key cipher.PubKey, connection *Connection) {
	f.regConnectionsMutex.Lock()
	c, ok := f.regConnections[key]
	if ok {
		if c == connection {
			f.regConnectionsMutex.Unlock()
			log.Debugf("reg %s %p already", key.Hex(), connection)
			return
		}
		log.Debugf("reg close %s %p for %p", key.Hex(), c, connection)
		defer c.Close()
	}
	connection.UpdateConnectTime()
	f.regConnections[key] = connection
	f.regConnectionsMutex.Unlock()
	log.Debugf("reg %s %p", key.Hex(), connection)
}

// Get accepted connection by key
func (f *MessengerFactory) GetConnection(key cipher.PubKey) (c *Connection, ok bool) {
	f.regConnectionsMutex.RLock()
	c, ok = f.regConnections[key]
	f.regConnectionsMutex.RUnlock()
	return
}

// Execute fn for each accepted connection
func (f *MessengerFactory) ForEachAcceptedConnection(fn func(key cipher.PubKey, conn *Connection)) {
	f.regConnectionsMutex.RLock()
	for k, v := range f.regConnections {
		fn(k, v)
	}
	f.regConnectionsMutex.RUnlock()
}

func (f *MessengerFactory) unregister(key cipher.PubKey, connection *Connection) {
	f.regConnectionsMutex.Lock()
	c, ok := f.regConnections[key]
	if ok && c == connection {
		delete(f.regConnections, key)
		f.regConnectionsMutex.Unlock()
		log.Debugf("unreg %s %p", key.Hex(), c)
	} else if ok {
		f.regConnectionsMutex.Unlock()
		log.Debugf("unreg %s %p != new %p", key.Hex(), connection, c)
	} else {
		f.regConnectionsMutex.Unlock()
	}
}

func (f *MessengerFactory) Connect(address string) (conn *Connection, err error) {
	return f.ConnectWithConfig(address, nil)
}

func (f *MessengerFactory) ConnectWithConfig(address string, config *ConnConfig) (conn *Connection, err error) {
	defer func() {
		if err != nil && conn != nil {
			conn.Close()
		}
	}()
	f.fieldsMutex.Lock()
	if f.factory == nil {
		tcpFactory := factory.NewTCPFactory()
		f.factory = tcpFactory
	}
	f.fieldsMutex.Unlock()
	c, err := f.factory.Connect(address)
	if err != nil {
		if config != nil && config.Reconnect {
			go func() {
				time.Sleep(config.ReconnectWait)
				f.ConnectWithConfig(address, config)
			}()
		}
		return nil, err
	}
	conn = newClientConnection(c, f)
	conn.SetContextLogger(conn.GetContextLogger().WithField("app", "messenger"))
	if config != nil {
		if len(config.Context) > 0 {
			for k, v := range config.Context {
				conn.StoreContext(k, v)
			}
		}
		var sc *SeedConfig
		if config.SeedConfig != nil {
			sc = config.SeedConfig
		} else if len(config.SeedConfigPath) > 0 {
			sc, err = ReadSeedConfig(config.SeedConfigPath)
			if err != nil {
				if os.IsNotExist(err) {
					sc = NewSeedConfig()
					err = WriteSeedConfig(sc, config.SeedConfigPath)
					if err != nil {
						err = fmt.Errorf("failed to write seed config  %v", err)
						return
					}
				} else {
					err = fmt.Errorf("failed to read seed config %v", err)
					return
				}
			}
		}
		if sc != nil {
			func() {
				defer func() {
					if e := recover(); e != nil {
						err = errors.New("invalid seed config file")
					}
				}()
				var k cipher.PubKey
				k, err = cipher.PubKeyFromHex(sc.PublicKey)
				if err != nil {
					return
				}
				var sk cipher.SecKey
				sk, err = cipher.SecKeyFromHex(sc.SecKey)
				if err != nil {
					return
				}
				conn.SetSecKey(sk)
				err = conn.RegWithKey(k, config.Context)
			}()
		} else {
			err = conn.Reg()
		}
	} else {
		err = conn.Reg()
	}

	if config != nil {
		conn.findServiceNodesByKeysCallback = config.FindServiceNodesByKeysCallback
		conn.findServiceNodesByAttributesCallback = config.FindServiceNodesByAttributesCallback
		conn.appConnectionInitCallback = config.AppConnectionInitCallback
		if config.OnConnected != nil {
			config.OnConnected(conn)
		}
		if config.Reconnect {
			go func() {
				conn.WaitForDisconnected()
				time.Sleep(config.ReconnectWait)
				f.ConnectWithConfig(address, config)
			}()
		}
	}
	return
}

func (f *MessengerFactory) connectUDPWithConfig(address string, config *ConnConfig) (conn *Connection, err error) {
	f.fieldsMutex.Lock()
	if f.udp == nil {
		ff := factory.NewUDPFactory()
		if config != nil && config.Creator != nil {
			ff.AcceptedCallback = config.Creator.acceptedUDPCallback
		}
		err = ff.Listen(":0")
		if err != nil {
			f.fieldsMutex.Unlock()
			return
		}
		f.udp = ff
	}
	f.fieldsMutex.Unlock()
	c, err := f.udp.ConnectAfterListen(address)
	if err != nil {
		return nil, err
	}
	conn = newUDPClientConnection(c, f)
	conn.SetContextLogger(conn.GetContextLogger().WithField("app", "transport"))
	if config != nil {
		if config.OnConnected != nil {
			config.OnConnected(conn)
		}
	}
	return
}

func (f *MessengerFactory) Close() (err error) {
	f.fieldsMutex.RLock()
	defer f.fieldsMutex.RUnlock()
	if f.factory != nil {
		err = f.factory.Close()
	}
	if err != nil {
		return
	}
	if f.udp != nil {
		err = f.udp.Close()
	}
	return
}

// Execute fn for each connection that connected to server
func (f *MessengerFactory) ForEachConn(fn func(connection *Connection)) {
	f.factory.ForEachConn(func(conn *factory.Connection) {
		real := conn.RealObject
		if real == nil {
			return
		}
		c, ok := real.(*Connection)
		if !ok {
			return
		}
		fn(c)
	})
}

func (f *MessengerFactory) discoveryRegister(conn *Connection, ns *NodeServices) {
	f.serviceDiscovery.register(conn, ns)
	if f.Proxy {
		nodeServices := f.pack()
		f.ForEachConn(func(connection *Connection) {
			connection.UpdateServices(nodeServices)
		})
	}
}

func (f *MessengerFactory) discoveryUnregister(conn *Connection) {
	f.serviceDiscovery.unregister(conn)
	if f.Proxy {
		nodeServices := f.pack()
		f.ForEachConn(func(connection *Connection) {
			connection.UpdateServices(nodeServices)
		})
	}
}

func (f *MessengerFactory) DisableLogger() {
	log.SetOutput(ioutil.Discard)
}

// These are the different logging levels. You can set the logging level to log
// on your instance of logger, obtained with `logrus.New()`.
const (
	// PanicLevel level, highest level of severity. Logs and then calls panic with the
	// message passed to Debug, Info, ...
	PanicLevel Level = iota
	// FatalLevel level. Logs and then calls `os.Exit(1)`. It will exit even if the
	// logging level is set to Panic.
	FatalLevel
	// ErrorLevel level. Logs. Used for errors that should definitely be noted.
	// Commonly used for hooks to send errors to an error tracking service.
	ErrorLevel
	// WarnLevel level. Non-critical entries that deserve eyes.
	WarnLevel
	// InfoLevel level. General operational entries about what's going on inside the
	// application.
	InfoLevel
	// DebugLevel level. Usually only enabled when debugging. Very verbose logging.
	DebugLevel
)

type Level log.Level

func (f *MessengerFactory) SetLoggerLevel(level Level) {
	log.SetLevel(log.Level(level))
}
