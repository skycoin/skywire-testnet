// Package router implements package router for skywire node.
package router

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"

	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/cipher"

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

// Config configures router.
type Config struct {
	PubKey     cipher.PubKey
	SecKey     cipher.SecKey
	SetupNodes []cipher.PubKey
}

// Router manages routing table by communicating with setup nodes, forward packets according to local rules and
// manages loops for apps.
type Router interface {
	Serve(ctx context.Context, am ProcManager) error
	FindRoutesAndSetupLoop(lm app.LoopMeta, nsMsg []byte) error
	ForwardPacket(tpID uuid.UUID, rtID routing.RouteID, payload []byte) error
	CloseLoop(lm app.LoopMeta) error
	Close() error
}

// New constructs a new router.
func New(l *logging.Logger, tpm *transport.Manager, rt routing.Table, rf routeFinder.Client, conf *Config) Router {
	return &router{
		log: l,
		c:   conf,
		tpm: tpm,
		rtm: NewRoutingTableManager(l, rt, DefaultRouteKeepalive, DefaultRouteCleanupDuration),
		rfc: rf,
	}
}

type router struct {
	log *logging.Logger
	c   *Config
	tpm *transport.Manager
	rtm *RoutingTableManager
	rfc routeFinder.Client
}

// Serve starts transport listening loop.
func (r *router) Serve(ctx context.Context, am ProcManager) error {

	setupPKs := make(map[cipher.PubKey]struct{})
	for _, pk := range r.c.SetupNodes {
		setupPKs[pk] = struct{}{}
	}

	// determines if the given transport is established with a setup node.
	isSetup := func(tp transport.Transport) bool {
		pk, ok := r.tpm.Remote(tp.Edges())
		if !ok {
			return false
		}
		_, ok = setupPKs[pk]
		return ok
	}

	// serves a given transport with the 'handler' running in a loop.
	// the loop exits on error.
	serve := func(tp transport.Transport, handle func(ProcManager, io.ReadWriter) error) {
		for {
			if err := handle(am, tp); err != nil && err != io.EOF {
				r.log.Warnf("Stopped serving Transport: %s", err)
				return
			}
		}
	}

	// listens for transports.
	go func() {
		acceptCh, dialCh := r.tpm.Observe()
		for {
			select {
			case tp, ok := <-acceptCh:
				if !ok {
					return
				}
				if !isSetup(tp) {
					go serve(tp, r.handleTransport)
				} else {
					go serve(tp, r.handleSetup)
				}
			case tp, ok := <-dialCh:
				if !ok {
					return
				}
				if !isSetup(tp) {
					go serve(tp, r.handleTransport)
				}
			}
		}
	}()

	// runs the routing table cleanup event loop.
	go r.rtm.Run()

	r.log.Info("Starting router")
	return r.tpm.Serve(ctx)
}

// Close safely stops router.
func (r *router) Close() error {
	r.log.Info("Closing all App connections and Loops")
	r.rtm.Stop()
	return r.tpm.Close()
}

func (r *router) ForwardPacket(tpID uuid.UUID, rtID routing.RouteID, payload []byte) error {
	tp := r.tpm.Transport(tpID)
	if tp == nil {
		return errors.New("transport not found")
	}
	_, err := tp.Write(routing.MakePacket(rtID, payload))
	return err
}

func (r *router) FindRoutesAndSetupLoop(lm app.LoopMeta, nsMsg []byte) error {
	fwdRt, rvsRt, err := r.fetchBestRoutes(lm.Local.PubKey, lm.Remote.PubKey)
	if err != nil {
		return fmt.Errorf("route finder: %s", err)
	}
	loop := routing.Loop{
		LocalPort:    lm.Local.Port,
		RemotePort:   lm.Remote.Port,
		NoiseMessage: nsMsg,
		Expiry:       time.Now().Add(RouteTTL),
		Forward:      fwdRt,
		Reverse:      rvsRt,
	}
	sProto, tp, err := r.setupProto(context.Background())
	if err != nil {
		return err
	}
	defer func() { _ = tp.Close() }()
	if err := setup.CreateLoop(sProto, &loop); err != nil {
		return fmt.Errorf("route setup: %s", err)
	}
	return nil
}

func (r *router) CloseLoop(lm app.LoopMeta) error {
	setupProto, setupTP, err := r.setupProto(context.Background())
	if err != nil {
		return err
	}
	defer func() { _ = setupTP.Close() }()
	ld := setup.LoopData{RemotePK: lm.Remote.PubKey, RemotePort: lm.Remote.Port, LocalPort: lm.Local.Port}
	if err := setup.CloseLoop(setupProto, &ld); err != nil {
		return fmt.Errorf("route setup: %s", err)
	}
	r.log.Infof("Closed loop %s", lm)
	return nil
}

func (r *router) setupProto(ctx context.Context) (*setup.Protocol, transport.Transport, error) {
	if len(r.c.SetupNodes) == 0 {
		return nil, nil, errors.New("route setup: no nodes")
	}

	tr, err := r.tpm.CreateTransport(ctx, r.c.SetupNodes[0], "messaging", false)
	if err != nil {
		return nil, nil, fmt.Errorf("transport: %s", err)
	}

	sProto := setup.NewSetupProtocol(tr)
	return sProto, tr, nil
}

func (r *router) fetchBestRoutes(srcPK, dstPK cipher.PubKey) (fwdRt routing.Route, rvsRt routing.Route, err error) {
	r.log.Infof("Requesting new routes from %s to %s", srcPK, dstPK)
	forwardRoutes, reverseRoutes, err := r.rfc.PairedRoutes(srcPK, dstPK, minHops, maxHops)
	if err != nil {
		return nil, nil, err
	}

	r.log.Infof("Found routes Forward: %s. Reverse %s", forwardRoutes, reverseRoutes)
	return forwardRoutes[0], reverseRoutes[0], nil
}

func (r *router) handleTransport(am ProcManager, rw io.ReadWriter) error {
	packet := make(routing.Packet, 6)
	if _, err := io.ReadFull(rw, packet); err != nil {
		return err
	}

	payload := make([]byte, packet.Size())
	if _, err := io.ReadFull(rw, payload); err != nil {
		return err
	}

	rule, err := r.rtm.Rule(packet.RouteID())
	if err != nil {
		return err
	}

	r.log.Infof("Got new remote packet with route ID %d. Using rule: %s", packet.RouteID(), rule)

	switch rule.Type() {
	case routing.RuleForward:
		return r.ForwardPacket(rule.TransportID(), rule.RouteID(), payload)

	case routing.RuleApp:
		proc, ok := am.ProcOfPort(rule.LocalPort())
		if !ok {
			return ErrProcNotFound
		}
		lm := app.LoopMeta{
			Local:  app.LoopAddr{PubKey: r.c.PubKey, Port: rule.LocalPort()},
			Remote: app.LoopAddr{PubKey: rule.RemotePK(), Port: rule.RemotePort()},
		}
		return proc.ConsumePacket(lm, payload)

	default:
		return errors.New("associated rule has invalid type")
	}
}

func (r *router) handleSetup(am ProcManager, rw io.ReadWriter) error {

	// triggered when a 'AddRules' packet is received from SetupNode
	addRules := func(rules []routing.Rule) ([]routing.RouteID, error) {
		res := make([]routing.RouteID, len(rules))
		for idx, rule := range rules {
			routeID, err := r.rtm.AddRule(rule)
			if err != nil {
				return nil, fmt.Errorf("routing table: %s", err)
			}
			res[idx] = routeID
			r.log.Infof("Added new Routing Rule with ID %d %s", routeID, rule)
		}
		return res, nil
	}

	// triggered when a 'DeleteRules' packet is received from SetupNode
	deleteRules := func(rtIDs []routing.RouteID) ([]routing.RouteID, error) {
		err := r.rtm.DeleteRules(rtIDs...)
		if err != nil {
			return nil, fmt.Errorf("routing table: %s", err)
		}

		r.log.Infof("Removed Routing Rules with IDs %s", rtIDs)
		return rtIDs, nil
	}

	// triggered when a 'ConfirmLoop' packet is received from SetupNode
	confirmLoop := func(ld setup.LoopData) ([]byte, error) {
		lm := makeLoopMeta(r.c.PubKey, ld)

		appRtID, appRule, ok := r.rtm.FindAppRule(lm)
		if !ok {
			return nil, errors.New("unknown loop")
		}
		fwdRule, err := r.rtm.FindFwdRule(ld.RouteID)
		if err != nil {
			return nil, err
		}

		proc, ok := am.ProcOfPort(lm.Local.Port)
		if !ok {
			return nil, ErrProcNotFound
		}
		msg, err := proc.ConfirmLoop(lm, fwdRule.TransportID(), fwdRule.RouteID(), ld.NoiseMessage)
		if err != nil {
			return nil, fmt.Errorf("confirm: %s", err)
		}

		r.log.Infof("Setting reverse route ID %d for rule with ID %d", ld.RouteID, appRtID)
		appRule.SetRouteID(ld.RouteID)

		if err := r.rtm.SetRule(appRtID, appRule); err != nil {
			return nil, fmt.Errorf("routing table: %s", err)
		}

		r.log.Infof("Confirmed loop with %s", lm.Remote)
		return msg, nil
	}

	// triggered when a 'LoopClosed' packet is received from SetupNode
	loopClosed := func(ld setup.LoopData) error {
		lm := makeLoopMeta(r.c.PubKey, ld)

		proc, ok := am.ProcOfPort(lm.Local.Port)
		if !ok {
			return ErrProcNotFound
		}
		return proc.ConfirmCloseLoop(lm)
	}

	proto := setup.NewSetupProtocol(rw)

	t, body, err := proto.ReadPacket()
	if err != nil {
		return err
	}

	reject := func(err error) error {
		r.log.Infof("Setup request with type %s failed: %s", t, err)
		return proto.WritePacket(setup.RespFailure, err.Error())
	}

	respondWith := func(v interface{}, err error) error {
		if err != nil {
			return reject(err)
		}
		return proto.WritePacket(setup.RespSuccess, v)
	}

	r.log.Infof("Got new Setup request with type %s", t)

	switch t {
	case setup.PacketAddRules:
		var rules []routing.Rule
		if err := json.Unmarshal(body, &rules); err != nil {
			return reject(err)
		}
		return respondWith(addRules(rules))

	case setup.PacketDeleteRules:
		var rtIDs []routing.RouteID
		if err := json.Unmarshal(body, &rtIDs); err != nil {
			return reject(err)
		}
		return respondWith(deleteRules(rtIDs))

	case setup.PacketConfirmLoop:
		var ld setup.LoopData
		if err := json.Unmarshal(body, &ld); err != nil {
			return reject(err)
		}
		return respondWith(confirmLoop(ld))

	case setup.PacketLoopClosed:
		var ld setup.LoopData
		if err := json.Unmarshal(body, &ld); err != nil {
			return reject(err)
		}
		return respondWith(nil, loopClosed(ld))

	default:
		return reject(errors.New("unknown foundation packet"))
	}
}

func makeLoopMeta(lPK cipher.PubKey, ld setup.LoopData) app.LoopMeta {
	return app.LoopMeta{
		Local:  app.LoopAddr{PubKey: lPK, Port: ld.LocalPort},
		Remote: app.LoopAddr{PubKey: ld.RemotePK, Port: ld.RemotePort},
	}
}
