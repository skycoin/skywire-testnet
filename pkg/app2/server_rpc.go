package app2

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/skycoin/dmsg"

	"github.com/skycoin/skywire/pkg/routing"
)

type ServerRPC struct {
	dmsgC       *dmsg.Client
	conns       map[uint16]net.Conn
	connsMx     sync.RWMutex
	lstConnID   uint16
	listeners   map[uint16]*dmsg.Listener
	listenersMx sync.RWMutex
	lstLisID    uint16
}

func (r *ServerRPC) nextConnID() (*uint16, error) {
	r.connsMx.Lock()

	connID := r.lstConnID + 1
	for ; connID < r.lstConnID; connID++ {
		if _, ok := r.conns[connID]; !ok {
			break
		}
	}

	if connID == r.lstConnID {
		r.connsMx.Unlock()
		return nil, errors.New("no more available conns")
	}

	r.conns[connID] = nil
	r.lstConnID = connID

	r.connsMx.Unlock()
	return &connID, nil
}

func (r *ServerRPC) nextLisID() (*uint16, error) {
	r.listenersMx.Lock()

	lisID := r.lstLisID + 1
	for ; lisID < r.lstLisID; lisID++ {
		if _, ok := r.listeners[lisID]; !ok {
			break
		}
	}

	if lisID == r.lstLisID {
		r.listenersMx.Unlock()
		return nil, errors.New("no more available listeners")
	}

	r.listeners[lisID] = nil
	r.lstLisID = lisID

	r.listenersMx.Unlock()
	return &lisID, nil
}

func (r *ServerRPC) setConn(connID uint16, conn net.Conn) error {
	r.connsMx.Lock()

	if c, ok := r.conns[connID]; ok && c != nil {
		r.connsMx.Unlock()
		return errors.New("conn already exists")
	}

	r.conns[connID] = conn

	r.connsMx.Unlock()
	return nil
}

func (r *ServerRPC) setListener(lisID uint16, lis *dmsg.Listener) error {
	r.listenersMx.Lock()

	if l, ok := r.listeners[lisID]; ok && l != nil {
		r.listenersMx.Unlock()
		return errors.New("listener already exists")
	}

	r.listeners[lisID] = lis

	r.listenersMx.Unlock()
	return nil
}

func (r *ServerRPC) getConn(connID uint16) (net.Conn, bool) {
	r.connsMx.RLock()
	conn, ok := r.conns[connID]
	r.connsMx.RUnlock()
	return conn, ok
}

func (r *ServerRPC) getListener(lisID uint16) (*dmsg.Listener, bool) {
	r.listenersMx.RLock()
	lis, ok := r.listeners[lisID]
	r.listenersMx.RUnlock()
	return lis, ok
}

type DialReq struct {
	Remote routing.Addr
}

func (r *ServerRPC) Dial(req *DialReq, connID *uint16) error {
	connID, err := r.nextConnID()
	if err != nil {
		return err
	}

	tp, err := r.dmsgC.Dial(context.TODO(), req.Remote.PubKey, uint16(req.Remote.Port))
	if err != nil {
		return err
	}

	if err := r.setConn(*connID, tp); err != nil {
		return err
	}

	return nil
}

type ListenReq struct {
	Local routing.Addr
}

func (r *ServerRPC) Listen(req *ListenReq, lisID *uint16) error {
	lisID, err := r.nextLisID()
	if err != nil {
		return err
	}

	dmsgL, err := r.dmsgC.Listen(uint16(req.Local.Port))
	if err != nil {
		return err
	}

	if err := r.setListener(*lisID, dmsgL); err != nil {
		// TODO: close listener
		return err
	}

	return nil
}

func (r *ServerRPC) Accept(lisID *uint16, connID *uint16) error {
	lis, ok := r.getListener(*lisID)
	if !ok {
		return fmt.Errorf("not listener with id %d", *lisID)
	}

	connID, err := r.nextConnID()
	if err != nil {
		return err
	}

	tp, err := lis.Accept()
	if err != nil {
		return err
	}

	if err := r.setConn(*connID, tp); err != nil {
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
	conn, ok := r.getConn(req.ConnID)
	if !ok {
		return fmt.Errorf("not conn with id %d", req.ConnID)
	}

}
