package router

import (
	"errors"
	"fmt"

	"github.com/skycoin/skywire/pkg/app"
	"github.com/skycoin/skywire/pkg/routing"
)

type portManager struct {
	ports *portList
}

func newPortManager(minPort uint16) *portManager {
	return &portManager{newPortList(minPort)}
}

func (pm *portManager) Alloc(conn *app.Protocol) uint16 {
	b := &portBind{conn, newLoopList()}
	return pm.ports.add(b)
}

func (pm *portManager) Open(port uint16, proto *app.Protocol) error {
	if pm.ports.get(port) != nil {
		return fmt.Errorf("port %d is already bound", port)
	}

	pm.ports.set(port, &portBind{proto, newLoopList()})
	return nil
}

func (pm *portManager) SetLoop(port uint16, raddr *routing.Addr, l *loop) error {
	b := pm.ports.get(port)
	if b == nil {
		return errors.New("port is not bound")
	}

	b.loops.set(raddr, l)
	return nil
}

func (pm *portManager) AppConns() []*app.Protocol {
	res := []*app.Protocol{}
	set := map[*app.Protocol]struct{}{}
	for _, bind := range pm.ports.all() {
		if _, ok := set[bind.conn]; !ok {
			res = append(res, bind.conn)
			set[bind.conn] = struct{}{}
		}
	}
	return res
}

func (pm *portManager) AppPorts(appConn *app.Protocol) []uint16 {
	res := []uint16{}
	for port, bind := range pm.ports.all() {
		if bind.conn == appConn {
			res = append(res, port)
		}
	}
	return res
}

func (pm *portManager) Close(port uint16) []routing.Addr {
	if pm == nil {
		return nil
	}

	b := pm.ports.remove(port)
	if b == nil {
		return nil
	}

	return b.loops.dropAll()
}

func (pm *portManager) RemoveLoop(port uint16, raddr *routing.Addr) error {
	b, err := pm.Get(port)
	if err != nil {
		return err
	}

	b.loops.remove(raddr)
	return nil
}

func (pm *portManager) Get(port uint16) (*portBind, error) {
	b := pm.ports.get(port)
	if b == nil {
		return nil, errors.New("port is not bound")
	}

	return b, nil
}

func (pm *portManager) GetLoop(localPort uint16, remoteAddr *routing.Addr) (*loop, error) {
	b, err := pm.Get(localPort)
	if err != nil {
		return nil, err
	}

	l := b.loops.get(remoteAddr)
	if l == nil {
		return nil, errors.New("unknown loop")
	}

	return l, nil
}
