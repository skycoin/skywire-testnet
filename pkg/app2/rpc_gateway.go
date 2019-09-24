package app2

import (
	"fmt"
	"net"

	"github.com/pkg/errors"
	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/app2/network"
	"github.com/skycoin/skywire/pkg/routing"
)

// RPCGateway is a RPC interface for the app server.
type RPCGateway struct {
	lm  *idManager // contains listeners associated with their IDs
	cm  *idManager // contains connections associated with their IDs
	log *logging.Logger
}

// newRPCGateway constructs new server RPC interface.
func newRPCGateway(log *logging.Logger) *RPCGateway {
	return &RPCGateway{
		lm:  newIDManager(),
		cm:  newIDManager(),
		log: log,
	}
}

// DialResp contains response parameters for `Dial`.
type DialResp struct {
	ConnID       uint16
	AssignedPort routing.Port
}

// Dial dials to the remote.
func (r *RPCGateway) Dial(remote *network.Addr, resp *DialResp) error {
	reservedConnID, free, err := r.cm.reserveNextID()
	if err != nil {
		return err
	}

	conn, err := network.Dial(*remote)
	if err != nil {
		free()
		return err
	}

	localAddr, err := network.WrapAddr(conn.LocalAddr())
	if err != nil {
		free()
		return err
	}

	if err := r.cm.set(*reservedConnID, conn); err != nil {
		if err := conn.Close(); err != nil {
			r.log.WithError(err).Error("error closing conn")
		}

		free()
		return err
	}

	resp.ConnID = *reservedConnID
	resp.AssignedPort = localAddr.Port

	return nil
}

// Listen starts listening.
func (r *RPCGateway) Listen(local *network.Addr, lisID *uint16) error {
	nextLisID, free, err := r.lm.reserveNextID()
	if err != nil {
		return err
	}

	l, err := network.Listen(*local)
	if err != nil {
		free()
		return err
	}

	if err := r.lm.set(*nextLisID, l); err != nil {
		if err := l.Close(); err != nil {
			r.log.WithError(err).Error("error closing listener")
		}

		free()
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

	connID, free, err := r.cm.reserveNextID()
	if err != nil {
		return err
	}

	conn, err := lis.Accept()
	if err != nil {
		free()
		return err
	}

	if err := r.cm.set(*connID, conn); err != nil {
		if err := conn.Close(); err != nil {
			r.log.WithError(err).Error("error closing DMSG transport")
		}

		free()
		return err
	}

	remote, ok := conn.RemoteAddr().(network.Addr)
	if !ok {
		free()
		return errors.New("wrong type for remote addr")
	}

	resp.Remote = remote
	resp.ConnID = *connID

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

// ReadReq contains arguments for `Read`.
type ReadReq struct {
	ConnID uint16
	BufLen int
}

// ReadResp contains response parameters for `Read`.
type ReadResp struct {
	B []byte
	N int
}

// Read reads data from connection specified by `connID`.
func (r *RPCGateway) Read(req *ReadReq, resp *ReadResp) error {
	conn, err := r.getConn(req.ConnID)
	if err != nil {
		return err
	}

	buf := make([]byte, req.BufLen)
	resp.N, err = conn.Read(buf)
	if err != nil {
		return err
	}

	resp.B = make([]byte, resp.N)
	copy(resp.B, buf[:resp.N])

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
		return nil, errors.Wrap(err, "no listener")
	}

	return assertListener(lisIfc)
}

// popConn gets conn from the manager by `connID` and removes it.
// Handles type assertion.
func (r *RPCGateway) popConn(connID uint16) (net.Conn, error) {
	connIfc, err := r.cm.pop(connID)
	if err != nil {
		return nil, errors.Wrap(err, "no conn")
	}

	return assertConn(connIfc)
}

// getListener gets listener from the manager by `lisID`. Handles type assertion.
func (r *RPCGateway) getListener(lisID uint16) (net.Listener, error) {
	lisIfc, ok := r.lm.get(lisID)
	if !ok {
		return nil, fmt.Errorf("no listener with key %d", lisID)
	}

	return assertListener(lisIfc)
}

// getConn gets conn from the manager by `connID`. Handles type assertion.
func (r *RPCGateway) getConn(connID uint16) (net.Conn, error) {
	connIfc, ok := r.cm.get(connID)
	if !ok {
		return nil, fmt.Errorf("no conn with key %d", connID)
	}

	return assertConn(connIfc)
}
