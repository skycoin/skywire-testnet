// Package router implements package router for skywire node.
package router

import (
	"context"
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
		log:  l,
		conf: conf,
		tpm:  tpm,
		rtm:  NewRoutingTableManager(l, rt, DefaultRouteKeepalive, DefaultRouteCleanupDuration),
		rfc:  rf,
	}
}

type router struct {
	log  *logging.Logger
	conf *Config
	tpm  *transport.Manager
	rtm  *RoutingTableManager
	rfc  routeFinder.Client
}

func (r *router) String() string {
	return fmt.Sprintf("router{%v, %v, %v, %v, %v }\n", r.log, r.conf, r.tpm, r.rtm, r.rfc)
}

// Serve starts transport listening loop.
func (r *router) Serve(ctx context.Context, pm ProcManager) error {

	rh := makeRouterHandlers(r, pm)

	// listens for transports.
	go func() {
		acceptCh, dialCh := r.tpm.Observe()
		for {
			select {
			case tp, ok := <-acceptCh:
				if !ok {
					return
				}
				if !rh.isSetup()(tp) {
					go rh.serve()(tp, r.handleTransport)
				} else {
					go rh.serve()(tp, r.handleSetup)
				}
			case tp, ok := <-dialCh:
				if !ok {
					return
				}
				if !rh.isSetup()(tp) {
					go rh.serve()(tp, r.handleTransport)
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
	if len(r.conf.SetupNodes) == 0 {
		return nil, nil, errors.New("route setup: no nodes")
	}

	tr, err := r.tpm.CreateTransport(ctx, r.conf.SetupNodes[0], "messaging", false)
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
			Local:  app.LoopAddr{PubKey: r.conf.PubKey, Port: rule.LocalPort()},
			Remote: app.LoopAddr{PubKey: rule.RemotePK(), Port: rule.RemotePort()},
		}
		return proc.ConsumePacket(lm, payload)

	default:
		return errors.New("associated rule has invalid type")
	}
}

func (r *router) handleSetup(am ProcManager, rw io.ReadWriter) error {

	sh, err := makeSetupHandlers(r, am, rw)
	if err != nil {
		return err
	}

	return sh.handle()
}

func makeLoopMeta(lPK cipher.PubKey, ld setup.LoopData) app.LoopMeta {
	return app.LoopMeta{
		Local:  app.LoopAddr{PubKey: lPK, Port: ld.LocalPort},
		Remote: app.LoopAddr{PubKey: ld.RemotePK, Port: ld.RemotePort},
	}
}
