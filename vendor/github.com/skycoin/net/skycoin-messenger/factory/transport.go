package factory

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
	"strconv"
	"sync"
	"sync/atomic"

	log "github.com/sirupsen/logrus"
	cn "github.com/skycoin/net/conn"
	"github.com/skycoin/skycoin/src/cipher"
	"time"
)

type transport struct {
	creator *MessengerFactory
	// node
	factory *MessengerFactory
	// conn between nodes
	conn *Connection
	// app
	appNet net.Listener

	FromNode, ToNode cipher.PubKey
	FromApp, ToApp   cipher.PubKey
	servingPort      int

	conns      map[uint32]net.Conn
	connsMutex sync.RWMutex

	timeoutTimer *time.Timer

	fieldsMutex sync.RWMutex
}

func NewTransport(creator *MessengerFactory, fromNode, toNode, fromApp, toApp cipher.PubKey) *transport {
	t := &transport{
		creator:  creator,
		FromNode: fromNode,
		ToNode:   toNode,
		FromApp:  fromApp,
		ToApp:    toApp,
		factory:  NewMessengerFactory(),
		conns:    make(map[uint32]net.Conn),
	}
	return t
}

// Listen and connect to node manager
func (t *transport) ListenAndConnect(address string) (conn *Connection, err error) {
	conn, err = t.factory.connectUDPWithConfig(address, &ConnConfig{
		Creator: t.creator,
	})
	return
}

// Connect to node A and server app
func (t *transport) Connect(address, appAddress string) (err error) {
	conn, err := t.factory.connectUDPWithConfig(address, &ConnConfig{
		OnConnected: func(connection *Connection) {
			connection.writeOP(OP_BUILD_APP_CONN_OK,
				&buildConnResp{
					FromNode: t.FromNode,
					Node:     t.ToNode,
					FromApp:  t.FromApp,
					App:      t.ToApp,
				})
		},
		Creator: t.creator,
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

func (t *transport) nodeReadLoop(conn *Connection, getAppConn func(id uint32) net.Conn) {
	defer func() {
		t.Close()
	}()
	var err error
	for {
		select {
		case m, ok := <-conn.GetChanIn():
			if !ok {
				log.Debugf("node conn read err %v", err)
				return
			}
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
			body := m[PKG_HEADER_END:]
			if len(body) < 1 {
				continue
			}
			err = writeAll(appConn, body)
			if err != nil {
				log.Debugf("app conn write err %v", err)
				continue
			}
		}
	}
}

func (t *transport) appReadLoop(id uint32, appConn net.Conn, conn *Connection, create bool) {
	buf := make([]byte, cn.MAX_UDP_PACKAGE_SIZE-100)
	binary.BigEndian.PutUint32(buf[PKG_HEADER_ID_BEGIN:PKG_HEADER_ID_END], id)
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
					conn.GetChanOut() <- buf[:PKG_HEADER_END]
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
		conn.GetChanOut() <- buf[:PKG_HEADER_END]
	}
	for {
		n, err := appConn.Read(buf[PKG_HEADER_END:])
		if err != nil {
			log.Debugf("app conn read err %v, %d", err, n)
			return
		}
		pkg := make([]byte, PKG_HEADER_END+n)
		copy(pkg, buf[:PKG_HEADER_END+n])
		conn.GetChanOut() <- pkg
	}
}

func (t *transport) setUDPConn(conn *Connection) {
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

func (t *transport) ListenForApp(fn func(port int)) (err error) {
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

func (t *transport) accept() {
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

func (t *transport) Close() {
	t.fieldsMutex.Lock()
	defer t.fieldsMutex.Unlock()

	if t.factory == nil {
		return
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

func (t *transport) GetServingPort() int {
	t.fieldsMutex.RLock()
	port := t.servingPort
	t.fieldsMutex.RUnlock()
	return port
}

func (t *transport) SetupTimeout(key cipher.PubKey, conn *Connection) {
	t.fieldsMutex.Lock()
	if t.timeoutTimer != nil {
		if !t.timeoutTimer.Stop() {
			<-t.timeoutTimer.C
		}
	}
	t.timeoutTimer = time.AfterFunc(30*time.Second, func() {
		t.Close()
		conn.setTransport(key, nil)
		conn.PutMessage(PriorityMsg{
			Type:     FAILED,
			Msg:      "Timeout",
			Priority: 100,
		})
	})
	t.fieldsMutex.Unlock()
}

func (t *transport) StopTimeout() {
	t.fieldsMutex.Lock()
	if t.timeoutTimer != nil {
		if !t.timeoutTimer.Stop() {
			<-t.timeoutTimer.C
		}
	}
	t.timeoutTimer = nil
	t.fieldsMutex.Unlock()
}
