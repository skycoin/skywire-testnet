package factory

import (
	"crypto/aes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/SkycoinProject/skycoin/src/cipher"
	"github.com/SkycoinProject/skywire/pkg/net/conn"
	"github.com/SkycoinProject/skywire/pkg/net/factory"
)

const keyWaitTimeout time.Duration = 60 * time.Second

type Connection struct {
	*factory.Connection
	factory *MessengerFactory

	closed     bool
	key        cipher.PubKey
	keySetCond *sync.Cond
	keySet     bool
	secKey     cipher.SecKey
	targetKey  cipher.PubKey

	context sync.Map

	services    *NodeServices
	servicesMap map[cipher.PubKey]*Service
	fieldsMutex sync.RWMutex

	in chan []byte

	proxyConnections map[uint32]*Connection

	appTransports      map[cipher.PubKey]*Transport
	appTransportsMutex sync.RWMutex

	CreatedByTransport *Transport
	transportPair      *transportPair

	connectTime int64

	skipFactoryReg bool

	appMessages        []PriorityMsg
	appMessagesReadCnt int
	appMessagesMutex   sync.RWMutex
	appFeedback        *AppFeedback
	appFeedbackMutex   sync.RWMutex
	// callbacks

	// call after received response for FindServiceNodesByKeys
	findServiceNodesByKeysCallback func(resp *QueryResp)

	// call after received response for FindServiceNodesByAttributes
	findServiceNodesByAttributesCallback func(resp *QueryByAttrsResp)

	// call after received response for BuildAppConnection
	appConnectionInitCallback func(resp *AppConnResp) *AppFeedback

	onConnected    func(connection *Connection)
	onDisconnected func(connection *Connection)
	reconnect      func()
}

// Used by factory to spawn connections for server side
func newConnection(c *factory.Connection, factory *MessengerFactory) *Connection {
	connection := &Connection{
		Connection:    c,
		factory:       factory,
		appTransports: make(map[cipher.PubKey]*Transport),
	}
	c.RealObject = connection
	connection.keySetCond = sync.NewCond(connection.fieldsMutex.RLocker())
	return connection
}

// Used by factory to spawn connections for client side
func newClientConnection(c *factory.Connection, factory *MessengerFactory) *Connection {
	connection := &Connection{
		Connection:       c,
		factory:          factory,
		in:               make(chan []byte),
		proxyConnections: make(map[uint32]*Connection),
		appTransports:    make(map[cipher.PubKey]*Transport),
	}
	c.RealObject = connection
	connection.keySetCond = sync.NewCond(connection.fieldsMutex.RLocker())
	go func() {
		connection.preprocessor()
	}()
	return connection
}

// Used by factory to spawn connections for udp client side
func newUDPClientConnection(c *factory.Connection, factory *MessengerFactory) *Connection {
	connection := &Connection{
		Connection: c,
		factory:    factory,
		in:         make(chan []byte),
	}
	c.RealObject = connection
	connection.keySetCond = sync.NewCond(connection.fieldsMutex.RLocker())
	go func() {
		connection.preprocessor()
	}()
	return connection
}

// Used by factory to spawn connections for udp server side
func newUDPServerConnection(c *factory.Connection, factory *MessengerFactory) *Connection {
	connection := &Connection{
		Connection: c,
		factory:    factory,
	}
	c.RealObject = connection
	connection.keySetCond = sync.NewCond(connection.fieldsMutex.RLocker())
	return connection
}

func (c *Connection) setProxyConnection(seq uint32, conn *Connection) {
	c.fieldsMutex.Lock()
	c.proxyConnections[seq] = conn
	c.fieldsMutex.Unlock()
}

func (c *Connection) removeProxyConnection(seq uint32) (conn *Connection, ok bool) {
	c.fieldsMutex.Lock()
	conn, ok = c.proxyConnections[seq]
	if ok {
		delete(c.proxyConnections, seq)
	}
	c.fieldsMutex.Unlock()
	return
}

func (c *Connection) SetKey(key cipher.PubKey) {
	c.fieldsMutex.Lock()
	c.key = key
	c.keySet = true
	c.fieldsMutex.Unlock()
	c.keySetCond.Broadcast()
	if c.onConnected != nil {
		c.onConnected(c)
	}
}

func (c *Connection) IsKeySet() (b bool) {
	c.fieldsMutex.RLock()
	b = c.keySet
	c.fieldsMutex.RUnlock()
	return
}

func (c *Connection) GetKey() cipher.PubKey {
	c.fieldsMutex.RLock()
	defer c.fieldsMutex.RUnlock()
	if !c.keySet {
		c.keySetCond.Wait()
	}
	return c.key
}

func (c *Connection) SetSecKey(key cipher.SecKey) {
	c.fieldsMutex.Lock()
	c.secKey = key
	c.fieldsMutex.Unlock()
}

func (c *Connection) GetSecKey() (key cipher.SecKey) {
	c.fieldsMutex.RLock()
	key = c.secKey
	c.fieldsMutex.RUnlock()
	return
}

func (c *Connection) SetTargetKey(key cipher.PubKey) {
	c.fieldsMutex.Lock()
	c.targetKey = key
	c.fieldsMutex.Unlock()
}

func (c *Connection) GetTargetKey() (key cipher.PubKey) {
	c.fieldsMutex.RLock()
	key = c.targetKey
	c.fieldsMutex.RUnlock()
	return
}

func (c *Connection) setServices(s *NodeServices) {
	if s == nil {
		c.fieldsMutex.Lock()
		c.services = nil
		c.servicesMap = nil
		c.fieldsMutex.Unlock()
		return
	}
	m := make(map[cipher.PubKey]*Service)
	for _, v := range s.Services {
		m[v.Key] = v
	}
	c.fieldsMutex.Lock()
	c.services = s
	c.servicesMap = m
	c.fieldsMutex.Unlock()
}

func (c *Connection) getService(key cipher.PubKey) (service *Service, ok bool) {
	c.fieldsMutex.Lock()
	defer c.fieldsMutex.Unlock()
	service, ok = c.servicesMap[key]
	return
}

func (c *Connection) GetServices() *NodeServices {
	c.fieldsMutex.RLock()
	defer c.fieldsMutex.RUnlock()
	return c.services
}

func (c *Connection) Reg() error {
	return c.Write(GenRegMsg())
}

func (c *Connection) RegWithKey(key cipher.PubKey, context map[string]string) error {
	c.StoreContext(publicKey, key)
	return c.writeOPSyn(OP_REG_KEY, &regWithKey{PublicKey: key, Context: context, Version: RegWithKeyAndEncryptionVersion})
}

func (c *Connection) RegWithKeys(key, target cipher.PubKey, context map[string]string) error {
	c.StoreContext(publicKey, key)
	c.SetTargetKey(target)
	return c.writeOPSyn(OP_REG_KEY, &regWithKey{PublicKey: key, Context: context, Version: RegWithKeyAndEncryptionVersion})
}

// register services to discovery
func (c *Connection) UpdateServices(ns *NodeServices) (err error) {
	if ns != nil {
		if !checkNodeServices(ns) {
			err = fmt.Errorf("invalid NodeServices %#v", ns)
			return
		}
		ns.Version = []string{c.factory.GetAppVersion(), VERSION, conn.VERSION}
	}
	c.setServices(ns)
	if ns == nil {
		ns = &NodeServices{}
	}
	err = c.writeOP(OP_OFFER_SERVICE, ns)
	if err != nil {
		return err
	}
	return
}

func checkAttrs(attrs []string) bool {
	if len(attrs) > 3 {
		return false
	}
	for _, v := range attrs {
		if len(v) > 50 {
			return false
		}
	}
	return true
}

func checkAddress(addr string) (valid bool) {
	host, p, err := net.SplitHostPort(addr)
	if err != nil {
		return
	}
	if len(host) != 0 && net.ParseIP(host) == nil {
		return
	}
	port, err := strconv.Atoi(p)
	if err != nil {
		return
	}
	if 0 > port || port > 65535 {
		return
	}
	valid = true
	return
}

func checkPubKeyHex(key string) (valid bool) {
	bytes, err := hex.DecodeString(key)
	if err != nil {
		return
	}
	if len(bytes) != 33 {
		return
	}
	valid = true
	return
}

func checkNodeServices(ns *NodeServices) (valid bool) {
	if ns == nil {
		return
	}
	if len(ns.ServiceAddress) > 0 {
		valid = checkAddress(ns.ServiceAddress)
		if !valid {
			return
		}
	}
	for _, s := range ns.Services {
		valid = checkAttrs(s.Attributes)
		if !valid {
			return
		}
		if len(s.Address) > 0 {
			valid = checkAddress(s.Address)
			if !valid {
				return
			}
		}
		if s.Key == EMPTY_PUBLIC_KEY {
			return false
		}
		for _, k := range s.AllowNodes {
			valid = checkPubKeyHex(k)
			if !valid {
				return
			}
		}
	}
	valid = true
	return
}

// register a service to discovery
func (c *Connection) OfferService(attrs ...string) error {
	return c.UpdateServices(&NodeServices{Services: []*Service{{Key: c.GetKey(), Attributes: attrs}}})
}

// register a service to discovery
func (c *Connection) OfferServiceWithAddress(address, version string, attrs ...string) error {
	return c.UpdateServices(&NodeServices{
		Services: []*Service{{Key: c.GetKey(),
			Attributes: attrs,
			Address:    address,
			Version:    version,
		}}})
}

// register a service to discovery
func (c *Connection) OfferPrivateServiceWithAddress(address, version string, allowNodes []string, attrs ...string) error {
	return c.UpdateServices(&NodeServices{
		Services: []*Service{{
			Key:               c.GetKey(),
			Attributes:        attrs,
			Address:           address,
			HideFromDiscovery: true,
			AllowNodes:        allowNodes,
			Version:           version,
		}}})
}

// find services by attributes
func (c *Connection) FindServiceNodesByAttributes(attrs ...string) error {
	return c.writeOP(OP_QUERY_BY_ATTRS, newQueryByAttrs(attrs))
}

// find services by attributes
func (c *Connection) FindServiceNodesWithSeqByAttributes(attrs ...string) (seq uint32, err error) {
	q := newQueryByAttrs(attrs)
	seq = q.Seq
	err = c.writeOP(OP_QUERY_BY_ATTRS, q)
	return
}

// find services by attributes
func (c *Connection) FindServiceNodesWithSeqByAttributesAndPaging(pages, limit int, attrs ...string) (seq uint32, err error) {
	q := newQueryByAttrsAndPage(pages, limit, attrs)
	seq = q.Seq
	err = c.writeOP(OP_QUERY_BY_ATTRS, q)
	return
}

// find services nodes by service public keys
func (c *Connection) FindServiceNodesByKeys(keys []cipher.PubKey) error {
	return c.writeOP(OP_QUERY_SERVICE_NODES, newQuery(keys))
}

func (c *Connection) BuildAppConnection(node, app, discovery cipher.PubKey) error {
	return c.writeOP(OP_BUILD_APP_CONN, &appConn{Node: node, App: app, Discovery: discovery})
}

func (c *Connection) Send(to cipher.PubKey, msg []byte) error {
	return c.Write(GenSendMsg(c.GetKey(), to, msg))
}

func (c *Connection) SendCustom(msg []byte) error {
	return c.writeOPBytes(OP_CUSTOM, msg)
}

func (c *Connection) preprocessor() (err error) {
	defer func() {
		if !conn.DEV {
			if e := recover(); e != nil {
				c.GetContextLogger().Debugf("panic in preprocessor %v", e)
			}
		}
		if err != nil {
			c.GetContextLogger().Debugf("preprocessor err %v", err)
		}
		c.Close()
	}()
OUTER:
	for {
		select {
		case m, ok := <-c.Connection.GetChanIn():
			if !ok {
				return
			}
			if conn.DEBUG_DATA_HEX {
				c.GetContextLogger().Debugf("preprocessor read %x", m)
			}
			if len(m) < MSG_HEADER_END {
				return
			}
			opn := m[MSG_OP_BEGIN]
			if opn&RESP_PREFIX > 0 {
				i := int(opn &^ RESP_PREFIX)
				r := getResp(i)
				if r != nil {
					body := m[MSG_HEADER_END:]
					if len(body) > 0 {
						err = json.Unmarshal(body, r)
						if err != nil {
							return
						}
					}
					c.GetContextLogger().Debug("preprocessor executing op")
					err = r.Run(c)
					if err != nil {
						c.GetContextLogger().WithError(err).Debug("preprocessor executed op")
						if err == ErrDetach {
							err = nil
							break OUTER
						}
						return
					}
					c.GetContextLogger().Debug("preprocessor executed op")
					putResp(i, r)
					continue
				}
			}

			c.in <- m
		}
	}
	for {
		select {
		case m, ok := <-c.Connection.GetChanIn():
			if !ok {
				return
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
	c.keySetCond.Broadcast()
	c.fieldsMutex.Lock()
	defer c.fieldsMutex.Unlock()
	if c.closed {
		return
	}
	c.closed = true
	if c.reconnect != nil {
		go c.reconnect()
	}
	if c.onDisconnected != nil {
		c.onDisconnected(c)
	}
	if c.keySet {
		if !c.skipFactoryReg {
			c.factory.unregister(c.key, c)
		}
		c.keySet = false
	}
	if c.in != nil {
		close(c.in)
	}

	if c.transportPair != nil {
		c.transportPair.close()
	}

	c.appTransportsMutex.RLock()
	if len(c.appTransports) > 0 {
		for _, v := range c.appTransports {
			v.Close()
		}
	}
	c.appTransportsMutex.RUnlock()

	c.Connection.Close()
}

func (c *Connection) WaitForKey() (err error) {
	c.GetContextLogger().WithField("timeout", keyWaitTimeout).Debug("WaitForKey")
	ok := make(chan struct{})
	go func() {
		c.GetKey()
		close(ok)
	}()

	t1 := time.Now()

	select {
	case <-time.After(keyWaitTimeout):
		err = errors.New("reg timeout")
		c.SetStatusToError(err)
		c.Close()
	case <-ok:
		t2 := time.Now()
		c.GetContextLogger().WithField("elapsed", t2.Sub(t1)).Debug("WaitForKey completed")
	}
	return err
}

func (c *Connection) writeOPBytes(op byte, body []byte) error {
	data := make([]byte, MSG_HEADER_END+len(body))
	data[MSG_OP_BEGIN] = op
	copy(data[MSG_HEADER_END:], body)
	return c.Write(data)
}

func (c *Connection) writeOP(op byte, object interface{}) error {
	js, err := json.Marshal(object)
	if err != nil {
		return err
	}

	if c.factory.LogWriteOps {
		c.GetContextLogger().Debugf("writeOP %#v", object)
	}

	return c.writeOPBytes(op, js)
}

func (c *Connection) writeOPSyn(op byte, object interface{}) error {
	body, err := json.Marshal(object)
	if err != nil {
		return err
	}

	if c.factory.LogWriteOps {
		c.GetContextLogger().Debugf("writeOP %#v", object)
	}

	data := make([]byte, MSG_HEADER_END+len(body))
	data[MSG_OP_BEGIN] = op
	copy(data[MSG_HEADER_END:], body)
	return c.WriteSyn(data)
}

// Set transport if key is not exists. Delete the transport of the key if tr is nil
func (c *Connection) setTransportIfNotExists(key cipher.PubKey, tr *Transport) (exists bool) {
	c.appTransportsMutex.Lock()
	if tr == nil {
		delete(c.appTransports, key)
	} else {
		_, exists = c.appTransports[key]
		if !exists {
			c.appTransports[key] = tr
		}
	}
	c.appTransportsMutex.Unlock()
	return
}

func (c *Connection) deleteTransport(key cipher.PubKey) {
	c.appTransportsMutex.Lock()
	delete(c.appTransports, key)
	c.appTransportsMutex.Unlock()
	return
}

func (c *Connection) setTransport(key cipher.PubKey, tr *Transport) {
	c.appTransportsMutex.Lock()
	c.appTransports[key] = tr
	c.appTransportsMutex.Unlock()
	return
}

func (c *Connection) getTransport(key cipher.PubKey) (tr *Transport, ok bool) {
	c.appTransportsMutex.RLock()
	tr, ok = c.appTransports[key]
	c.appTransportsMutex.RUnlock()
	return
}

func (c *Connection) UpdateConnectTime() {
	atomic.StoreInt64(&c.connectTime, time.Now().Unix())
}

func (c *Connection) GetConnectTime() int64 {
	return atomic.LoadInt64(&c.connectTime)
}

func (c *Connection) EnableSkipFactoryReg() {
	c.fieldsMutex.Lock()
	c.skipFactoryReg = true
	c.fieldsMutex.Unlock()
}

func (c *Connection) IsSkipFactoryReg() (skip bool) {
	c.fieldsMutex.RLock()
	skip = c.skipFactoryReg
	c.fieldsMutex.RUnlock()
	return
}

func (c *Connection) ForEachTransport(fn func(t *Transport)) {
	filter := make(map[*Transport]struct{})
	c.appTransportsMutex.RLock()
	defer c.appTransportsMutex.RUnlock()
	for _, tr := range c.appTransports {
		_, ok := filter[tr]
		if ok {
			continue
		}
		filter[tr] = struct{}{}
		fn(tr)
	}
}

func (c *Connection) StoreContext(key, value interface{}) {
	c.context.Store(key, value)
}

func (c *Connection) LoadContext(key interface{}) (value interface{}, ok bool) {
	return c.context.Load(key)
}

func (c *Connection) PutMessage(v PriorityMsg) {
	c.appMessagesMutex.Lock()
	v.Time = time.Now().Unix()
	c.appMessages = append(c.appMessages, v)
	c.appMessagesMutex.Unlock()
}

// Get messages
func (c *Connection) GetMessages() (result []PriorityMsg) {
	c.appMessagesMutex.Lock()
	result = c.appMessages
	c.appMessagesReadCnt = len(result)
	c.appMessagesMutex.Unlock()
	return result
}

// Return unread messages count
func (c *Connection) CheckMessages() (result int) {
	c.appMessagesMutex.RLock()
	result = len(c.appMessages) - c.appMessagesReadCnt
	c.appMessagesMutex.RUnlock()
	return result
}

func (c *Connection) SetAppFeedback(fb *AppFeedback) {
	c.appFeedbackMutex.Lock()
	if c.appFeedback == nil || fb.Discovery == c.appFeedback.Discovery {
		c.appFeedback = fb
	}
	c.appFeedbackMutex.Unlock()
}

func (c *Connection) GetAppFeedback() (v *AppFeedback) {
	c.appFeedbackMutex.RLock()
	v = c.appFeedback
	c.appFeedbackMutex.RUnlock()
	return
}

func (c *Connection) SetCrypto(pk cipher.PubKey, sk cipher.SecKey, target cipher.PubKey, iv []byte) (err error) {
	c.fieldsMutex.Lock()
	defer c.fieldsMutex.Unlock()
	if c.Connection.GetCrypto() != nil {
		return
	}
	crypto := conn.NewCrypto(pk, sk)
	err = crypto.SetTargetKey(target)
	if err != nil {
		return
	}
	if len(iv) == aes.BlockSize {
		err = crypto.Init(iv)
		if err != nil {
			return
		}
	}
	c.Connection.SetCrypto(crypto)
	return
}

func (c *Connection) SetTransportPair(pair *transportPair) {
	c.fieldsMutex.Lock()
	c.transportPair = pair
	c.fieldsMutex.Unlock()
}

func (c *Connection) GetTransportPair() (pair *transportPair) {
	c.fieldsMutex.RLock()
	pair = c.transportPair
	c.fieldsMutex.RUnlock()
	return
}
