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
	Serve(ctx context.Context) error
	FindRoutesAndSetupLoop(lm app.LoopMeta, nsMsg []byte) error
	ForwardPacket(tpID uuid.UUID, rtID routing.RouteID, payload []byte) error
	CloseLoop(lm app.LoopMeta) error
	Close() error
}

// New constructs a new router.
func New(l *logging.Logger, tpm *transport.Manager, rt routing.Table, rfc routeFinder.Client, pm ProcManager, conf *Config) Router {
	r := &router{
		log:      l,
		conf:     conf,
		tpm:      tpm,
		rtm:      NewRoutingTableManager(l, rt, DefaultRouteKeepalive, DefaultRouteCleanupDuration),
		pm:       pm,
		rfc:      rfc,
		setupPKs: make(map[cipher.PubKey]struct{}),
	}
	for _, pk := range conf.SetupNodes {
		r.setupPKs[pk] = struct{}{}
	}
	return r
}

type router struct {
	log  *logging.Logger
	conf *Config
	tpm  *transport.Manager
	rtm  *RoutingTableManager
	pm   ProcManager
	rfc  routeFinder.Client

	setupPKs map[cipher.PubKey]struct{} // map of remote PKs that are of setup nodes
}

// Serve starts transport listening loop.
func (r *router) Serve(ctx context.Context) error {

	// listens for transports.
	go func() {
		acceptCh, dialCh := r.tpm.Observe()
		for {
			select {
			case tp, ok := <-acceptCh:
				if !ok {
					return
				}
				if !r.isSetupTp(tp) {
					go r.serveTp(tp, r.multiplexTransportPacket)
				} else {
					go r.serveTp(tp, r.multiplexSetupPacket)
				}
			case tp, ok := <-dialCh:
				if !ok {
					return
				}
				if !r.isSetupTp(tp) {
					go r.serveTp(tp, r.multiplexTransportPacket)
				}
			}
		}
	}()

	// runs the routing table cleanup event loop.
	go r.rtm.Run()

	r.log.Info("Starting router")
	return r.tpm.Serve(ctx)
}

// determines if the given transport is established with a setup node.
func (r *router) isSetupTp(tp transport.Transport) bool {
	pk, ok := r.tpm.Remote(tp.Edges())
	if !ok {
		return false
	}
	_, ok = r.setupPKs[pk]
	return ok
}

// serves a given transport with the 'handler' running in a loop.
// the loop exits on error.
func (r *router) serveTp(tp transport.Transport, handle tpHandlerFunc) {
	for {
		if err := handle(tp); err != nil && err != io.EOF {
			r.log.Warnf("Stopped serving Transport: %s", err)
			return
		}
	}
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

func makeLoopMeta(lPK cipher.PubKey, ld setup.LoopData) app.LoopMeta {
	return app.LoopMeta{
		Local:  app.LoopAddr{PubKey: lPK, Port: ld.LocalPort},
		Remote: app.LoopAddr{PubKey: ld.RemotePK, Port: ld.RemotePort},
	}
}
