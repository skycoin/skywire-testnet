package app2

import (
	"context"
	"fmt"
	"net"

	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/pkg/errors"
	"github.com/skycoin/dmsg"

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
	lis, ok := r.getListener(*lisID)
	if !ok {
		return fmt.Errorf("not listener with id %d", *lisID)
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
	conn, ok := r.getConn(req.ConnID)
	if !ok {
		return fmt.Errorf("no conn with id %d", req.ConnID)
	}

	var err error
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
	conn, ok := r.getConn(*connID)
	if !ok {
		return fmt.Errorf("no conn with id %d", *connID)
	}

	var err error
	resp.N, err = conn.Read(resp.B)
	if err != nil {
		return err
	}

	return nil
}

// CloseConn closes connection specified by `connID`.
func (r *ServerRPC) CloseConn(connID *uint16, _ *struct{}) error {
	conn, err := r.cm.getAndRemove(*connID)
	if err != nil {
		return err
	}

	return conn.Close()
}

// CloseListener closes listener specified by `lisID`.
func (r *ServerRPC) CloseListener(lisID *uint16, _ *struct{}) error {
	lis, err := r.lm.getAndRemove(*lisID)
	if err != nil {
		return err
	}

	return lis.Close()
}

func (r *ServerRPC) getAndRemoveListener(lisID uint16) (*dmsg.Listener, error) {
	lisIfc, err := r.lm.getAndRemove(lisID)
	if err != nil {
		return nil, err
	}

	return r.assertListener(lisIfc)
}

func (r *ServerRPC) getAndRemoveConn(connID uint16) (net.Conn, error) {
	connIfc, err := r.cm.getAndRemove(connID)
	if err != nil {
		return nil, err
	}

	return r.assertConn(connIfc)
}

func (r *ServerRPC) getListener(lisID uint16) (*dmsg.Listener, error) {
	lisIfc, ok := r.lm.get(lisID)
	if !ok {
		return nil, false
	}

	return r.assertListener(lisIfc)
}

func (r *ServerRPC) getConn(connID uint16) (net.Conn, error) {
	connIfc, ok := r.cm.get(connID)
	if !ok {
		return nil, false
	}

	conn, ok := connIfc.(net.Conn)
	if !ok {
		r.log.Errorln("wrong type of value stored for conn")
		return nil, false
	}

	return conn, true
}
func (r *ServerRPC) assertListener(v interface{}) (*dmsg.Listener, error) {
	lis, ok := v.(*dmsg.Listener)
	if !ok {
		return nil, errors.New("wrong type of value stored for listener")
	}

	return lis, nil
}

func (r *ServerRPC) assertConn(v interface{}) (net.Conn, error) {
	conn, ok := v.(net.Conn)
	if !ok {
		return nil, errors.New("wrong type of value stored for conn")
	}

	return conn, nil
}
