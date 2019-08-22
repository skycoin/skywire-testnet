package router

import (
	"errors"
	"fmt"

	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/app"
	"github.com/skycoin/skywire/pkg/routing"

	th "github.com/skycoin/skywire/internal/testhelpers"
)

// var (
// 	logger = logging.MustGetLogger("router")
// 	debug  = logger.Debug
// )

type portManager struct {
	ports  *portList
	logger *logging.Logger
}

func (pm *portManager) debug(args ...interface{}) {
	pm.logger.Debug(args)
}

func newPortManager(minPort routing.Port, logger *logging.Logger) *portManager {
	return &portManager{newPortList(minPort), logger}
}

func (pm *portManager) Alloc(conn *app.Protocol) routing.Port {
	pm.debug(th.Trace("ENTER"))
	defer pm.debug(th.Trace("EXIT"))

	b := &portBind{conn, newLoopList()}
	return pm.ports.add(b)
}

func (pm *portManager) Open(port routing.Port, proto *app.Protocol) error {
	pm.debug(th.Trace("ENTER"))
	defer pm.debug(th.Trace("EXIT"))

	if pm.ports.get(port) != nil {
		return fmt.Errorf("port %d is already bound", port)
	}

	pm.ports.set(port, &portBind{proto, newLoopList()})
	return nil
}

func (pm *portManager) SetLoop(port routing.Port, raddr routing.Addr, l *loop) error {
	pm.debug(th.Trace("ENTER"))
	defer pm.debug(th.Trace("EXIT"))

	b := pm.ports.get(port)
	if b == nil {
		return errors.New("port is not bound")
	}

	b.loops.set(raddr, l)
	return nil
}

func (pm *portManager) AppConns() []*app.Protocol {
	pm.debug(th.Trace("ENTER"))
	defer pm.debug(th.Trace("EXIT"))

	res := make([]*app.Protocol, 0)
	set := map[*app.Protocol]struct{}{}
	for _, bind := range pm.ports.all() {
		if _, ok := set[bind.conn]; !ok {
			res = append(res, bind.conn)
			set[bind.conn] = struct{}{}
		}
	}
	return res
}

func (pm *portManager) AppPorts(appConn *app.Protocol) []routing.Port {
	pm.debug(th.Trace("ENTER"))
	defer pm.debug(th.Trace("EXIT"))

	res := make([]routing.Port, 0)
	for port, bind := range pm.ports.all() {
		if bind.conn == appConn {
			res = append(res, port)
		}
	}
	return res
}

func (pm *portManager) Close(port routing.Port) []routing.Addr {
	pm.debug(th.Trace("ENTER"))
	defer pm.debug(th.Trace("EXIT"))

	if pm == nil {
		return nil
	}

	b := pm.ports.remove(port)
	if b == nil {
		return nil
	}

	return b.loops.dropAll()
}

func (pm *portManager) RemoveLoop(port routing.Port, raddr routing.Addr) error {
	pm.debug(th.Trace("ENTER"))
	defer pm.debug(th.Trace("EXIT"))

	b, err := pm.Get(port)
	if err != nil {
		return err
	}

	b.loops.remove(raddr)
	return nil
}

func (pm *portManager) Get(port routing.Port) (*portBind, error) {
	pm.debug(th.Trace("ENTER"))
	defer pm.debug(th.Trace("EXIT"))

	b := pm.ports.get(port)
	if b == nil {
		return nil, errors.New("port is not bound")
	}

	return b, nil
}

func (pm *portManager) GetLoop(localPort routing.Port, remoteAddr routing.Addr) (*loop, error) {
	pm.debug(th.Trace("ENTER"))
	defer pm.debug(th.Trace("EXIT"))

	b, err := pm.Get(localPort)
	if err != nil {
		fmt.Println("pm.Get err:", err)
		return nil, err
	}

	l := b.loops.get(remoteAddr)
	if l == nil {
		fmt.Println("b.loops.get err:", err)
		return nil, errors.New("unknown loop")
	}

	return l, nil
}

// Portlist - debug only
type Portlist = portList

func (pm *portManager) Ports() *portList {
	return pm.ports
}
