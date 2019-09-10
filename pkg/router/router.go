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

	"github.com/skycoin/skywire/pkg/snet"

	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/app"
	routeFinder "github.com/skycoin/skywire/pkg/route-finder/client"
	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/setup"
	"github.com/skycoin/skywire/pkg/transport"
)

const (
	// DefaultRouteKeepAlive is the default expiration interval for routes
	DefaultRouteKeepAlive = 2 * time.Hour

	// DefaultGarbageCollectDuration is the default duration for garbage collection of routing rules.
	DefaultGarbageCollectDuration = time.Second * 5

	minHops = 0
	maxHops = 50
)

var log = logging.MustGetLogger("router")

// Config configures Router.
type Config struct {
	Logger                 *logging.Logger
	PubKey                 cipher.PubKey
	SecKey                 cipher.SecKey
	TransportManager       *transport.Manager
	RoutingTable           routing.Table
	RouteFinder            routeFinder.Client
	SetupNodes             []cipher.PubKey
	GarbageCollectDuration time.Duration
}

// SetDefaults sets default values for certain empty values.
func (c *Config) SetDefaults() {
	if c.Logger == nil {
		c.Logger = log
	}
	if c.GarbageCollectDuration <= 0 {
		c.GarbageCollectDuration = DefaultGarbageCollectDuration
	}
}

// Router implements node.PacketRouter. It manages routing table by
// communicating with setup nodes, forward packets according to local
// rules and manages loops for apps.
type Router struct {
	Logger *logging.Logger

	conf        *Config
	staticPorts map[routing.Port]struct{}

	n  *snet.Network
	tm *transport.Manager
	pm *portManager
	rm *routeManager

	wg sync.WaitGroup
	mx sync.Mutex
}

// New constructs a new Router.
func New(n *snet.Network, config *Config) (*Router, error) {
	config.SetDefaults()

	r := &Router{
		Logger:      config.Logger,
		n:           n,
		tm:          config.TransportManager,
		pm:          newPortManager(10),
		conf:        config,
		staticPorts: make(map[routing.Port]struct{}),
	}

	// Prepare route manager.
	rm, err := newRouteManager(n, config.RoutingTable, RMConfig{
		SetupPKs:               config.SetupNodes,
		GarbageCollectDuration: config.GarbageCollectDuration,
		OnConfirmLoop:          r.confirmLoop,
		OnLoopClosed:           r.loopClosed,
	})
	if err != nil {
		return nil, err
	}
	r.rm = rm

	return r, nil
}

// Serve starts transport listening loop.
func (r *Router) Serve(ctx context.Context) error {
	r.Logger.Info("Starting router")

	go func() {
		for {
			packet, err := r.tm.ReadPacket()
			if err != nil {
				return
			}
			if err := r.handlePacket(ctx, packet); err != nil {
				if err == transport.ErrNotServing {
					r.Logger.WithError(err).Warnf("Stopped serving Transport.")
					return
				}
				r.Logger.Warnf("Failed to handle transport frame: %v", err)
			}
		}
	}()

	r.wg.Add(1)
	go func() {
		r.rm.Serve()
		r.wg.Done()
	}()

	r.tm.Serve(ctx)
	return nil
}

func (r *Router) handlePacket(ctx context.Context, packet routing.Packet) error {
	rule, err := r.rm.GetRule(packet.RouteID())
	if err != nil {
		return err
	}
	r.Logger.Infof("Got new remote packet with route ID %d. Using rule: %s", packet.RouteID(), rule)
	if rule.Type() == routing.RuleForward {
		return r.forwardPacket(ctx, packet.Payload(), rule)
	}
	return r.consumePacket(packet.Payload(), rule)
}

// ServeApp handles App packets from the App connection on provided port.
func (r *Router) ServeApp(conn net.Conn, port routing.Port, appConf *app.Config) error {
	fmt.Println("!!! [ServeApp] start !!!")

	r.wg.Add(1)
	defer r.wg.Done()

	appProto := app.NewProtocol(conn)
	if err := r.pm.Open(port, appProto); err != nil {
		return err
	}

	r.mx.Lock()
	r.staticPorts[port] = struct{}{}
	r.mx.Unlock()

	callbacks := &appCallbacks{
		CreateLoop: r.requestLoop,
		CloseLoop:  r.closeLoop,
		Forward:    r.forwardAppPacket,
	}
	am := &appManager{r.Logger, appProto, appConf, callbacks}
	err := am.Serve()

	for _, port := range r.pm.AppPorts(appProto) {
		for _, addr := range r.pm.Close(port) {
			if err := r.closeLoop(context.TODO(), appProto, routing.Loop{Local: routing.Addr{Port: port}, Remote: addr}); err != nil {
				log.WithError(err).Warn("Failed to close loop")
			}
		}
	}

	r.mx.Lock()
	delete(r.staticPorts, port)
	r.mx.Unlock()

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

	for _, conn := range r.pm.AppConns() {
		if err := conn.Close(); err != nil {
			r.Logger.WithError(err).Warn("Failed to close connection")
		}
	}

	if err := r.rm.Close(); err != nil {
		r.Logger.WithError(err).Warnf("closing route_manager returned error")
	}
	r.wg.Wait()

	return r.tm.Close()
}

func (r *Router) forwardPacket(ctx context.Context, payload []byte, rule routing.Rule) error {
	tp := r.tm.Transport(rule.TransportID())
	if tp == nil {
		return errors.New("unknown transport")
	}
	if err := tp.WritePacket(ctx, rule.RouteID(), payload); err != nil {
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
	fmt.Println("got it!")
	if err := b.conn.Send(app.FrameSend, p, nil); err != nil { // TODO: Stuck here.
		fmt.Println("!!! Send err:", err)
		return err
	}
	fmt.Println("done")

	r.Logger.Infof("Forwarded packet to App on Port %d", rule.LocalPort())
	return nil
}

func (r *Router) forwardAppPacket(ctx context.Context, appConn *app.Protocol, packet *app.Packet) error {
	if packet.Loop.Remote.PubKey == r.conf.PubKey {
		return r.forwardLocalAppPacket(packet)
	}

	l, err := r.pm.GetLoop(packet.Loop.Local.Port, packet.Loop.Remote)
	if err != nil {
		return err
	}

	tr := r.tm.Transport(l.trID)
	if tr == nil {
		return errors.New("unknown transport")
	}

	r.Logger.Infof("Forwarded App packet from LocalPort %d using route ID %d", packet.Loop.Local.Port, l.routeID)
	return tr.WritePacket(ctx, l.routeID, packet.Payload)
}

func (r *Router) forwardLocalAppPacket(packet *app.Packet) error {
	b, err := r.pm.Get(packet.Loop.Remote.Port)
	if err != nil {
		return nil
	}

	p := &app.Packet{
		Loop: routing.Loop{
			Local:  routing.Addr{Port: packet.Loop.Remote.Port},
			Remote: routing.Addr{PubKey: packet.Loop.Remote.PubKey, Port: packet.Loop.Local.Port},
		},
		Payload: packet.Payload,
	}
	return b.conn.Send(app.FrameSend, p, nil)
}

func (r *Router) requestLoop(ctx context.Context, appConn *app.Protocol, raddr routing.Addr) (routing.Addr, error) {
	lport := r.pm.Alloc(appConn)
	if err := r.pm.SetLoop(lport, raddr, &loop{}); err != nil {
		return routing.Addr{}, err
	}

	laddr := routing.Addr{PubKey: r.conf.PubKey, Port: lport}
	if raddr.PubKey == r.conf.PubKey {
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
		KeepAlive: DefaultRouteKeepAlive,
		Forward:   forwardRoute,
		Reverse:   reverseRoute,
	}

	sConn, err := r.rm.dialSetupConn(ctx)
	if err != nil {
		return routing.Addr{}, err
	}
	defer func() {
		if err := sConn.Close(); err != nil {
			r.Logger.Warnf("Failed to close transport: %s", err)
		}
	}()
	if err := setup.CreateLoop(ctx, setup.NewSetupProtocol(sConn), ld); err != nil {
		return routing.Addr{}, fmt.Errorf("route setup: %s", err)
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

	addrs := [2]routing.Addr{{PubKey: r.conf.PubKey, Port: l.Local.Port}, l.Remote}
	if err = b.conn.Send(app.FrameConfirmLoop, addrs, nil); err != nil {
		r.Logger.Warnf("Failed to notify App about new loop: %s", err)
	}

	return nil
}

func (r *Router) closeLoop(ctx context.Context, appConn *app.Protocol, loop routing.Loop) error {
	if err := r.destroyLoop(loop); err != nil {
		r.Logger.Warnf("Failed to remove loop: %s", err)
	}

	sConn, err := r.rm.dialSetupConn(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err := sConn.Close(); err != nil {
			r.Logger.Warnf("Failed to close transport: %s", err)
		}
	}()
	if err := setup.CloseLoop(ctx, setup.NewSetupProtocol(sConn), routing.LoopData{Loop: loop}); err != nil {
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
	r.mx.Lock()
	_, ok := r.staticPorts[loop.Local.Port]
	r.mx.Unlock()

	if ok {
		if err := r.pm.RemoveLoop(loop.Local.Port, loop.Remote); err != nil {
			log.WithError(err).Warn("Failed to remove loop")
		}
	} else {
		r.pm.Close(loop.Local.Port)
	}

	return r.rm.RemoveLoopRule(loop)
}

func (r *Router) fetchBestRoutes(source, destination cipher.PubKey) (fwd routing.Route, rev routing.Route, err error) {
	r.Logger.Infof("Requesting new routes from %s to %s", source, destination)

	timer := time.NewTimer(time.Second * 10)
	defer timer.Stop()

fetchRoutesAgain:
	ctx := context.Background()
	routes, err := r.conf.RouteFinder.FindRoutes(ctx, [][2]cipher.PubKey{{source, destination}, {destination, source}},
		&routeFinder.RouteOptions{MinHops: minHops, MaxHops: maxHops})
	if err != nil {
		select {
		case <-timer.C:
			return nil, nil, err
		default:
			goto fetchRoutesAgain
		}
	}

	r.Logger.Infof("Found routes Forward: %s. Reverse %s", routes[0], routes[1])
	return routes[0][0], routes[1][0], nil
}

// SetupIsTrusted checks if setup node is trusted.
func (r *Router) SetupIsTrusted(sPK cipher.PubKey) bool {
	return r.rm.conf.SetupIsTrusted(sPK)
}
