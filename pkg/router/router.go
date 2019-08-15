// Package router implements package router for skywire visor.
package router

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/app"
	routeFinder "github.com/skycoin/skywire/pkg/route-finder/client"
	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/setup"
	"github.com/skycoin/skywire/pkg/transport"
)

const (
	// RouteTTL is the default expiration interval for routes
	RouteTTL = 2 * time.Hour
	minHops  = 0
	maxHops  = 50
)

var log = logging.MustGetLogger("router")

// Config configures Router.
type Config struct {
	Logger           *logging.Logger
	PubKey           cipher.PubKey
	SecKey           cipher.SecKey
	TransportManager *transport.Manager
	RoutingTable     routing.Table
	RouteFinder      routeFinder.Client
	SetupNodes       []cipher.PubKey
	TransportType    string
}

// Router implements node.PacketRouter. It manages routing table by
// communicating with setup nodes, forward packets according to local
// rules and manages loops for apps.
type Router struct {
	Logger *logging.Logger

	config *Config
	tm     *transport.Manager
	pm     *portManager
	rm     *routeManager

	expiryTicker *time.Ticker
	wg           sync.WaitGroup

	staticPorts map[routing.Port]struct{}
	mu          sync.Mutex
}

// New constructs a new Router.
func New(config *Config) *Router {
	r := &Router{
		Logger:       config.Logger,
		tm:           config.TransportManager,
		pm:           newPortManager(10),
		config:       config,
		expiryTicker: time.NewTicker(10 * time.Minute),
		staticPorts:  make(map[routing.Port]struct{}),
	}
	callbacks := &setupCallbacks{
		ConfirmLoop: r.confirmLoop,
		LoopClosed:  r.loopClosed,
	}
	r.rm = &routeManager{r.Logger, manageRoutingTable(config.RoutingTable), callbacks}
	return r
}

// Serve starts transport listening loop.
func (r *Router) Serve(ctx context.Context) error {
	r.Logger.Info("Starting router")

	go func() {
		for {
			select {
			case dTp, ok := <-r.tm.DataTpChan:
				if !ok {
					return
				}
				initStatus := "locally"
				if dTp.Accepted {
					initStatus = "remotely"
				}
				r.Logger.Infof("New %s-initiated transport: purpose(data)", initStatus)
				r.handleTransport(dTp, dTp.Accepted, false)

			case sTp, ok := <-r.tm.SetupTpChan:
				if !ok {
					return
				}
				r.Logger.Infof("New remotely-initiated transport: purpose(setup)")
				r.handleTransport(sTp, true, true)

			case <-r.expiryTicker.C:
				if err := r.rm.rt.Cleanup(); err != nil {
					r.Logger.Warnf("Failed to expiry routes: %s", err)
				}
			}
		}
	}()
	return r.tm.Serve(ctx)
}

func (r *Router) handleTransport(tp transport.Transport, isAccepted, isSetup bool) {
	var serve func(io.ReadWriter) error
	switch {
	case isAccepted && isSetup:
		serve = r.rm.Serve
	case !isSetup:
		serve = r.serveTransport
	default:
		return
	}

	go func(tp transport.Transport) {
		defer func() {
			if err := tp.Close(); err != nil {
				r.Logger.Warnf("Failed to close transport: %s", err)
			}
		}()
		for {
			if err := serve(tp); err != nil {
				if err != io.EOF {
					r.Logger.Warnf("Stopped serving Transport: %s", err)
				}
				return
			}
		}
	}(tp)
}

// ServeApp handles App packets from the App connection on provided port.
func (r *Router) ServeApp(conn net.Conn, port routing.Port, appConf *app.Config) error {
	r.wg.Add(1)
	defer r.wg.Done()

	appProto := app.NewProtocol(conn)
	if err := r.pm.Open(port, appProto); err != nil {
		return err
	}

	r.mu.Lock()
	r.staticPorts[port] = struct{}{}
	r.mu.Unlock()

	callbacks := &appCallbacks{
		CreateLoop:       r.requestLoop,
		CloseLoop:        r.closeLoop,
		ForwardAppPacket: r.forwardAppPacket,
	}
	am := &appManager{r.Logger, appProto, appConf, callbacks}
	err := am.Serve()

	for _, port := range r.pm.AppPorts(appProto) {
		for _, addr := range r.pm.Close(port) {
			if err := r.closeLoop(appProto, routing.Loop{Local: routing.Addr{Port: port}, Remote: addr}); err != nil {
				log.WithError(err).Warn("Failed to close loop")
			}
		}
	}

	r.mu.Lock()
	delete(r.staticPorts, port)
	r.mu.Unlock()

	if err == io.EOF {
		return nil
	}
	return err
}

// Close safely stops Router.
func (r *Router) Close() error {
	if r == nil {
		return nil
	}

	r.Logger.Info("Closing all App connections and Loops")
	r.expiryTicker.Stop()

	for _, conn := range r.pm.AppConns() {
		if err := conn.Close(); err != nil {
			log.WithError(err).Warn("Failed to close connection")
		}
	}

	r.wg.Wait()
	return r.tm.Close()
}

func (r *Router) serveTransport(rw io.ReadWriter) error {
	packet := make(routing.Packet, 6)
	if _, err := io.ReadFull(rw, packet); err != nil {
		return err
	}

	payload := make([]byte, packet.Size())
	if _, err := io.ReadFull(rw, payload); err != nil {
		return err
	}

	rule, err := r.rm.GetRule(packet.RouteID())
	if err != nil {
		return err
	}

	r.Logger.Infof("Got new remote packet with route ID %d. Using rule: %s", packet.RouteID(), rule)
	if rule.Type() == routing.RuleForward {
		return r.forwardPacket(payload, rule)
	}

	return r.consumePacket(payload, rule)
}

func (r *Router) forwardPacket(payload []byte, rule routing.Rule) error {
	packet := routing.MakePacket(rule.RouteID(), payload)
	tr := r.tm.Transport(rule.TransportID())
	if tr == nil {
		return errors.New("unknown transport")
	}

	if _, err := tr.Write(packet); err != nil {
		return err
	}

	r.Logger.Infof("Forwarded packet via Transport %s using rule %d", rule.TransportID(), rule.RouteID())
	return nil
}

func (r *Router) consumePacket(payload []byte, rule routing.Rule) error {
	laddr := routing.Addr{Port: rule.LocalPort()}
	raddr := routing.Addr{PubKey: rule.RemotePK(), Port: rule.RemotePort()}

	p := &app.Packet{Loop: routing.Loop{Local: laddr, Remote: raddr}, Payload: payload}
	b, err := r.pm.Get(rule.LocalPort())
	if err != nil {
		return err
	}
	if err := b.conn.Send(app.FrameSend, p, nil); err != nil {
		return err
	}

	r.Logger.Infof("Forwarded packet to App on Port %d", rule.LocalPort())
	return nil
}

func (r *Router) forwardAppPacket(appConn *app.Protocol, packet *app.Packet) error {

	r.Logger.WithField("packet.Loop", packet.Loop).Info("Entering r.forwardAppPacket")

	if packet.Loop.Remote.PubKey == r.config.PubKey {
		return r.forwardLocalAppPacket(packet)
	}
	r.Logger.Info("Entering r.forwardAppPacket GetLoop")
	l, err := r.pm.GetLoop(packet.Loop.Local.Port, packet.Loop.Remote)
	if err != nil {
		return err
	}

	r.Logger.WithField("trID", l.trID).Infof("Entering r.forwardAppPacket r.tm.Transport(l.trID)")
	tr := r.tm.Transport(l.trID)
	if tr == nil {
		return fmt.Errorf("unknown transport id %v", l.trID)
	}

	r.Logger.Info("r.forwardAppPacket enter routing.MakePacket")

	p := routing.MakePacket(l.routeID, packet.Payload)
	r.Logger.Infof("Forwarded App packet from LocalPort %d using route ID %d", packet.Loop.Local.Port, l.routeID)
	_, err = tr.Write(p)
	return err
}

func (r *Router) forwardLocalAppPacket(packet *app.Packet) error {
	r.Logger.Info("entering r.forwardLocalAppPacket ")
	b, err := r.pm.Get(packet.Loop.Remote.Port)
	if err != nil {
		return nil
	}
	// r.Logger.WithField("Local", ).Info("entering r.forwardLocalAppPacket app.Packet")
	p := &app.Packet{
		Loop: routing.Loop{
			Local:  routing.Addr{PubKey: packet.Loop.Remote.PubKey, Port: packet.Loop.Remote.Port},
			Remote: routing.Addr{PubKey: packet.Loop.Remote.PubKey, Port: packet.Loop.Local.Port},
		},
		Payload: packet.Payload,
	}

	r.Logger.WithField("packet", p).Info("entering r.forwardLocalAppPacket Send")
	return b.conn.Send(app.FrameSend, p, nil)
}

func (r *Router) requestLoop(appConn *app.Protocol, raddr routing.Addr) (routing.Addr, error) {

	lport := r.pm.Alloc(appConn)
	if err := r.pm.SetLoop(lport, raddr, &loop{}); err != nil {
		return routing.Addr{}, err
	}

	laddr := routing.Addr{PubKey: r.config.PubKey, Port: lport}
	if raddr.PubKey == r.config.PubKey {
		if err := r.confirmLocalLoop(laddr, raddr); err != nil {
			return routing.Addr{}, fmt.Errorf("confirm: %s", err)
		}
		r.Logger.Infof("Created local loop on port %d", laddr.Port)
		return laddr, nil
	}

	forwardRoute, reverseRoute, err := r.fetchBestRoutes(laddr.PubKey, raddr.PubKey)
	if err != nil {
		return routing.Addr{}, fmt.Errorf("route finder: %s", err)
	}

	ld := routing.LoopDescriptor{
		Loop: routing.Loop{
			Local:  laddr,
			Remote: raddr,
		},
		Expiry:  time.Now().Add(RouteTTL),
		Forward: forwardRoute,
		Reverse: reverseRoute,
	}

	r.Logger.Infof("Router.requestLoop\n")
	r.Logger.Infof("laddr: %v\n, raddr: %v\n", laddr, raddr)

	r.Logger.Info("Attempt to r.setupProto from r.requestLoop")
	switch r.config.TransportType {
	case "dmsg":
		proto, tr, err := r.setupProto(context.Background())
		if err != nil {
			return routing.Addr{}, err
		}
		defer func() {
			if err := tr.Close(); err != nil {
				r.Logger.Warnf("Failed to close transport: %s", err)
			}
		}()

		r.Logger.Infof("Router.requestLoop 6\n")
		if err := setup.CreateLoop(proto, ld); err != nil {
			return routing.Addr{}, fmt.Errorf("route setup: %s", err)
		}
	case "tcp-transport":
		r.Logger.Info("Skipping setup for tcp-transport")
		_, err := r.tm.CreateSetupTransport(context.Background(), raddr.PubKey, "tcp-transport")
		if err != nil {
			r.Logger.Warnf("error creating transport %s", err)
		}

	}

	r.Logger.Infof("Created new loop to %s on port %d", raddr, laddr.Port)
	return laddr, nil
}

func (r *Router) confirmLocalLoop(laddr, raddr routing.Addr) error {
	b, err := r.pm.Get(raddr.Port)
	if err != nil {
		return err
	}

	addrs := [2]routing.Addr{raddr, laddr}
	if err = b.conn.Send(app.FrameConfirmLoop, addrs, nil); err != nil {
		return err
	}

	return nil
}

func (r *Router) confirmLoop(l routing.Loop, rule routing.Rule) error {
	b, err := r.pm.Get(l.Local.Port)
	if err != nil {
		return err
	}

	if err := r.pm.SetLoop(l.Local.Port, l.Remote, &loop{rule.TransportID(), rule.RouteID()}); err != nil {
		return err
	}

	addrs := [2]routing.Addr{{PubKey: r.config.PubKey, Port: l.Local.Port}, l.Remote}
	if err = b.conn.Send(app.FrameConfirmLoop, addrs, nil); err != nil {
		r.Logger.Warnf("Failed to notify App about new loop: %s", err)
	}

	return nil
}

func (r *Router) closeLoop(appConn *app.Protocol, loop routing.Loop) error {
	if err := r.destroyLoop(loop); err != nil {
		r.Logger.Warnf("Failed to remove loop: %s", err)
	}
	r.Logger.Info("Attempt to r.setupProto from r.closeLoop")
	proto, tr, err := r.setupProto(context.Background())
	if err != nil {
		return err
	}
	defer func() {
		if err := tr.Close(); err != nil {
			r.Logger.Warnf("Failed to close transport: %s", err)
		}
	}()

	ld := routing.LoopData{Loop: loop}
	if err := setup.CloseLoop(proto, ld); err != nil {
		return fmt.Errorf("route setup: %s", err)
	}

	r.Logger.Infof("Closed loop %s", loop)
	return nil
}

func (r *Router) loopClosed(loop routing.Loop) error {
	b, err := r.pm.Get(loop.Local.Port)
	if err != nil {
		return nil
	}

	if err := r.destroyLoop(loop); err != nil {
		r.Logger.Warnf("Failed to remove loop: %s", err)
	}

	if err := b.conn.Send(app.FrameClose, loop, nil); err != nil {
		return err
	}

	r.Logger.Infof("Closed loop %s", loop)
	return nil
}

func (r *Router) destroyLoop(loop routing.Loop) error {
	r.mu.Lock()
	_, ok := r.staticPorts[loop.Local.Port]
	r.mu.Unlock()

	if ok {
		if err := r.pm.RemoveLoop(loop.Local.Port, loop.Remote); err != nil {
			log.WithError(err).Warn("Failed to remove loop")
		}
	} else {
		r.pm.Close(loop.Local.Port)
	}

	return r.rm.RemoveLoopRule(loop)
}

func (r *Router) setupProto(ctx context.Context) (*setup.Protocol, transport.Transport, error) {
	if len(r.config.SetupNodes) == 0 {
		return nil, nil, errors.New("route setup: no nodes")
	}

	trType := r.config.TransportType
	// TODO(evanlinjin): need string constant for tp type.
	tr, err := r.tm.CreateSetupTransport(ctx, r.config.SetupNodes[0], trType)
	if err != nil {
		return nil, nil, fmt.Errorf("setup transport: %s", err)
	}

	sProto := setup.NewSetupProtocol(tr)
	return sProto, tr, nil
}

func (r *Router) fetchBestRoutes(source, destination cipher.PubKey) (fwd routing.Route, rev routing.Route, err error) {
	r.Logger.Infof("Requesting new routes from %s to %s", source, destination)

	timer := time.NewTimer(time.Second * 10)
	defer timer.Stop()

fetchRoutesAgain:
	fwdRoutes, revRoutes, err := r.config.RouteFinder.PairedRoutes(source, destination, minHops, maxHops)
	if err != nil {
		select {
		case <-timer.C:
			return nil, nil, err
		default:
			goto fetchRoutesAgain
		}
	}

	r.Logger.Infof("Found routes Forward: %s. Reverse %s", fwdRoutes, revRoutes)
	return fwdRoutes[0], revRoutes[0], nil
}

// IsSetupTransport checks whether `tr` is running in the `setup` mode.
func (r *Router) IsSetupTransport(tr *transport.ManagedTransport) bool {
	for _, pk := range r.config.SetupNodes {
		if tr.RemotePK() == pk {
			return true
		}
	}

	return false
}
