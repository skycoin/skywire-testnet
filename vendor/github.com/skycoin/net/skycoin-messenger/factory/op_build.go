package factory

import (
	"crypto/aes"
	"crypto/rand"
	"fmt"
	"github.com/skycoin/skycoin/src/cipher"
	"io"
	"net"
	"sync"
)

func init() {
	ops[OP_BUILD_APP_CONN] = &sync.Pool{
		New: func() interface{} {
			return new(appConn)
		},
	}
	ops[OP_FORWARD_NODE_CONN] = &sync.Pool{
		New: func() interface{} {
			return new(forwardNodeConn)
		},
	}
	resps[OP_BUILD_NODE_CONN] = &sync.Pool{
		New: func() interface{} {
			return new(buildConn)
		},
	}
	ops[OP_FORWARD_NODE_CONN_RESP] = &sync.Pool{
		New: func() interface{} {
			return new(forwardNodeConnResp)
		},
	}
	resps[OP_FORWARD_NODE_CONN_RESP] = &sync.Pool{
		New: func() interface{} {
			return new(forwardNodeConnResp)
		},
	}
	ops[OP_BUILD_APP_CONN_OK] = &sync.Pool{
		New: func() interface{} {
			return new(buildConnResp)
		},
	}
	resps[OP_BUILD_APP_CONN] = &sync.Pool{
		New: func() interface{} {
			return new(AppConnResp)
		},
	}
	resps[OP_APP_CONN_ACK] = &sync.Pool{
		New: func() interface{} {
			return new(connAck)
		},
	}
	ops[OP_APP_FEEDBACK] = &sync.Pool{
		New: func() interface{} {
			return new(AppFeedback)
		},
	}
	resps[OP_BUILD_APP_CONN_OK] = &sync.Pool{
		New: func() interface{} {
			return new(nop)
		},
	}
}

type appConn struct {
	Node      cipher.PubKey
	App       cipher.PubKey
	Discovery cipher.PubKey
}

// run on node A
func (req *appConn) Execute(f *MessengerFactory, conn *Connection) (r resp, err error) {
	if !f.Proxy {
		return
	}

	sent := make(map[string]struct{})
	f.ForEachConn(func(connection *Connection) {
		discoveryKey := connection.GetTargetKey()
		if discoveryKey != req.Discovery && req.Discovery != EMPATY_PUBLIC_KEY {
			return
		}
		_, ok := sent[discoveryKey.Hex()]
		if ok {
			return
		}
		sent[discoveryKey.Hex()] = struct{}{}
		fromNode := connection.GetKey()
		fromApp := conn.GetKey()
		iv := make([]byte, aes.BlockSize)
		if _, err = io.ReadFull(rand.Reader, iv); err != nil {
			conn.GetContextLogger().Debugf("transport err %v", err)
			return
		}
		tr := NewTransport(f, conn, fromNode, req.Node, fromApp, req.App)
		tr.SetOnAcceptedUDPCallback(func(connection *Connection) {
			connection.CreatedByTransport = tr
			sc := f.GetDefaultSeedConfig()
			connection.GetContextLogger().Debugf("set crypto sc %v", sc)
			if sc == nil {
				connection.GetContextLogger().Debugf("tr sc is nil")
			}
			connection.SetKey(req.Node)
			err := connection.SetCrypto(sc.publicKey, sc.secKey, req.Node, iv)
			if err != nil {
				connection.GetContextLogger().Debugf("set crypto err %v", err)
			}
		})
		conn.GetContextLogger().Debugf("app conn create transport to %s", connection.GetRemoteAddr().String())
		c, err := tr.ListenAndConnect(connection.GetRemoteAddr().String(), discoveryKey)
		if err != nil {
			conn.GetContextLogger().Debugf("transport err %v", err)
			return
		}
		nodeConn := &forwardNodeConn{
			Node:     req.Node,
			App:      req.App,
			FromApp:  fromApp,
			FromNode: fromNode,
			Num:      iv,
		}
		c.writeOP(OP_FORWARD_NODE_CONN, nodeConn)
		tr.SetupTimeout()
		conn.setTransport(discoveryKey, tr)
	})
	return
}

type Priority int
type MsgType int

const (
	Success MsgType = iota
	Failed
)

const (
	_ Priority = iota
	Building
	NotFound
	NotAllowed
	Connected
	Timeout
	TransportClosed
)

type PriorityMsg struct {
	Priority Priority `json:"priority"`
	Msg      string   `json:"msg"`
	Type     MsgType  `json:"type"`
	Time     int64    `json:"time"`
}

type AppConnResp struct {
	Discovery cipher.PubKey
	App       cipher.PubKey
	Host      string `json:",omitempty"`
	Port      int
	Failed    bool
	Msg       PriorityMsg
}

// run on app
func (req *AppConnResp) Run(conn *Connection) (err error) {
	conn.GetContextLogger().Debugf("recv %#v", req)
	if conn.appConnectionInitCallback != nil {
		addr := conn.GetRemoteAddr().String()
		host, _, err := net.SplitHostPort(addr)
		if err != nil {
			return err
		}
		req.Host = host
		fb := conn.appConnectionInitCallback(req)
		fb.App = req.App
		fb.Discovery = req.Discovery
		err = conn.writeOP(OP_APP_FEEDBACK, fb)
	}
	return
}

type AppFeedback struct {
	Discovery cipher.PubKey
	// to app
	App    cipher.PubKey
	Port   int         `json:"port"`
	Failed bool        `json:"failed"`
	Msg    PriorityMsg `json:"msg"`
}

func (req *AppFeedback) Execute(f *MessengerFactory, conn *Connection) (r resp, err error) {
	conn.GetContextLogger().Debugf("recv %#v", req)
	conn.SetAppFeedback(req)
	tr, ok := conn.getTransport(req.App)
	if !ok {
		conn.GetContextLogger().Debugf("AppFeedback tr %x not found", req.App)
		return
	}
	tr.StopTimeout()
	return
}

type buildConnResp buildConn

// run on node A, conn is udp from node B
func (req *buildConnResp) Execute(f *MessengerFactory, conn *Connection) (r resp, err error) {
	conn.GetContextLogger().Debugf("buildConnResp %#v", req)
	appConn, ok := f.Parent.GetConnection(req.FromApp)
	if !ok {
		err = fmt.Errorf("buildConnResp app %x not found", req.FromApp)
		return
	}
	tr := conn.CreatedByTransport
	if tr == nil {
		err = fmt.Errorf("buildConnResp tr %x not found", req.App)
		return
	}
	tr.setUDPConn(conn)
	tr.connAck()
	exists := appConn.setTransportIfNotExists(req.App, tr)
	if exists {
		tr.Close()
		conn.GetContextLogger().Debugf("buildConnResp transport exists")
		return
	}
	fnOK := func(port int) {
		msg := fmt.Sprintf("Discovery(%s): Connected app %x",
			tr.getDiscoveryKey().Hex(), req.App)
		priorityMsg := PriorityMsg{Priority: Connected, Msg: msg}
		appConn.PutMessage(priorityMsg)
		appConn.writeOP(OP_BUILD_APP_CONN|RESP_PREFIX, &AppConnResp{
			Discovery: tr.getDiscoveryKey(),
			App:       req.App,
			Port:      port,
			Msg:       priorityMsg,
		})
	}
	err = tr.ListenForApp(fnOK)
	if err != nil {
		err = fmt.Errorf("ListenForApp err %v", err)
		return
	}
	err = conn.writeOP(OP_APP_CONN_ACK|RESP_PREFIX, &connAck{
		FromApp: req.FromApp,
		App:     req.App,
	})
	if err != nil {
		err = fmt.Errorf("buildConnResp err %v", err)
		return
	}
	conn.GetContextLogger().Debugf("buildConnResp detach")
	err = ErrDetach
	return
}

type forwardNodeConn struct {
	Node     cipher.PubKey
	App      cipher.PubKey
	FromApp  cipher.PubKey
	FromNode cipher.PubKey
	Num      []byte
}

// run on manager, conn is udp conn from node A
func (req *forwardNodeConn) Execute(f *MessengerFactory, conn *Connection) (r resp, err error) {
	c, ok := f.GetConnection(req.Node)
	if !ok {
		cause := fmt.Sprintf("Node %x not exists", req.Node)
		conn.GetContextLogger().Debugf(cause)
		err = conn.writeOP(OP_FORWARD_NODE_CONN_RESP|RESP_PREFIX, &forwardNodeConnResp{
			Node:     req.Node,
			App:      req.App,
			FromApp:  req.FromApp,
			FromNode: req.FromNode,
			Failed:   true,
			Msg:      PriorityMsg{Priority: NotFound, Msg: cause, Type: Failed},
			Num:      req.Num,
		})
		return
	}

	conn.GetContextLogger().Debugf("conn remote addr %v", conn.GetRemoteAddr())
	p := globalTransportPairManagerInstance.create(req.FromApp, req.FromNode, req.Node, req.App)
	err = p.setFromConn(conn)
	if err != nil {
		err = fmt.Errorf("set from Conn err: %s", err)
		return
	}
	conn.SetTransportPair(p)
	err = c.writeOP(OP_BUILD_NODE_CONN|RESP_PREFIX,
		&buildConn{
			Address:  conn.GetRemoteAddr().String(),
			Node:     req.Node,
			App:      req.App,
			FromApp:  req.FromApp,
			FromNode: req.FromNode,
			Num:      req.Num,
		})
	return
}

type forwardNodeConnResp struct {
	Node     cipher.PubKey
	App      cipher.PubKey
	FromApp  cipher.PubKey
	FromNode cipher.PubKey
	Failed   bool
	Msg      PriorityMsg
	Address  string
	Num      []byte
}

// run on manager, conn is tcp/udp from node B
func (req *forwardNodeConnResp) Execute(f *MessengerFactory, conn *Connection) (r resp, err error) {
	c, ok := f.GetConnection(req.FromNode)
	if !ok {
		conn.GetContextLogger().Debugf("node %x not exists", req.FromNode)
		return
	}

	if conn.IsUDP() {
		req.Address = conn.GetRemoteAddr().String()
		if !req.Failed {
			p, ok := globalTransportPairManagerInstance.get(req.FromApp, req.FromNode, req.Node, req.App)
			if !ok {
				err = fmt.Errorf("conn transport pair not exists!? %#v", req)
				return
			}
			p.ok()
			err = p.setToConn(conn)
			if err != nil {
				err = fmt.Errorf("set to Conn: %s", err)
				return
			}
			conn.SetTransportPair(p)
		}
	}
	err = c.writeOP(OP_FORWARD_NODE_CONN_RESP|RESP_PREFIX, req)
	return
}

// run on node A, from manager
func (req *forwardNodeConnResp) Run(conn *Connection) (err error) {
	factory := conn.factory.Parent
	if factory == nil {
		factory = conn.factory
	}
	appConn, ok := factory.GetConnection(req.FromApp)
	if !ok {
		conn.GetContextLogger().Debugf("forwardNodeConnResp app %x not found", req.FromApp)
		return
	}
	tr, ok := appConn.getTransport(conn.GetTargetKey())
	if !ok {
		conn.GetContextLogger().Debugf("forwardNodeConnResp tr %s not found", req.App.Hex())
		return
	}
	appConn.deleteTransport(conn.GetTargetKey())
	if tr.isConnAck() {
		return
	}
	req.Msg.Msg = "Discovery(" + conn.GetTargetKey().Hex() + "): " + req.Msg.Msg
	appConn.PutMessage(req.Msg)
	if req.Failed {
		appConn.writeOP(OP_BUILD_APP_CONN|RESP_PREFIX, &AppConnResp{
			Discovery: conn.GetTargetKey(),
			App:       req.App,
			Failed:    req.Failed,
			Msg:       req.Msg,
		})
		tr.Close()
		return
	}
	if len(req.Address) > 0 {
		e := tr.clientSideConnect(req.Address, conn.factory.GetDefaultSeedConfig(), req.Num)
		if e != nil {
			conn.GetContextLogger().Debugf("forwardNodeConnResp clientSideConnect %v", e)
		}
	}
	return
}

type buildConn struct {
	Address  string
	Node     cipher.PubKey
	App      cipher.PubKey
	FromApp  cipher.PubKey
	FromNode cipher.PubKey
	Num      []byte
}

func (req *buildConn) Run(conn *Connection) (err error) {
	appConn, ok := conn.factory.GetConnection(req.App)
	if !ok {
		cause := fmt.Sprintf("Node %x app %x not exists", req.Node, req.App)
		conn.GetContextLogger().Debugf(cause)
		err = conn.writeOP(OP_FORWARD_NODE_CONN_RESP, &forwardNodeConnResp{
			Node:     req.Node,
			App:      req.App,
			FromApp:  req.FromApp,
			FromNode: req.FromNode,
			Failed:   true,
			Msg:      PriorityMsg{Priority: NotFound, Msg: cause, Type: Failed},
			Num:      req.Num,
		})
		return
	}

	s, ok := appConn.getService(req.App)
	if !ok {
		cause := fmt.Sprintf("Node %x app %x not exists", req.Node, req.App)
		conn.GetContextLogger().Debugf(cause)
		err = conn.writeOP(OP_FORWARD_NODE_CONN_RESP, &forwardNodeConnResp{
			Node:     req.Node,
			App:      req.App,
			FromApp:  req.FromApp,
			FromNode: req.FromNode,
			Failed:   true,
			Msg:      PriorityMsg{Priority: NotFound, Msg: cause, Type: Failed},
			Num:      req.Num,
		})
		return
	}

	if len(s.AllowNodes) > 0 {
		allow := false
		for _, k := range s.AllowNodes {
			if k == req.FromNode.Hex() {
				allow = true
				break
			}
		}
		if !allow {
			cause := fmt.Sprintf("Node %x app %x forbid %x", req.Node, req.App, req.FromNode)
			conn.GetContextLogger().Debugf(cause)
			err = conn.writeOP(OP_FORWARD_NODE_CONN_RESP, &forwardNodeConnResp{
				Node:     req.Node,
				App:      req.App,
				FromApp:  req.FromApp,
				FromNode: req.FromNode,
				Failed:   true,
				Msg:      PriorityMsg{Priority: NotAllowed, Msg: cause, Type: Failed},
				Num:      req.Num,
			})
			return
		}
	}

	tr := NewTransport(conn.factory, appConn, req.FromNode, req.Node, req.FromApp, req.App)
	connection, err := tr.ListenAndConnect(conn.GetRemoteAddr().String(), conn.GetTargetKey())
	if err != nil {
		return
	}
	err = connection.writeOP(OP_FORWARD_NODE_CONN_RESP, &forwardNodeConnResp{
		Node:     req.Node,
		App:      req.App,
		FromApp:  req.FromApp,
		FromNode: req.FromNode,
		Msg:      PriorityMsg{Priority: Building, Msg: "Building connection"},
		Num:      req.Num,
	})
	if err != nil {
		return
	}
	err = tr.serverSiceConnect(req.Address, s.Address, conn.factory.GetDefaultSeedConfig(), req.Num)
	tr.SetupTimeout()
	return
}

type connAck struct {
	FromApp, App cipher.PubKey
}

// run on node b from node a udp
func (req *connAck) Run(conn *Connection) (err error) {
	conn.GetContextLogger().Debugf("recv conn ack %s", req.App.Hex())
	tr := conn.CreatedByTransport
	if tr == nil {
		err = fmt.Errorf("tr %x not exists", tr)
		return
	}
	tr.StopTimeout()
	err = ErrDetach
	return
}

type nop struct {
}

// run on node b from node a udp
func (req *nop) Run(conn *Connection) (err error) {
	return
}
