package app2

import (
	"fmt"
	"net"

	"github.com/pkg/errors"
	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/app2/network"
)

// RPCGateway is a RPC interface for the app server.
type RPCGateway struct {
	lm  *manager
	cm  *manager
	log *logging.Logger
}

// newRPCGateway constructs new server RPC interface.
func newRPCGateway(log *logging.Logger) *RPCGateway {
	return &RPCGateway{
		lm:  newManager(),
		cm:  newManager(),
		log: log,
	}
}

// Dial dials to the remote.
func (r *RPCGateway) Dial(remote *network.Addr, connID *uint16) error {
	reservedConnID, err := r.cm.nextKey()
	if err != nil {
		return err
	}

	conn, err := network.Dial(*remote)
	if err != nil {
		return err
	}

	if err := r.cm.set(*reservedConnID, conn); err != nil {
		if err := conn.Close(); err != nil {
			r.log.WithError(err).Error("error closing conn")
		}

		return err
	}

	*connID = *reservedConnID

	return nil
}

// Listen starts listening.
func (r *RPCGateway) Listen(local *network.Addr, lisID *uint16) error {
	nextLisID, err := r.lm.nextKey()
	if err != nil {
		return err
	}

	l, err := network.Listen(*local)
	if err != nil {
		return err
	}

	if err := r.lm.set(*nextLisID, l); err != nil {
		if err := l.Close(); err != nil {
			r.log.WithError(err).Error("error closing listener")
		}

		return err
	}

	*lisID = *nextLisID

	return nil
}

// AcceptResp contains response parameters for `Accept`.
type AcceptResp struct {
	Remote network.Addr
	ConnID uint16
}

// Accept accepts connection from the listener specified by `lisID`.
func (r *RPCGateway) Accept(lisID *uint16, resp *AcceptResp) error {
	lis, err := r.getListener(*lisID)
	if err != nil {
		return err
	}

	connID, err := r.cm.nextKey()
	if err != nil {
		return err
	}

	conn, err := lis.Accept()
	if err != nil {
		return err
	}

	if err := r.cm.set(*connID, conn); err != nil {
		if err := conn.Close(); err != nil {
			r.log.WithError(err).Error("error closing DMSG transport")
		}

		return err
	}

	remote, ok := conn.RemoteAddr().(network.Addr)
	if !ok {
		return errors.New("wrong type for remote addr")
	}

	resp = &AcceptResp{
		Remote: remote,
		ConnID: *connID,
	}

	return nil
}

// WriteReq contains arguments for `Write`.
type WriteReq struct {
	ConnID uint16
	B      []byte
}

// Write writes to the connection.
func (r *RPCGateway) Write(req *WriteReq, n *int) error {
	conn, err := r.getConn(req.ConnID)
	if err != nil {
		return err
	}

	*n, err = conn.Write(req.B)
	if err != nil {
		return err
	}

	return nil
}

// ReadResp contains response parameters for `Read`.
type ReadResp struct {
	B []byte
	N int
}

// Read reads data from connection specified by `connID`.
func (r *RPCGateway) Read(connID *uint16, resp *ReadResp) error {
	conn, err := r.getConn(*connID)
	if err != nil {
		return err
	}

	resp.N, err = conn.Read(resp.B)
	if err != nil {
		return err
	}

	return nil
}

// CloseConn closes connection specified by `connID`.
func (r *RPCGateway) CloseConn(connID *uint16, _ *struct{}) error {
	conn, err := r.popConn(*connID)
	if err != nil {
		return err
	}

	return conn.Close()
}

// CloseListener closes listener specified by `lisID`.
func (r *RPCGateway) CloseListener(lisID *uint16, _ *struct{}) error {
	lis, err := r.popListener(*lisID)
	if err != nil {
		return err
	}

	return lis.Close()
}

// popListener gets listener from the manager by `lisID` and removes it.
// Handles type assertion.
func (r *RPCGateway) popListener(lisID uint16) (net.Listener, error) {
	lisIfc, err := r.lm.pop(lisID)
	if err != nil {
		return nil, err
	}

	return r.assertListener(lisIfc)
}

// popConn gets conn from the manager by `connID` and removes it.
// Handles type assertion.
func (r *RPCGateway) popConn(connID uint16) (net.Conn, error) {
	connIfc, err := r.cm.pop(connID)
	if err != nil {
		return nil, err
	}

	return r.assertConn(connIfc)
}

// getListener gets listener from the manager by `lisID`. Handles type assertion.
func (r *RPCGateway) getListener(lisID uint16) (net.Listener, error) {
	lisIfc, ok := r.lm.get(lisID)
	if !ok {
		return nil, fmt.Errorf("no listener with key %d", lisID)
	}

	return r.assertListener(lisIfc)
}

// getConn gets conn from the manager by `connID`. Handles type assertion.
func (r *RPCGateway) getConn(connID uint16) (net.Conn, error) {
	connIfc, ok := r.cm.get(connID)
	if !ok {
		return nil, fmt.Errorf("no conn with key %d", connID)
	}

	return r.assertConn(connIfc)
}

// assertListener asserts that `v` is of type `net.Listener`.
func (r *RPCGateway) assertListener(v interface{}) (net.Listener, error) {
	lis, ok := v.(net.Listener)
	if !ok {
		return nil, errors.New("wrong type of value stored for listener")
	}

	return lis, nil
}

// assertConn asserts that `v` is of type `net.Conn`.
func (r *RPCGateway) assertConn(v interface{}) (net.Conn, error) {
	conn, ok := v.(net.Conn)
	if !ok {
		return nil, errors.New("wrong type of value stored for conn")
	}

	return conn, nil
}
