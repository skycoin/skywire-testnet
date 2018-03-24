package factory

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	cn "github.com/skycoin/net/conn"
	"github.com/skycoin/net/msg"
	"github.com/skycoin/skycoin/src/cipher"
	"io"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type Transport struct {
	creator *MessengerFactory
	// node
	factory *MessengerFactory
	// conn between nodes
	conn *Connection
	// app
	appNet net.Listener
	// is this client side transport
	clientSide bool

	FromNode, ToNode cipher.PubKey
	FromApp, ToApp   cipher.PubKey
	servingPort      int

	conns      map[uint32]net.Conn
	connsMutex sync.RWMutex

	timeoutTimer  *time.Timer
	appConnHolder *Connection

	uploadBW   bandwidth
	downloadBW bandwidth

	connAcked bool

	discoveryConn *Connection

	ticketSeqCounter  uint32
	unChargeMsgs      []*msg.UDPMessage
	unChargeMsgsMutex sync.Mutex

	fieldsMutex sync.RWMutex
}

type transportPair struct {
	uid                                    uint64
	fromApp, fromNode, toNode, toApp       cipher.PubKey
	fromConn, toConn                       *Connection
	fromHostPort, toHostPort, fromIp, toIp string
	tickets                                map[uint32]*workTicket
	lastTicket                             *workTicket
	ticketsMutex                           sync.Mutex
	timeoutTimer                           *time.Timer
	closed                                 bool
	lastCheckedTime                        time.Time
	fieldsMutex                            sync.RWMutex
}

func (p *transportPair) ok() {
	p.fieldsMutex.Lock()
	if p.timeoutTimer == nil {
		p.fieldsMutex.Unlock()
		return
	}
	p.timeoutTimer.Stop()
	p.timeoutTimer = nil
	p.fieldsMutex.Unlock()
}

func (p *transportPair) close() {
	p.fieldsMutex.Lock()
	if p.closed {
		p.fieldsMutex.Unlock()
		return
	}
	p.closed = true
	p.fieldsMutex.Unlock()
	keys := p.fromApp.Hex() + p.fromNode.Hex() + p.toNode.Hex() + p.toApp.Hex()
	globalTransportPairManagerInstance.del(keys)
}

func (p *transportPair) submitTicket(ticket *workTicket) (ok uint, err error) {
	p.ticketsMutex.Lock()
	defer p.ticketsMutex.Unlock()

	if p.lastCheckedTime.IsZero() {
		p.lastCheckedTime = time.Now()
	} else if time.Now().Sub(p.lastCheckedTime) > 30*time.Second {
		if len(p.tickets) > 10 {
			err = errors.New("too many uncheck tickets")
			return
		}
	}

	if len(ticket.Codes) > 0 {
		if p.lastTicket == nil {
			clone := *ticket
			p.lastTicket = &clone
			return
		}
		t := p.lastTicket
		p.lastTicket = nil
		for i, c := range ticket.Codes {
			if hmac.Equal(t.Codes[i], c) {
				ok++
			} else {
				return
			}
		}
		return
	}

	t, o := p.tickets[ticket.Seq]
	if !o {
		clone := *ticket
		p.tickets[ticket.Seq] = &clone
		return
	}
	delete(p.tickets, ticket.Seq)
	if !hmac.Equal(t.Code, ticket.Code) {
		err = errors.New("ticket code is not valid")
		return
	}
	ok = msgsEveryTicket
	p.lastCheckedTime = time.Now()
	return
}

func (p *transportPair) setFromConn(fromConn *Connection) (err error) {
	p.fieldsMutex.Lock()
	addr := fromConn.GetRemoteAddr().String()
	fromIp, _, err := net.SplitHostPort(addr)
	if err != nil {
		p.fieldsMutex.Unlock()
		return
	}
	p.fromConn = fromConn
	hash := sha256.New()
	hash.Write([]byte(addr))
	p.fromHostPort = hex.EncodeToString(hash.Sum(nil))
	hash.Reset()
	hash.Write([]byte(fromIp))
	p.fromIp = hex.EncodeToString(hash.Sum(nil))

	p.fieldsMutex.Unlock()
	return
}

func (p *transportPair) setToConn(toConn *Connection) (err error) {
	p.fieldsMutex.Lock()
	addr := toConn.GetRemoteAddr().String()
	toIp, _, err := net.SplitHostPort(addr)
	if err != nil {
		p.fieldsMutex.Unlock()
		return
	}
	p.toConn = toConn
	hash := sha256.New()
	hash.Write([]byte(addr))
	p.toHostPort = hex.EncodeToString(hash.Sum(nil))
	hash.Reset()
	hash.Write([]byte(toIp))
	p.toIp = hex.EncodeToString(hash.Sum(nil))

	p.fieldsMutex.Unlock()
	return
}

var globalTransportPairManagerInstance = newTransportPairManager()

type transportPairManager struct {
	pairs      map[string]*transportPair
	pairsMutex sync.RWMutex
}

func newTransportPairManager() *transportPairManager {
	return &transportPairManager{
		pairs: make(map[string]*transportPair),
	}
}

var guid uint64 = 0

func (m *transportPairManager) create(fromApp, fromNode, toNode, toApp cipher.PubKey) (p *transportPair) {
	keys := fromApp.Hex() + fromNode.Hex() + toNode.Hex() + toApp.Hex()
	m.pairsMutex.Lock()
	p, ok := m.pairs[keys]
	if ok {
		delete(m.pairs, keys)
	}
	p = &transportPair{
		uid:      atomic.AddUint64(&guid, 1),
		fromApp:  fromApp,
		fromNode: fromNode,
		toNode:   toNode,
		toApp:    toApp,
		tickets:  make(map[uint32]*workTicket),
	}
	p.timeoutTimer = time.AfterFunc(120*time.Second, func() {
		p.close()
	})
	m.pairs[keys] = p
	m.pairsMutex.Unlock()
	return
}

func (m *transportPairManager) get(fromApp, fromNode, toNode, toApp cipher.PubKey) (p *transportPair, ok bool) {
	keys := fromApp.Hex() + fromNode.Hex() + toNode.Hex() + toApp.Hex()
	m.pairsMutex.RLock()
	p, ok = m.pairs[keys]
	m.pairsMutex.RUnlock()
	return
}

func (m *transportPairManager) del(keys string) {
	m.pairsMutex.Lock()
	delete(m.pairs, keys)
	m.pairsMutex.Unlock()
}

const msgsEveryTicket = 1000

func NewTransport(creator *MessengerFactory, appConn *Connection, fromNode, toNode, fromApp, toApp cipher.PubKey) *Transport {
	if appConn == nil {
		panic("appConn can not be nil")
	}
	cs := false
	if appConn.GetKey() == fromApp {
		cs = true
	} else if appConn.GetKey() != toApp {
		panic("invalid appConn value")
	}
	t := &Transport{
		creator:       creator,
		appConnHolder: appConn,
		FromNode:      fromNode,
		ToNode:        toNode,
		FromApp:       fromApp,
		ToApp:         toApp,
		clientSide:    cs,
		factory:       NewMessengerFactory(),
		conns:         make(map[uint32]net.Conn),
		unChargeMsgs:  make([]*msg.UDPMessage, 0, msgsEveryTicket-1),
	}
	ticketFunc := func(m *msg.UDPMessage) {
		c := atomic.AddUint32(&t.ticketSeqCounter, 1)
		if c%msgsEveryTicket != 0 {
			t.unChargeMsgsMutex.Lock()
			t.unChargeMsgs = append(t.unChargeMsgs, m)
			t.unChargeMsgsMutex.Unlock()
			return
		}
		t.unChargeMsgsMutex.Lock()
		t.unChargeMsgs = t.unChargeMsgs[:0]
		t.unChargeMsgsMutex.Unlock()
		t.sendTicket(c/msgsEveryTicket, m)
	}
	if cs {
		t.factory.BeforeReadOnConn = ticketFunc
	} else {
		t.factory.BeforeSendOnConn = ticketFunc
	}
	t.factory.Parent = creator
	t.factory.SetDefaultSeedConfig(creator.GetDefaultSeedConfig())
	return t
}

func (t *Transport) sendTicket(seq uint32, m *msg.UDPMessage) {
	mac := hmac.New(sha256.New, t.FromNode[:])
	mac.Write(m.Body)
	code := mac.Sum(nil)
	t.discoveryConn.writeOP(OP_POW, &workTicket{
		Seq:  seq,
		Code: code,
	})
}

func (t *Transport) sendLastTicket() {
	c := atomic.AddUint32(&t.ticketSeqCounter, 1)
	if c < msgsEveryTicket {
		return
	}
	t.unChargeMsgsMutex.Lock()
	codes := make([][]byte, len(t.unChargeMsgs))
	for i, m := range t.unChargeMsgs {
		mac := hmac.New(sha256.New, t.FromNode[:])
		mac.Write(m.Body)
		code := mac.Sum(nil)
		codes[i] = code
	}
	t.unChargeMsgsMutex.Unlock()
	t.discoveryConn.writeOP(OP_POW, &workTicket{
		Codes: codes,
		Last:  true,
	})
}

func (t *Transport) SetOnAcceptedUDPCallback(fn func(connection *Connection)) {
	t.factory.OnAcceptedUDPCallback = fn
}

func (t *Transport) String() string {
	return fmt.Sprintf("transport From App%s Node%s To Node%s App%s",
		t.FromApp.Hex(), t.FromNode.Hex(), t.ToNode.Hex(), t.ToApp.Hex())
}

// Listen and connect to node manager
func (t *Transport) ListenAndConnect(address string, key cipher.PubKey) (conn *Connection, err error) {
	err = t.factory.listenForUDP()
	if err != nil {
		return
	}
	conn, err = t.factory.connectUDPWithConfig(address, &ConnConfig{
		UseCrypto:           RegWithKeyAndEncryptionVersion,
		TargetKey:           key,
		SkipBeforeCallbacks: true,
	})
	conn.CreatedByTransport = t
	t.discoveryConn = conn
	return
}

// Connect to node B
func (t *Transport) clientSideConnect(address string, sc *SeedConfig, iv []byte) (err error) {
	t.fieldsMutex.Lock()
	defer t.fieldsMutex.Unlock()
	if t.connAcked {
		return
	}
	t.connAcked = true
	if t.factory == nil {
		err = errors.New("transport has been closed")
		return
	}

	conn, err := t.factory.acceptUDPWithConfig(address, &ConnConfig{})
	if err != nil {
		return
	}
	if conn == nil {
		err = errors.New("clientSideConnect acceptUDPWithConfig return nil conn")
		return
	}
	err = conn.SetCrypto(sc.publicKey, sc.secKey, t.ToNode, iv)
	if err != nil {
		return
	}
	err = conn.writeOP(OP_BUILD_APP_CONN_OK|RESP_PREFIX, &nop{})
	return
}

func (t *Transport) connAck() {
	t.fieldsMutex.Lock()
	t.connAcked = true
	t.fieldsMutex.Unlock()
}

func (t *Transport) isConnAck() (is bool) {
	t.fieldsMutex.RLock()
	is = t.connAcked
	t.fieldsMutex.RUnlock()
	return
}

// Connect to node A and server app
func (t *Transport) serverSiceConnect(address, appAddress string, sc *SeedConfig, iv []byte) (err error) {
	conn, err := t.factory.connectUDPWithConfig(address, &ConnConfig{})
	if err != nil {
		return
	}
	conn.CreatedByTransport = t
	conn.SetKey(t.FromNode)
	err = conn.SetCrypto(sc.publicKey, sc.secKey, t.FromNode, iv)
	if err != nil {
		return
	}
	err = conn.writeOP(OP_BUILD_APP_CONN_OK,
		&buildConnResp{
			FromNode: t.FromNode,
			Node:     t.ToNode,
			FromApp:  t.FromApp,
			App:      t.ToApp,
		})
	if err != nil {
		return
	}
	t.fieldsMutex.Lock()
	t.conn = conn
	t.fieldsMutex.Unlock()

	go t.nodeReadLoop(conn, func(id uint32) net.Conn {
		t.connsMutex.Lock()
		defer t.connsMutex.Unlock()
		appConn, ok := t.conns[id]
		if !ok {
			appConn, err = net.Dial("tcp", appAddress)
			if err != nil {
				log.Debugf("app conn dial err %v", err)
				return nil
			}
			t.conns[id] = appConn
			go t.appReadLoop(id, appConn, conn, false)
		}
		return appConn
	})

	return
}

func (t *Transport) getDiscoveryDisconntedChan() <-chan struct{} {
	if t.discoveryConn == nil {
		return nil
	}
	return t.discoveryConn.GetDisconnectedChan()
}

// Read from node, write to app
func (t *Transport) nodeReadLoop(conn *Connection, getAppConn func(id uint32) net.Conn) {
	defer func() {
		t.Close()
	}()
	var err error
	for {
		select {
		case m, ok := <-conn.GetChanIn():
			if !ok {
				conn.GetContextLogger().Debugf("node conn read err %v", err)
				return
			}
			if cn.DEBUG_DATA_HEX {
				conn.GetContextLogger().Debugf("get chan in %x", m)
			}
			t.downloadBW.add(len(m))
			id := binary.BigEndian.Uint32(m[PKG_HEADER_ID_BEGIN:PKG_HEADER_ID_END])
			appConn := getAppConn(id)
			if appConn == nil {
				continue
			}
			op := m[PKG_HEADER_OP_BEGIN]
			if op == OP_CLOSE {
				t.connsMutex.Lock()
				t.conns[id] = nil
				t.connsMutex.Unlock()
				appConn.Close()
				continue
			}
			if len(m) <= PKG_HEADER_END {
				continue
			}
			body := m[PKG_HEADER_END:]
			err = writeAll(appConn, body)
			if err != nil {
				conn.GetContextLogger().Debugf("app conn write err %v", err)
				t.connsMutex.Lock()
				t.conns[id] = nil
				t.connsMutex.Unlock()
				appConn.Close()
				continue
			}
		case <-t.getDiscoveryDisconntedChan():
			conn.GetContextLogger().Debugf("transport discovery conn closed")
			return
		}
	}
}

// Read from app, write to node
func (t *Transport) appReadLoop(id uint32, appConn net.Conn, conn *Connection, create bool) {
	buf := make([]byte, cn.MAX_UDP_PACKAGE_SIZE-100)
	binary.BigEndian.PutUint32(buf[PKG_HEADER_ID_BEGIN:PKG_HEADER_ID_END], id)
	channel := conn.NewPendingChannel()
	defer conn.DeletePendingChannel(channel)
	defer func() {
		if e := recover(); e != nil {
			conn.GetContextLogger().Debugf("close app conn %d, err %v", id, e)
		}
		t.connsMutex.Lock()
		defer t.connsMutex.Unlock()
		// exited by err
		if t.conns[id] != nil {
			buf[PKG_HEADER_OP_BEGIN] = OP_CLOSE
			//log.Infof("close %v, %d", create, id)
			if !conn.IsClosed() {
				func() {
					defer func() {
						if e := recover(); e != nil {
							conn.GetContextLogger().Debugf("close app conn %d, err %v", id, e)
						}
					}()
					conn.WriteToChannel(channel, buf[:PKG_HEADER_END])
				}()
			}
			if create {
				delete(t.conns, id)
			} else {
				t.conns[id] = nil
			}
			return
		}
		if create {
			delete(t.conns, id)
		}
	}()
	if create {
		conn.WriteToChannel(channel, buf[:PKG_HEADER_END])
	}
	for {
		n, err := appConn.Read(buf[PKG_HEADER_END:])
		if err != nil {
			log.Debugf("app conn read err %v, %d", err, n)
			return
		}
		pkg := buf[:PKG_HEADER_END+n]
		if cn.DEBUG_DATA_HEX {
			conn.GetContextLogger().Debugf("app conn in %x", pkg)
		}
		t.uploadBW.add(len(pkg))
		conn.WriteToChannel(channel, pkg)
	}
}

func (t *Transport) setUDPConn(conn *Connection) {
	t.fieldsMutex.Lock()
	t.conn = conn
	t.fieldsMutex.Unlock()
}

var (
	appPort      int = 30000
	appPortMutex sync.Mutex
)

func getAppPort() (port int) {
	appPortMutex.Lock()
	port = appPort
	if appPort+1 >= 60000 {
		appPort = 30000
	} else {
		appPort++
	}
	appPortMutex.Unlock()
	return
}

func (t *Transport) ListenForApp(fn func(port int)) (err error) {
	t.fieldsMutex.Lock()
	defer t.fieldsMutex.Unlock()
	if t.appNet != nil {
		return
	}

	var ln net.Listener
	var port int
	for i := 0; i < 3; i++ {
		port = getAppPort()
		address := net.JoinHostPort("", strconv.Itoa(port))
		ln, err = net.Listen("tcp", address)
		if err == nil {
			goto OK
		}
	}
	err = errors.New("can not listen for app")
	return

OK:
	t.appNet = ln
	t.servingPort = port

	fn(port)

	go t.accept()
	return
}

const (
	PKG_HEADER_ID_SIZE = 4
	PKG_HEADER_OP_SIZE = 1

	PKG_HEADER_BEGIN = 0
	PKG_HEADER_OP_BEGIN
	PKG_HEADER_OP_END = PKG_HEADER_OP_BEGIN + PKG_HEADER_OP_SIZE
	PKG_HEADER_ID_BEGIN
	PKG_HEADER_ID_END = PKG_HEADER_ID_BEGIN + PKG_HEADER_ID_SIZE
	PKG_HEADER_END
)

const (
	OP_TRANSPORT = iota
	OP_CLOSE
	OP_SHUTDOWN
)

func (t *Transport) accept() {
	t.fieldsMutex.RLock()
	tConn := t.conn
	t.fieldsMutex.RUnlock()

	go t.nodeReadLoop(tConn, func(id uint32) net.Conn {
		t.connsMutex.RLock()
		conn := t.conns[id]
		t.connsMutex.RUnlock()
		return conn
	})
	var idSeq uint32
	for {
		conn, err := t.appNet.Accept()
		if err != nil {
			return
		}
		id := atomic.AddUint32(&idSeq, 1)
		t.connsMutex.Lock()
		t.conns[id] = conn
		t.connsMutex.Unlock()
		go t.appReadLoop(id, conn, tConn, true)
	}
}

func (t *Transport) getDiscoveryKey() cipher.PubKey {
	if t.discoveryConn == nil {
		return EMPATY_PUBLIC_KEY
	}
	return t.discoveryConn.GetTargetKey()
}

func (t *Transport) Close() {
	t.fieldsMutex.Lock()
	defer t.fieldsMutex.Unlock()

	if t.factory == nil {
		return
	}
	t.sendLastTicket()

	var key cipher.PubKey
	if t.clientSide {
		key = t.ToApp
	} else {
		key = t.FromApp
	}
	tr, ok := t.appConnHolder.getTransport(key)
	if !ok || !t.clientSide || tr == t {
		msg := PriorityMsg{
			Priority: TransportClosed,
			Msg:      fmt.Sprintf("Discovery(%s): Transport closed", t.getDiscoveryKey().Hex()),
			Type:     Failed,
		}
		t.appConnHolder.PutMessage(msg)
		t.appConnHolder.SetAppFeedback(&AppFeedback{
			Discovery: t.getDiscoveryKey(),
			App:       key,
			Failed:    true,
			Msg:       msg,
		})
		t.appConnHolder.deleteTransport(key)
	}

	if t.timeoutTimer != nil {
		t.timeoutTimer.Stop()
	}
	t.connsMutex.RLock()
	for _, v := range t.conns {
		if v == nil {
			continue
		}
		v.Close()
	}
	t.connsMutex.RUnlock()
	if t.appNet != nil {
		t.appNet.Close()
		t.appNet = nil
	}
	if t.conn != nil {
		t.conn.Close()
		t.conn = nil
	}
	t.factory.Close()
	t.factory = nil
}

func (t *Transport) IsClientSide() bool {
	t.fieldsMutex.RLock()
	defer t.fieldsMutex.RUnlock()

	return t.clientSide
}

func writeAll(conn io.Writer, m []byte) error {
	for i := 0; i < len(m); {
		n, err := conn.Write(m[i:])
		if err != nil {
			return err
		}
		i += n
	}
	return nil
}

func (t *Transport) GetServingPort() int {
	t.fieldsMutex.RLock()
	port := t.servingPort
	t.fieldsMutex.RUnlock()
	return port
}

func (t *Transport) SetupTimeout() {
	t.fieldsMutex.Lock()
	if t.timeoutTimer != nil {
		t.timeoutTimer.Stop()
	}
	t.timeoutTimer = time.AfterFunc(30*time.Second, func() {
		t.appConnHolder.PutMessage(PriorityMsg{
			Type:     Failed,
			Msg:      "Timeout",
			Priority: Timeout,
		})
		t.Close()
	})
	t.fieldsMutex.Unlock()
}

func (t *Transport) StopTimeout() {
	t.fieldsMutex.Lock()
	if t.timeoutTimer != nil {
		t.timeoutTimer.Stop()
	}
	t.timeoutTimer = nil
	t.fieldsMutex.Unlock()
}

type bandwidth struct {
	bytes     uint
	lastBytes uint
	sec       int64
	total     uint
	sync.RWMutex
}

func (b *bandwidth) add(s int) {
	b.Lock()
	now := time.Now().Unix()
	if b.sec != now {
		b.sec = now
		b.total += b.lastBytes
		b.lastBytes = b.bytes
		b.bytes = uint(s)
		b.Unlock()
		return
	}
	b.bytes += uint(s)
	b.Unlock()
}

// Bandwidth bytes/sec
func (b *bandwidth) get() (r uint) {
	now := time.Now().Unix()
	b.RLock()
	if now != b.sec {
		r = 0
		b.RUnlock()
		return
	}
	r = b.lastBytes
	b.RUnlock()
	return
}

func (b *bandwidth) getTotal() (r uint) {
	b.RLock()
	r = b.total + b.lastBytes + b.bytes
	b.RUnlock()
	return
}

func (t *Transport) GetUploadBandwidth() uint {
	return t.uploadBW.get()
}

func (t *Transport) GetDownloadBandwidth() uint {
	return t.downloadBW.get()
}

func (t *Transport) GetUploadTotal() uint {
	return t.uploadBW.getTotal()
}

func (t *Transport) GetDownloadTotal() uint {
	return t.downloadBW.getTotal()
}
