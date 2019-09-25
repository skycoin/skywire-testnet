package app2

import (
	"net/rpc"

	"github.com/skycoin/skywire/pkg/app2/network"
	"github.com/skycoin/skywire/pkg/routing"
)

//go:generate mockery -name RPCClient -case underscore -inpkg

// RPCClient describes RPC interface to communicate with the server.
type RPCClient interface {
	Dial(remote network.Addr) (connID uint16, localPort routing.Port, err error)
	Listen(local network.Addr) (uint16, error)
	Accept(lisID uint16) (connID uint16, remote network.Addr, err error)
	Write(connID uint16, b []byte) (int, error)
	Read(connID uint16, b []byte) (int, []byte, error)
	CloseConn(id uint16) error
	CloseListener(id uint16) error
}

// rpcClient implements `RPCClient`.
type rpcCLient struct {
	rpc *rpc.Client
}

// NewRPCClient constructs new `rpcClient`.
func NewRPCClient(rpc *rpc.Client) RPCClient {
	return &rpcCLient{
		rpc: rpc,
	}
}

// Dial sends `Dial` command to the server.
func (c *rpcCLient) Dial(remote network.Addr) (connID uint16, localPort routing.Port, err error) {
	var resp DialResp
	if err := c.rpc.Call("Dial", &remote, &resp); err != nil {
		return 0, 0, err
	}

	return resp.ConnID, resp.LocalPort, nil
}

// Listen sends `Listen` command to the server.
func (c *rpcCLient) Listen(local network.Addr) (uint16, error) {
	var lisID uint16
	if err := c.rpc.Call("Listen", &local, &lisID); err != nil {
		return 0, err
	}

	return lisID, nil
}

// Accept sends `Accept` command to the server.
func (c *rpcCLient) Accept(lisID uint16) (connID uint16, remote network.Addr, err error) {
	var acceptResp AcceptResp
	if err := c.rpc.Call("Accept", &lisID, &acceptResp); err != nil {
		return 0, network.Addr{}, err
	}

	return acceptResp.ConnID, acceptResp.Remote, nil
}

// Write sends `Write` command to the server.
func (c *rpcCLient) Write(connID uint16, b []byte) (int, error) {
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

// Read sends `Read` command to the server.
func (c *rpcCLient) Read(connID uint16, b []byte) (int, []byte, error) {
	req := ReadReq{
		ConnID: connID,
		BufLen: len(b),
	}

	var resp ReadResp
	if err := c.rpc.Call("Read", &req, &resp); err != nil {
		return 0, nil, err
	}

	copy(b[:resp.N], resp.B[:resp.N])

	return resp.N, resp.B, nil
}

// CloseConn sends `CloseConn` command to the server.
func (c *rpcCLient) CloseConn(id uint16) error {
	return c.rpc.Call("CloseConn", &id, nil)
}

// CloseListener sends `CloseListener` command to the server.
func (c *rpcCLient) CloseListener(id uint16) error {
	return c.rpc.Call("CloseListener", &id, nil)
}
