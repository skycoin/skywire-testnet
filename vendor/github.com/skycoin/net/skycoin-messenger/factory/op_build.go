package factory

import (
	"sync"

	"net"

	"github.com/skycoin/skycoin/src/cipher"
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
	resps[OP_FORWARD_NODE_CONN] = &sync.Pool{
		New: func() interface{} {
			return new(buildConnResp)
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
}

type appConn struct {
	Node cipher.PubKey
	App  cipher.PubKey
}

// run on node A
func (req *appConn) Execute(f *MessengerFactory, conn *Connection) (r resp, err error) {
	if !f.Proxy {
		return
	}

	f.ForEachConn(func(connection *Connection) {
		fromNode := connection.GetKey()
		fromApp := conn.GetKey()
		tr := NewTransport(f, fromNode, req.Node, fromApp, req.App)
		conn.GetContextLogger().Debugf("app conn create transport to %s", connection.GetRemoteAddr().String())
		c, err := tr.ListenAndConnect(connection.GetRemoteAddr().String())
		if err != nil {
			conn.GetContextLogger().Debugf("transport err %v", err)
			return
		}
		c.writeOP(OP_FORWARD_NODE_CONN, &forwardNodeConn{Node: req.Node, App: req.App, FromApp: fromApp, FromNode: fromNode})
		conn.setTransport(req.App, tr)
	})
	return
}

type AppConnResp struct {
	Host string `json:",omitempty"`
	Port int
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
		conn.appConnectionInitCallback(req)
	}
	return
}

type buildConnResp buildConn

// run on node A, conn is udp from node B
func (req *buildConnResp) Execute(f *MessengerFactory, conn *Connection) (r resp, err error) {
	appConn, ok := f.GetConnection(req.FromApp)
	if !ok {
		conn.GetContextLogger().Debugf("buildConnResp app %x not found", req.FromApp)
		return
	}
	tr, ok := appConn.getTransport(req.App)
	if !ok {
		conn.GetContextLogger().Debugf("buildConnResp tr %x not found", req.App)
		return
	}
	conn.GetContextLogger().Debugf("recv %#v tr %#v", req, tr)
	tr.setUDPConn(conn)
	conn.writeOP(OP_APP_CONN_ACK|RESP_PREFIX, &connAck{})
	fnOK := func(port int) {
		appConn.writeOP(OP_BUILD_APP_CONN|RESP_PREFIX, &AppConnResp{Port: port})
	}
	err = tr.ListenForApp(fnOK)
	if err != nil {
		conn.GetContextLogger().Debugf("ListenForApp err %v", err)
		return
	}
	err = ErrDetach
	return
}

// run on node A, from manager udp
func (req *buildConnResp) Run(conn *Connection) (err error) {
	tr, ok := conn.getTransport(req.App)
	if !ok {
		conn.GetContextLogger().Debugf("buildConnResp run tr %#v not found", req)
		return
	}
	conn.GetContextLogger().Debugf("recv %#v tr %#v", req, tr)
	return
}

type forwardNodeConn struct {
	Node     cipher.PubKey
	App      cipher.PubKey
	FromApp  cipher.PubKey
	FromNode cipher.PubKey
}

// run on manager, conn is udp conn from node A
func (req *forwardNodeConn) Execute(f *MessengerFactory, conn *Connection) (r resp, err error) {
	c, ok := f.GetConnection(req.Node)
	if !ok {
		conn.GetContextLogger().Debugf("node %x not exists", req.Node)
		return
	}

	_, ok = c.getService(req.App)
	if !ok {
		conn.GetContextLogger().Debugf("node %x app %x not exists", req.Node, req.App)
		return
	}

	conn.GetContextLogger().Debugf("conn remote addr %v", conn.GetRemoteAddr())
	err = c.writeOP(OP_BUILD_NODE_CONN|RESP_PREFIX,
		&buildConn{
			Address:  conn.GetRemoteAddr().String(),
			Node:     req.Node,
			App:      req.App,
			FromApp:  req.FromApp,
			FromNode: req.FromNode,
		})
	return
}

type forwardNodeConnResp forwardNodeConn

// run on manager, conn is udp conn from node B
func (req *forwardNodeConnResp) Execute(f *MessengerFactory, conn *Connection) (r resp, err error) {
	c, ok := f.GetConnection(req.FromNode)
	if !ok {
		conn.GetContextLogger().Debugf("node %x not exists", req.FromNode)
		return
	}

	err = c.writeOP(OP_FORWARD_NODE_CONN|RESP_PREFIX,
		&buildConnResp{
			Address:  conn.GetRemoteAddr().String(),
			Node:     req.Node,
			App:      req.App,
			FromApp:  req.FromApp,
			FromNode: req.FromNode,
		})
	return
}

type buildConn struct {
	Address  string
	Node     cipher.PubKey
	App      cipher.PubKey
	FromApp  cipher.PubKey
	FromNode cipher.PubKey
}

func (req *buildConn) Run(conn *Connection) (err error) {
	appConn, ok := conn.factory.GetConnection(req.App)
	if !ok {
		conn.GetContextLogger().Debugf("node %x app %x not exists", req.Node, req.App)
		return
	}

	s, ok := appConn.getService(req.App)
	if !ok {
		conn.GetContextLogger().Debugf("node %x app %x not exists", req.Node, req.App)
		return
	}

	tr := NewTransport(conn.factory, req.FromNode, req.Node, req.FromApp, req.App)
	connection, err := tr.ListenAndConnect(conn.GetRemoteAddr().String())
	if err != nil {
		return
	}
	err = connection.writeOP(OP_FORWARD_NODE_CONN_RESP, &forwardNodeConnResp{Node: req.Node, App: req.App, FromApp: req.FromApp, FromNode: req.FromNode})
	if err != nil {
		return
	}
	err = tr.Connect(req.Address, s.Address)
	return
}

type connAck struct {
}

// run on node b from node a udp
func (req *connAck) Run(conn *Connection) (err error) {
	err = ErrDetach
	return
}
