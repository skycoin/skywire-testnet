package factory

import (
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/skycoin/net/conn"
	"github.com/skycoin/net/factory"
	"github.com/skycoin/skycoin/src/cipher"
	"io/ioutil"
	"sync"
	"time"
)

type MessengerFactory struct {
	factory             factory.Factory
	udp                 *factory.UDPFactory
	udpMutex            sync.Mutex
	regConnections      map[cipher.PubKey]*Connection
	regConnectionsMutex sync.RWMutex

	// custom msg callback
	CustomMsgHandler func(*Connection, []byte)

	// will deliver the services data to server if true
	Proxy bool
	serviceDiscovery

	defaultSeedConfig *SeedConfig

	Parent *MessengerFactory
	// on accepted callback
	OnAcceptedUDPCallback func(connection *Connection)

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
	c, ok := connection.RealObject.(*Connection)
	if !ok {
		c = newUDPServerConnection(connection, f)
	}
	c.SetContextLogger(c.GetContextLogger().WithField("app", "messenger"))
	if !conn.DEV {
		defer func() {
			if e := recover(); e != nil {
				c.GetContextLogger().Errorf("acceptedUDPCallback recover err %v", e)
			}
			if err != nil {
				c.GetContextLogger().Errorf("acceptedUDPCallback err %v", err)
			}
			c.Close()
		}()
	}
	if f.OnAcceptedUDPCallback != nil {
		f.OnAcceptedUDPCallback(c)
	}
	err = f.callbackLoop(c)
	if err == ErrDetach {
		err = nil
		c.WaitForDisconnected()
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
				conn.GetContextLogger().Debugf("op not found %x", m)
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
	c := newConnection(connection, f)
	c.SetContextLogger(c.GetContextLogger().WithField("app", "messenger"))
	if !conn.DEV {
		defer func() {
			if e := recover(); e != nil {
				c.GetContextLogger().Errorf("acceptedCallback recover err %v", e)
			}
			if err != nil {
				c.GetContextLogger().Errorf("acceptedCallback err %v", err)
			}
			f.discoveryUnregister(c)
			c.Close()
		}()
	}
	err = f.callbackLoop(c)
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

func (f *MessengerFactory) Connect(address string) (err error) {
	return f.ConnectWithConfig(address, nil)
}

func (f *MessengerFactory) loadSeedConfig(config *ConnConfig) (key cipher.PubKey, secKey cipher.SecKey, err error) {
	var sc *SeedConfig
	if config.SeedConfig != nil {
		sc = config.SeedConfig
		err = sc.parse()
		if err != nil {
			return
		}
	} else if len(config.SeedConfigPath) > 0 {
		sc, err = ReadOrCreateSeedConfig(config.SeedConfigPath)
	} else {
		sc = f.GetDefaultSeedConfig()

	}
	if sc == nil {
		err = fmt.Errorf("failed to load seed config %#v", config)
		return
	}
	key = sc.publicKey
	secKey = sc.secKey
	return
}

func (f *MessengerFactory) SetDefaultSeedConfigPath(path string) error {
	sc, err := ReadOrCreateSeedConfig(path)
	if err != nil {
		return err
	}
	f.fieldsMutex.Lock()
	f.defaultSeedConfig = sc
	f.fieldsMutex.Unlock()
	return nil
}

func (f *MessengerFactory) SetDefaultSeedConfig(sc *SeedConfig) error {
	f.fieldsMutex.Lock()
	f.defaultSeedConfig = sc
	f.fieldsMutex.Unlock()
	return nil
}

func (f *MessengerFactory) GetDefaultSeedConfig() (sc *SeedConfig) {
	f.fieldsMutex.RLock()
	sc = f.defaultSeedConfig
	f.fieldsMutex.RUnlock()
	return
}

func (f *MessengerFactory) ConnectWithConfig(address string, config *ConnConfig) (err error) {
	var conn *Connection
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
		return err
	}
	conn = newClientConnection(c, f)
	conn.SetContextLogger(conn.GetContextLogger().WithField("app", "messenger"))
	if config != nil {
		conn.onConnected = config.OnConnected
		conn.onDisconnected = config.OnDisconnected
		conn.findServiceNodesByKeysCallback = config.FindServiceNodesByKeysCallback
		conn.findServiceNodesByAttributesCallback = config.FindServiceNodesByAttributesCallback
		conn.appConnectionInitCallback = config.AppConnectionInitCallback
		if config.Reconnect {
			conn.reconnect = func() {
				time.Sleep(config.ReconnectWait)
				f.ConnectWithConfig(address, config)
			}
		}
		if len(config.Context) > 0 {
			for k, v := range config.Context {
				conn.StoreContext(k, v)
			}
		}
		var key cipher.PubKey
		var secKey cipher.SecKey
		key, secKey, err = f.loadSeedConfig(config)
		if err == nil {
			conn.SetSecKey(secKey)
			if config.TargetKey != EMPATY_PUBLIC_KEY {
				err = conn.RegWithKeys(key, config.TargetKey, config.Context)
			} else {
				err = conn.RegWithKey(key, config.Context)
			}
		} else {
			conn.GetContextLogger().Error(err)
			err = conn.Reg()
		}
	} else {
		err = conn.Reg()
	}

	if err != nil {
		return
	}
	err = conn.WaitForKey()
	return
}

func (f *MessengerFactory) listenForUDP() (err error) {
	f.fieldsMutex.Lock()
	if f.udp == nil {
		ff := factory.NewUDPFactory()
		ff.AcceptedCallback = f.acceptedUDPCallback
		err = ff.Listen(":0")
		if err != nil {
			f.fieldsMutex.Unlock()
			return
		}
		f.udp = ff
	}
	f.fieldsMutex.Unlock()
	return
}

func (f *MessengerFactory) connectUDPWithConfig(address string, config *ConnConfig) (connection *Connection, err error) {
	f.fieldsMutex.Lock()
	if f.udp == nil {
		err = errors.New("udp is nil")
		return
	}
	f.fieldsMutex.Unlock()
	c, err := f.udp.ConnectAfterListen(address)
	if err != nil {
		return nil, err
	}
	if c == nil {
		panic("c == nil")
	}
	connection = newUDPClientConnection(c, f)
	connection.SetContextLogger(connection.GetContextLogger().WithField("app", "transport"))
	if config != nil {
		if config.Creator != nil {
			connection.factory = config.Creator
		}
		if config.UseCrypto == RegWithKeyAndEncryptionVersion {
			var key cipher.PubKey
			var secKey cipher.SecKey
			key, secKey, err = f.loadSeedConfig(config)
			if err == nil {
				connection.SetSecKey(secKey)
				if config.TargetKey != EMPATY_PUBLIC_KEY {
					err = connection.RegWithKeys(key, config.TargetKey, config.Context)
				} else {
					err = connection.RegWithKey(key, config.Context)
				}
				err = connection.WaitForKey()
			}
		}
	}
	return
}

func (f *MessengerFactory) acceptUDPWithConfig(address string, config *ConnConfig) (connection *Connection, err error) {
	f.fieldsMutex.Lock()
	if f.udp == nil {
		err = errors.New("udp is nil")
		return
	}
	f.fieldsMutex.Unlock()
	c, err := f.udp.ConnectAfterListen(address)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, nil
	}
	connection = newUDPServerConnection(c, f)
	go f.udp.AcceptedCallback(c)
	connection.SetContextLogger(connection.GetContextLogger().WithField("app", "transport"))
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
		if !c.IsKeySet() {
			c.GetKey()
		}
		fn(c)
	})
}

func (f *MessengerFactory) discoveryRegister(conn *Connection, ns *NodeServices) (err error) {
	if ns != nil && !checkNodeServices(ns) {
		err = fmt.Errorf("invalid NodeServices %#v", ns)
		return
	}
	f.serviceDiscovery.register(conn, ns)
	if f.Proxy {
		nodeServices := f.pack()
		f.ForEachConn(func(connection *Connection) {
			err := connection.UpdateServices(nodeServices)
			if err != nil {
				connection.GetContextLogger().Errorf("discoveryRegister err %v", err)
			}
		})
	}
	return
}

func (f *MessengerFactory) ResyncToDiscovery(connection *Connection) (err error) {
	if !f.Proxy {
		return
	}
	nodeServices := f.pack()
	if nodeServices == nil {
		return
	}
	err = connection.UpdateServices(nodeServices)
	if err != nil {
		connection.GetContextLogger().Errorf("ResyncToDiscovery err %v", err)
	}
	return
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
