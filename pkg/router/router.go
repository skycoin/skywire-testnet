// Package router implements package router for skywire visor.
package router

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/rpc"
	"sync"
	"time"

	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/routefinder/rfclient"
	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/setup/setupclient"
	"github.com/skycoin/skywire/pkg/snet"
	"github.com/skycoin/skywire/pkg/transport"
)

const (
	// DefaultRouteKeepAlive is the default expiration interval for routes
	DefaultRouteKeepAlive = 2 * time.Hour // TODO: change
	acceptSize            = 1024

	minHops = 0
	maxHops = 50
)

var log = logging.MustGetLogger("router")

// Config configures Router.
type Config struct {
	Logger           *logging.Logger
	PubKey           cipher.PubKey
	SecKey           cipher.SecKey
	TransportManager *transport.Manager
	RoutingTable     routing.Table
	RouteFinder      rfclient.Client
	SetupNodes       []cipher.PubKey
}

// SetDefaults sets default values for certain empty values.
func (c *Config) SetDefaults() {
	if c.Logger == nil {
		c.Logger = log
	}
}

type DialOptions struct {
	MinForwardRts int
	MaxForwardRts int
	MinConsumeRts int
	MaxConsumeRts int
}

var DefaultDialOptions = &DialOptions{
	MinForwardRts: 1,
	MaxForwardRts: 1,
	MinConsumeRts: 1,
	MaxConsumeRts: 1,
}

type Router interface {
	io.Closer

	// DialRoutes dials to a given visor of 'rPK'.
	// 'lPort'/'rPort' specifies the local/remote ports respectively.
	// A nil 'opts' input results in a value of '1' for all DialOptions fields.
	// A single call to DialRoutes should perform the following:
	// - Find routes via RouteFinder (in one call).
	// - Setup routes via SetupNode (in one call).
	// - Save to routing.Table and internal RouteGroup map.
	// - Return RouteGroup if successful.
	DialRoutes(ctx context.Context, rPK cipher.PubKey, lPort, rPort routing.Port, opts *DialOptions) (*RouteGroup, error)

	// AcceptRoutes should block until we receive an visorAddRules packet from SetupNode that contains ConsumeRule(s) or ForwardRule(s).
	// Then the following should happen:
	// - Save to routing.Table and internal RouteGroup map.
	// - Return the RoutingGroup.
	AcceptRoutes() (*RouteGroup, error)

	Serve(context.Context) error

	SetupIsTrusted(cipher.PubKey) bool
}

// Router implements node.PacketRouter. It manages routing table by
// communicating with setup nodes, forward packets according to local
// rules and manages loops for apps.
type router struct {
	mx           sync.Mutex
	wg           sync.WaitGroup
	conf         *Config
	logger       *logging.Logger
	n            *snet.Network
	sl           *snet.Listener
	accept       chan routing.EdgeRules
	trustedNodes map[cipher.PubKey]struct{}
	tm           *transport.Manager
	rt           routing.Table
	rfc          rfclient.Client                         // route finder client
	rgs          map[routing.RouteDescriptor]*RouteGroup // route groups to push incoming reads from transports.
	rpcSrv       *rpc.Server
}

// New constructs a new Router.
func New(n *snet.Network, config *Config) (*router, error) {
	config.SetDefaults()

	sl, err := n.Listen(snet.DmsgType, snet.AwaitSetupPort)
	if err != nil {
		return nil, err
	}

	trustedNodes := make(map[cipher.PubKey]struct{})
	for _, node := range config.SetupNodes {
		trustedNodes[node] = struct{}{}
	}

	r := &router{
		conf:         config,
		logger:       config.Logger,
		n:            n,
		tm:           config.TransportManager,
		rt:           config.RoutingTable,
		sl:           sl,
		rfc:          config.RouteFinder,
		rpcSrv:       rpc.NewServer(),
		accept:       make(chan routing.EdgeRules, acceptSize),
		trustedNodes: trustedNodes,
	}

	if err := r.rpcSrv.Register(NewGateway(r)); err != nil {
		return nil, fmt.Errorf("failed to register RPC server")
	}

	return r, nil
}

// DialRoutes dials to a given visor of 'rPK'.
// 'lPort'/'rPort' specifies the local/remote ports respectively.
// A nil 'opts' input results in a value of '1' for all DialOptions fields.
// A single call to DialRoutes should perform the following:
// - Find routes via RouteFinder (in one call).
// - Setup routes via SetupNode (in one call).
// - Save to routing.Table and internal RouteGroup map.
// - Return RouteGroup if successful.
func (r *router) DialRoutes(ctx context.Context, rPK cipher.PubKey, lPort, rPort routing.Port, opts *DialOptions) (*RouteGroup, error) {
	if opts == nil {
		opts = DefaultDialOptions
	}

	lPK := r.conf.PubKey
	forwardDesc := routing.NewRouteDescriptor(lPK, rPK, lPort, rPort)

	forwardPath, reversePath, err := r.fetchBestRoutes(lPK, rPK, opts)
	if err != nil {
		return nil, fmt.Errorf("route finder: %s", err)
	}

	req := routing.BidirectionalRoute{
		Desc:      forwardDesc,
		KeepAlive: DefaultRouteKeepAlive,
		Forward:   forwardPath,
		Reverse:   reversePath,
	}

	rules, err := setupclient.DialRouteGroup(ctx, r.logger, r.n, r.conf.SetupNodes, req)
	if err != nil {
		return nil, err
	}

	if err := r.saveRoutingRules(rules.Forward, rules.Reverse); err != nil {
		return nil, err
	}

	rg := r.saveRouteGroupRules(rules)

	r.logger.Infof("Created new routes to %s on port %d", rPK, lPort)
	return rg, nil
}

// AcceptsRoutes should block until we receive an AddRules packet from SetupNode that contains ConsumeRule(s) or ForwardRule(s).
// Then the following should happen:
// - Save to routing.Table and internal RouteGroup map.
// - Return the RoutingGroup.
func (r *router) AcceptRoutes() (*RouteGroup, error) {
	rules := <-r.accept

	if err := r.saveRoutingRules(rules.Forward, rules.Reverse); err != nil {
		return nil, err
	}

	rg := r.saveRouteGroupRules(rules)
	return rg, nil
}

// Serve starts transport listening loop.
func (r *router) Serve(ctx context.Context) error {
	r.logger.Info("Starting router")

	go r.serveTransportManager(ctx)

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.serveSetup()
	}()

	r.tm.Serve(ctx)
	return nil
}

func (r *router) serveTransportManager(ctx context.Context) {
	for {
		packet, err := r.tm.ReadPacket()
		if err != nil {
			r.logger.WithError(err).Errorf("Failed to read packet")
			return
		}

		if err := r.handleTransportPacket(ctx, packet); err != nil {
			if err == transport.ErrNotServing {
				r.logger.WithError(err).Warnf("Stopped serving Transport.")
				return
			}
			r.logger.Warnf("Failed to handle transport frame: %v", err)
		}
	}
}

func (r *router) serveSetup() {
	for {
		conn, err := r.sl.AcceptConn()
		if err != nil {
			r.logger.WithError(err).Warnf("setup client stopped serving")
		}

		if !r.SetupIsTrusted(conn.RemotePK()) {
			r.logger.Warnf("closing conn from untrusted setup node: %v", conn.Close())
			continue
		}
		r.logger.Infof("handling setup request: setupPK(%s)", conn.RemotePK())

		go r.rpcSrv.ServeConn(conn)

		if err := conn.Close(); err != nil {
			log.WithError(err).Warn("Failed to close connection")
		}
	}
}

func (r *router) saveRouteGroupRules(rules routing.EdgeRules) *RouteGroup {
	r.mx.Lock()
	defer r.mx.Unlock()

	rg, ok := r.rgs[rules.Desc]
	if !ok || rg == nil {
		rg = NewRouteGroup(r.rt, rules.Desc)
		r.rgs[rules.Desc] = rg
	}
	rg.fwd = append(rg.fwd, rules.Forward)
	rg.rvs = append(rg.fwd, rules.Reverse)

	// TODO: fill transports
	return rg
}

// TODO: handle other packet types
func (r *router) handleTransportPacket(ctx context.Context, packet routing.Packet) error {
	rule, err := r.GetRule(packet.RouteID())
	if err != nil {
		return err
	}

	desc := rule.RouteDescriptor()
	rg, ok := r.rgs[desc]
	if !ok {
		return errors.New("route descriptor does not exist")
	}
	if rg == nil {
		return errors.New("RouteGroup is nil")
	}

	r.logger.Infof("Got new remote packet with route ID %d. Using rule: %s", packet.RouteID(), rule)
	switch t := rule.Type(); t {
	case routing.RuleForward, routing.RuleIntermediaryForward:
		return r.forwardPacket(ctx, packet.Payload(), rule)
	default:
		rg.readCh <- packet.Payload()
		return nil
	}
}

// GetRule gets routing rule.
func (r *router) GetRule(routeID routing.RouteID) (routing.Rule, error) {
	rule, err := r.rt.Rule(routeID)
	if err != nil {
		return nil, fmt.Errorf("routing table: %s", err)
	}

	if rule == nil {
		return nil, errors.New("unknown RouteID")
	}

	// TODO(evanlinjin): This is a workaround for ensuring the read-in rule is of the correct size.
	// Sometimes it is not, causing a segfault later down the line.
	if len(rule) < routing.RuleHeaderSize {
		return nil, errors.New("corrupted rule")
	}

	return rule, nil
}

// Close safely stops Router.
func (r *router) Close() error {
	if r == nil {
		return nil
	}
	r.logger.Info("Closing all App connections and Loops")

	if err := r.sl.Close(); err != nil {
		r.logger.WithError(err).Warnf("closing route_manager returned error")
	}
	r.wg.Wait()

	return r.tm.Close()
}

func (r *router) forwardPacket(ctx context.Context, payload []byte, rule routing.Rule) error {
	tp := r.tm.Transport(rule.NextTransportID())
	if tp == nil {
		return errors.New("unknown transport")
	}
	packet := routing.MakeDataPacket(rule.KeyRouteID(), payload)
	if err := tp.WritePacket(ctx, packet); err != nil {
		return err
	}
	r.logger.Infof("Forwarded packet via Transport %s using rule %d", rule.NextTransportID(), rule.KeyRouteID())
	return nil
}

// func (r *router) consumePacket(payload []byte, rule routing.Rule) error {
// 	laddr := routing.Addr{Port: rule.RouteDescriptor().SrcPort()}
// 	raddr := routing.Addr{PubKey: rule.RouteDescriptor().DstPK(), Port: rule.RouteDescriptor().DstPort()}
//
// 	route := routing.Route{Desc: routing.NewRouteDescriptor(laddr.PubKey, raddr.PubKey, laddr.Port, raddr.Port)}
// 	p := &app.Packet{Desc: route.Desc, Payload: payload}
// 	b, err := r.pm.Get(rule.RouteDescriptor().SrcPort())
// 	if err != nil {
// 		return err
// 	}
// 	if err := b.conn.Send(app.FrameSend, p, nil); err != nil { // TODO: Stuck here.
// 		return err
// 	}
//
// 	r.logger.Infof("Forwarded packet to App on Port %d", rule.RouteDescriptor().SrcPort())
// 	return nil
// }

// RemoveRouteDescriptor removes loop rule.
func (r *router) RemoveRouteDescriptor(desc routing.RouteDescriptor) {
	rules := r.rt.AllRules()
	for _, rule := range rules {
		if rule.Type() != routing.RuleConsume {
			continue
		}

		rd := rule.RouteDescriptor()
		if rd.DstPK() == desc.DstPK() && rd.DstPort() == desc.DstPort() && rd.SrcPort() == desc.SrcPort() {
			r.rt.DelRules([]routing.RouteID{rule.KeyRouteID()})
			return
		}
	}
}

func (r *router) fetchBestRoutes(source, destination cipher.PubKey, opts *DialOptions) (fwd routing.Path, rev routing.Path, err error) {
	// TODO: use opts

	r.logger.Infof("Requesting new routes from %s to %s", source, destination)

	timer := time.NewTimer(time.Second * 10)
	defer timer.Stop()

	forward := [2]cipher.PubKey{source, destination}
	backward := [2]cipher.PubKey{destination, source}

fetchRoutesAgain:
	ctx := context.Background()
	paths, err := r.conf.RouteFinder.FindRoutes(ctx, []routing.PathEdges{forward, backward},
		&rfclient.RouteOptions{MinHops: minHops, MaxHops: maxHops})
	if err != nil {
		select {
		case <-timer.C:
			return nil, nil, err
		default:
			goto fetchRoutesAgain
		}
	}

	r.logger.Infof("Found routes Forward: %s. Reverse %s", paths[forward], paths[backward])
	return paths[forward][0], paths[backward][0], nil
}

// SetupIsTrusted checks if setup node is trusted.
func (r *router) SetupIsTrusted(sPK cipher.PubKey) bool {
	_, ok := r.trustedNodes[sPK]
	return ok
}

func (r *router) saveRoutingRules(rules ...routing.Rule) error {
	for _, rule := range rules {
		if err := r.rt.SaveRule(rule); err != nil {
			return fmt.Errorf("routing table: %s", err)
		}

		r.logger.Infof("Save new Routing Rule with ID %d %s", rule.KeyRouteID(), rule)
	}

	return nil
}

func (r *router) occupyRouteID(n uint8) ([]routing.RouteID, error) {
	var ids = make([]routing.RouteID, n)
	for i := range ids {
		routeID, err := r.rt.ReserveKey()
		if err != nil {
			return nil, err
		}
		ids[i] = routeID
	}
	return ids, nil
}

func (r *router) routeGroup(desc routing.RouteDescriptor) (*RouteGroup, bool) {
	r.mx.Lock()
	defer r.mx.Unlock()

	rg, ok := r.rgs[desc]
	return rg, ok
}
