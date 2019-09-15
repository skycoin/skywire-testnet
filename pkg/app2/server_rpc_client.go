package app2

import (
	"net/rpc"

	"github.com/skycoin/skywire/pkg/routing"
)

type ServerRPCClient interface {
	Dial(remote routing.Addr) (uint16, error)
	Listen(local routing.Addr) (uint16, error)
	Accept(lisID uint16) (uint16, error)
	Write(connID uint16, b []byte) (int, error)
	Read(connID uint16, b []byte) (int, error)
	CloseConn(id uint16) error
	CloseListener(id uint16) error
}

type ListenerRPCClient interface {
	Accept(id uint16) (uint16, error)
	CloseListener(id uint16) error
	Write(connID uint16, b []byte) (int, error)
	Read(connID uint16, b []byte) (int, error)
	CloseConn(id uint16) error
}

type ConnRPCClient interface {
	Write(id uint16, b []byte) (int, error)
	Read(id uint16, b []byte) (int, error)
	CloseConn(id uint16) error
}

type serverRPCCLient struct {
	rpc *rpc.Client
}

func newServerRPCClient(rpc *rpc.Client) ServerRPCClient {
	return &serverRPCCLient{
		rpc: rpc,
	}
}

func (c *serverRPCCLient) Dial(remote routing.Addr) (uint16, error) {
	var connID uint16
	if err := c.rpc.Call("Dial", &remote, &connID); err != nil {
		return 0, err
	}

	return connID, nil
}

func (c *serverRPCCLient) Listen(local routing.Addr) (uint16, error) {
	var lisID uint16
	if err := c.rpc.Call("Listen", &local, &lisID); err != nil {
		return 0, err
	}

	return lisID, nil
}

func (c *serverRPCCLient) Accept(lisID uint16) (uint16, error) {
	var connID uint16
	if err := c.rpc.Call("Accept", &lisID, &connID); err != nil {
		return 0, err
	}

	return connID, nil
}

func (c *serverRPCCLient) Write(connID uint16, b []byte) (int, error) {
	req := WriteReq{
		ConnID: connID,
		B:      b,
	}

	var n int
	if err := c.rpc.Call("Write", &req, &n); err != nil {
		return n, err
	}

	return n, nil
}

func (c *serverRPCCLient) Read(connID uint16, b []byte) (int, error) {
	var resp ReadResp
	if err := c.rpc.Call("Read", &connID, &resp); err != nil {
		return 0, err
	}

	copy(b[:resp.N], resp.B[:resp.N])

	return resp.N, nil
}

func (c *serverRPCCLient) CloseConn(id uint16) error {
	return c.rpc.Call("CloseConn", &id, nil)
}

func (c *serverRPCCLient) CloseListener(id uint16) error {
	return c.rpc.Call("CloseListener", &id, nil)
}
