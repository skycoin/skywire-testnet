package app2

import (
	"context"
	"fmt"
	"net"

	"github.com/pkg/errors"
	"github.com/skycoin/dmsg"
	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/routing"
)

// ServerRPC is a RPC interface for the app server.
type ServerRPC struct {
	dmsgC *dmsg.Client
	lm    *manager
	cm    *manager
	log   *logging.Logger
}

// newServerRPC constructs new server RPC interface.
func newServerRPC(log *logging.Logger, dmsgC *dmsg.Client) *ServerRPC {
	return &ServerRPC{
		dmsgC: dmsgC,
		lm:    newManager(),
		cm:    newManager(),
		log:   log,
	}
}

// Dial dials to the remote.
func (r *ServerRPC) Dial(remote *routing.Addr, connID *uint16) error {
	connID, err := r.cm.nextID()
	if err != nil {
		return err
	}

	tp, err := r.dmsgC.Dial(context.TODO(), remote.PubKey, uint16(remote.Port))
	if err != nil {
		return err
	}

	if err := r.cm.set(*connID, tp); err != nil {
		return err
	}

	return nil
}

// Listen starts listening.
func (r *ServerRPC) Listen(local *routing.Addr, lisID *uint16) error {
	lisID, err := r.lm.nextID()
	if err != nil {
		return err
	}

	dmsgL, err := r.dmsgC.Listen(uint16(local.Port))
	if err != nil {
		return err
	}

	if err := r.lm.set(*lisID, dmsgL); err != nil {
		if err := dmsgL.Close(); err != nil {
			r.log.WithError(err).Error("error closing DMSG listener")
		}

		return err
	}

	return nil
}

// AcceptResp contains response parameters for `Accept`.
type AcceptResp struct {
	Remote routing.Addr
	ConnID uint16
}

// Accept accepts connection from the listener specified by `lisID`.
func (r *ServerRPC) Accept(lisID *uint16, resp *AcceptResp) error {
	lis, err := r.getListener(*lisID)
	if err != nil {
		return err
	}

	connID, err := r.cm.nextID()
	if err != nil {
		return err
	}

	tp, err := lis.Accept()
	if err != nil {
		return err
	}

	if err := r.cm.set(*connID, tp); err != nil {
		if err := tp.Close(); err != nil {
			r.log.WithError(err).Error("error closing DMSG transport")
		}

		return err
	}

	remote, ok := tp.RemoteAddr().(dmsg.Addr)
	if !ok {
		return errors.New("wrong type for transport remote addr")
	}

	resp = &AcceptResp{
		Remote: routing.Addr{
			PubKey: remote.PK,
			Port:   routing.Port(remote.Port),
		},
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
func (r *ServerRPC) Write(req *WriteReq, n *int) error {
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
func (r *ServerRPC) Read(connID *uint16, resp *ReadResp) error {
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
func (r *ServerRPC) CloseConn(connID *uint16, _ *struct{}) error {
	conn, err := r.getAndRemoveConn(*connID)
	if err != nil {
		return err
	}

	return conn.Close()
}

// CloseListener closes listener specified by `lisID`.
func (r *ServerRPC) CloseListener(lisID *uint16, _ *struct{}) error {
	lis, err := r.getAndRemoveListener(*lisID)
	if err != nil {
		return err
	}

	return lis.Close()
}

// getAndRemoveListener gets listener from the manager by `lisID` and removes it.
// Handles type assertion.
func (r *ServerRPC) getAndRemoveListener(lisID uint16) (*dmsg.Listener, error) {
	lisIfc, err := r.lm.getAndRemove(lisID)
	if err != nil {
		return nil, err
	}

	return r.assertListener(lisIfc)
}

// getAndRemoveConn gets conn from the manager by `connID` and removes it.
// Handles type assertion.
func (r *ServerRPC) getAndRemoveConn(connID uint16) (net.Conn, error) {
	connIfc, err := r.cm.getAndRemove(connID)
	if err != nil {
		return nil, err
	}

	return r.assertConn(connIfc)
}

// getListener gets listener from the manager by `lisID`. Handles type assertion.
func (r *ServerRPC) getListener(lisID uint16) (*dmsg.Listener, error) {
	lisIfc, ok := r.lm.get(lisID)
	if !ok {
		return nil, fmt.Errorf("no listener with key %d", lisID)
	}

	return r.assertListener(lisIfc)
}

// getConn gets conn from the manager by `connID`. Handles type assertion.
func (r *ServerRPC) getConn(connID uint16) (net.Conn, error) {
	connIfc, ok := r.cm.get(connID)
	if !ok {
		return nil, fmt.Errorf("no conn with key %d", connID)
	}

	return r.assertConn(connIfc)
}

// assertListener asserts that `v` is of type `*dmsg.Listener`.
func (r *ServerRPC) assertListener(v interface{}) (*dmsg.Listener, error) {
	lis, ok := v.(*dmsg.Listener)
	if !ok {
		return nil, errors.New("wrong type of value stored for listener")
	}

	return lis, nil
}

// assertConn asserts that `v` is of type `net.Conn`.
func (r *ServerRPC) assertConn(v interface{}) (net.Conn, error) {
	conn, ok := v.(net.Conn)
	if !ok {
		return nil, errors.New("wrong type of value stored for conn")
	}

	return conn, nil
}
