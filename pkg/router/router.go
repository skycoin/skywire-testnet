// Package router implements package router for skywire node.
package router

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/skycoin/skywire/pkg/dmsg"

	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/cipher"

	"github.com/skycoin/skywire/internal/noise"
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

// Config configures Router.
type Config struct {
	Logger           *logging.Logger
	PubKey           cipher.PubKey
	SecKey           cipher.SecKey
	TransportManager *transport.Manager
	RoutingTable     routing.Table
	RouteFinder      routeFinder.Client
	SetupNodes       []cipher.PubKey
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

	staticPorts map[uint16]struct{}
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
		staticPorts:  make(map[uint16]struct{}),
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

	go func() {
		for tp := range r.tm.TrChan {
			r.mu.Lock()
			isAccepted, isSetup := tp.Accepted, r.IsSetupTransport(tp)
			r.mu.Unlock()

			var serve func(io.ReadWriter) error
			switch {
			case isAccepted && isSetup:
				serve = r.rm.Serve
			case !isSetup:
				serve = r.serveTransport
			default:
				continue
			}

			go func(tp transport.Transport) {
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
	}()

	go func() {
		for range r.expiryTicker.C {
			if err := r.rm.rt.Cleanup(); err != nil {
				r.Logger.Warnf("Failed to expiry routes: %s", err)
			}
		}
	}()

	r.Logger.Info("Starting router")
	return r.tm.Serve(ctx)
}

// ServeApp handles App packets from the App connection on provided port.
func (r *Router) ServeApp(conn net.Conn, port uint16, appConf *app.Config) error {
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
		CreateLoop: r.requestLoop,
		CloseLoop:  r.closeLoop,
		Forward:    r.forwardAppPacket,
	}
	am := &appManager{r.Logger, appProto, appConf, callbacks}
	err := am.Serve()

	for _, port := range r.pm.AppPorts(appProto) {
		for _, addr := range r.pm.Close(port) {
			r.closeLoop(appProto, &app.LoopAddr{Port: port, Remote: addr}) // nolint: errcheck
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
	r.Logger.Info("Closing all App connections and Loops")
	r.expiryTicker.Stop()

	for _, conn := range r.pm.AppConns() {
		conn.Close()
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
	raddr := &app.Addr{PubKey: rule.RemotePK(), Port: rule.RemotePort()}
	l, err := r.pm.GetLoop(rule.LocalPort(), raddr)
	if err != nil {
		return errors.New("unknown loop")
	}

	data, err := l.noise.DecryptUnsafe(payload)
	if err != nil {
		return fmt.Errorf("noise: %s", err)
	}

	p := &app.Packet{Addr: &app.LoopAddr{Port: rule.LocalPort(), Remote: *raddr}, Payload: data}
	b, _ := r.pm.Get(rule.LocalPort()) // nolint: errcheck
	if err := b.conn.Send(app.FrameSend, p, nil); err != nil {
		return err
	}

	r.Logger.Infof("Forwarded packet to App on Port %d", rule.LocalPort())
	return nil
}

func (r *Router) forwardAppPacket(appConn *app.Protocol, packet *app.Packet) error {
	if packet.Addr.Remote.PubKey == r.config.PubKey {
		return r.forwardLocalAppPacket(packet)
	}

	l, err := r.pm.GetLoop(packet.Addr.Port, &packet.Addr.Remote)
	if err != nil {
		return err
	}

	tr := r.tm.Transport(l.trID)
	if tr == nil {
		return errors.New("unknown transport")
	}

	p := routing.MakePacket(l.routeID, l.noise.EncryptUnsafe(packet.Payload))
	r.Logger.Infof("Forwarded App packet from LocalPort %d using route ID %d", packet.Addr.Port, l.routeID)
	_, err = tr.Write(p)
	return err
}

func (r *Router) forwardLocalAppPacket(packet *app.Packet) error {
	b, err := r.pm.Get(packet.Addr.Remote.Port)
	if err != nil {
		return nil
	}

	p := &app.Packet{
		Addr: &app.LoopAddr{
			Port:   packet.Addr.Remote.Port,
			Remote: app.Addr{PubKey: packet.Addr.Remote.PubKey, Port: packet.Addr.Port},
		},
		Payload: packet.Payload,
	}
	return b.conn.Send(app.FrameSend, p, nil)
}

func (r *Router) requestLoop(appConn *app.Protocol, raddr *app.Addr) (*app.Addr, error) {
	r.Logger.Infof("Requesting new loop to %s", raddr)
	nConf := noise.Config{
		LocalSK:   r.config.SecKey,
		LocalPK:   r.config.PubKey,
		RemotePK:  raddr.PubKey,
		Initiator: true,
	}
	ni, err := noise.KKAndSecp256k1(nConf)
	if err != nil {
		return nil, fmt.Errorf("noise: %s", err)
	}

	msg, err := ni.HandshakeMessage()
	if err != nil {
		return nil, fmt.Errorf("noise handshake: %s", err)
	}

	lport := r.pm.Alloc(appConn)
	if err := r.pm.SetLoop(lport, raddr, &loop{noise: ni}); err != nil {
		return nil, err
	}

	laddr := &app.Addr{PubKey: r.config.PubKey, Port: lport}
	if raddr.PubKey == r.config.PubKey {
		if err := r.confirmLocalLoop(laddr, raddr); err != nil {
			return nil, fmt.Errorf("confirm: %s", err)
		}
		r.Logger.Infof("Created local loop on port %d", laddr.Port)
		return laddr, nil
	}

	forwardRoute, reverseRoute, err := r.fetchBestRoutes(laddr.PubKey, raddr.PubKey)
	if err != nil {
		return nil, fmt.Errorf("route finder: %s", err)
	}

	l := &routing.Loop{LocalPort: laddr.Port, RemotePort: raddr.Port,
		NoiseMessage: msg, Expiry: time.Now().Add(RouteTTL),
		Forward: forwardRoute, Reverse: reverseRoute}

	proto, tr, err := r.setupProto(context.Background())
	if err != nil {
		return nil, err
	}
	defer tr.Close()

	if err := setup.CreateLoop(proto, l); err != nil {
		return nil, fmt.Errorf("route setup: %s", err)
	}

	r.Logger.Infof("Created new loop to %s on port %d", raddr, laddr.Port)
	return laddr, nil
}

func (r *Router) confirmLocalLoop(laddr, raddr *app.Addr) error {
	b, err := r.pm.Get(raddr.Port)
	if err != nil {
		return err
	}

	addrs := [2]*app.Addr{raddr, laddr}
	if err = b.conn.Send(app.FrameConfirmLoop, addrs, nil); err != nil {
		return err
	}

	return nil
}

func (r *Router) confirmLoop(addr *app.LoopAddr, rule routing.Rule, noiseMsg []byte) ([]byte, error) {
	b, err := r.pm.Get(addr.Port)
	if err != nil {
		return nil, err
	}

	ni, msg, err := r.advanceNoiseHandshake(addr, noiseMsg)
	if err != nil {
		return nil, fmt.Errorf("noise handshake: %s", err)
	}

	if err := r.pm.SetLoop(addr.Port, &addr.Remote, &loop{rule.TransportID(), rule.RouteID(), ni}); err != nil {
		return nil, err
	}

	addrs := [2]*app.Addr{&app.Addr{PubKey: r.config.PubKey, Port: addr.Port}, &addr.Remote}
	if err = b.conn.Send(app.FrameConfirmLoop, addrs, nil); err != nil {
		r.Logger.Warnf("Failed to notify App about new loop: %s", err)
	}

	return msg, nil
}

func (r *Router) closeLoop(appConn *app.Protocol, addr *app.LoopAddr) error {
	if err := r.destroyLoop(addr); err != nil {
		r.Logger.Warnf("Failed to remove loop: %s", err)
	}

	proto, tr, err := r.setupProto(context.Background())
	if err != nil {
		return err
	}
	defer tr.Close()

	ld := &setup.LoopData{RemotePK: addr.Remote.PubKey, RemotePort: addr.Remote.Port, LocalPort: addr.Port}
	if err := setup.CloseLoop(proto, ld); err != nil {
		return fmt.Errorf("route setup: %s", err)
	}

	r.Logger.Infof("Closed loop %s", addr)
	return nil
}

func (r *Router) loopClosed(addr *app.LoopAddr) error {
	b, err := r.pm.Get(addr.Port)
	if err != nil {
		return nil
	}

	if err := r.destroyLoop(addr); err != nil {
		r.Logger.Warnf("Failed to remove loop: %s", err)
	}

	if err := b.conn.Send(app.FrameClose, addr, nil); err != nil {
		return err
	}

	r.Logger.Infof("Closed loop %s", addr)
	return nil
}

func (r *Router) destroyLoop(addr *app.LoopAddr) error {
	r.mu.Lock()
	_, ok := r.staticPorts[addr.Port]
	r.mu.Unlock()

	if ok {
		r.pm.RemoveLoop(addr.Port, &addr.Remote) // nolint: errcheck
	} else {
		r.pm.Close(addr.Port)
	}

	return r.rm.RemoveLoopRule(addr)
}

func (r *Router) setupProto(ctx context.Context) (*setup.Protocol, transport.Transport, error) {
	if len(r.config.SetupNodes) == 0 {
		return nil, nil, errors.New("route setup: no nodes")
	}

	// TODO(evanlinjin): need string constant for tp type.
	tr, err := r.tm.CreateTransport(ctx, r.config.SetupNodes[0], dmsg.Type, false)
	if err != nil {
		return nil, nil, fmt.Errorf("transport: %s", err)
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

func (r *Router) advanceNoiseHandshake(addr *app.LoopAddr, noiseMsg []byte) (ni *noise.Noise, noiseRes []byte, err error) {
	var l *loop
	l, _ = r.pm.GetLoop(addr.Port, &addr.Remote) // nolint: errcheck

	if l != nil && l.routeID != 0 {
		err = errors.New("loop already exist")
		return
	}

	if l != nil && l.noise != nil {
		return l.noise, nil, l.noise.ProcessMessage(noiseMsg)
	}

	nConf := noise.Config{
		LocalSK:   r.config.SecKey,
		LocalPK:   r.config.PubKey,
		RemotePK:  addr.Remote.PubKey,
		Initiator: false,
	}
	ni, err = noise.KKAndSecp256k1(nConf)
	if err != nil {
		return
	}
	if err = ni.ProcessMessage(noiseMsg); err != nil {
		return
	}
	noiseRes, err = ni.HandshakeMessage()
	return
}

// IsSetupTransport checks whether `tr` is running in the `setup` mode.
func (r *Router) IsSetupTransport(tr *transport.ManagedTransport) bool {
	for _, pk := range r.config.SetupNodes {
		remote, ok := r.tm.Remote(tr.Edges())
		if ok && (remote == pk) {
			return true
		}
	}

	return false
}
