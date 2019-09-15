package app2

import (
	"context"
	"fmt"

	"github.com/skycoin/dmsg"

	"github.com/skycoin/skywire/pkg/routing"
)

type ServerRPC struct {
	dmsgC *dmsg.Client
	lm    *listenersManager
	cm    *connsManager
}

func newServerRPC(dmsgC *dmsg.Client) *ServerRPC {
	return &ServerRPC{
		dmsgC: dmsgC,
		lm:    newListenersManager(),
		cm:    newConnsManager(),
	}
}

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
		// TODO: close listener
		return err
	}

	return nil
}

func (r *ServerRPC) Accept(lisID *uint16, connID *uint16) error {
	lis, ok := r.lm.get(*lisID)
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
		// TODO: close conn
		return err
	}

	return nil
}

type WriteReq struct {
	ConnID uint16
	B      []byte
}

func (r *ServerRPC) Write(req *WriteReq, n *int) error {
	conn, ok := r.cm.get(req.ConnID)
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

type ReadResp struct {
	B []byte
	N int
}

func (r *ServerRPC) Read(connID *uint16, resp *ReadResp) error {
	conn, ok := r.cm.get(*connID)
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

func (r *ServerRPC) CloseConn(connID *uint16, _ *struct{}) error {
	conn, err := r.cm.getAndRemove(*connID)
	if err != nil {
		return err
	}

	return conn.Close()
}

func (r *ServerRPC) CloseListener(lisID *uint16, _ *struct{}) error {
	lis, err := r.lm.getAndRemove(*lisID)
	if err != nil {
		return err
	}

	return lis.Close()
}
