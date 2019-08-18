package setup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/dmsg/disc"
	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/metrics"
	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/transport"
	"github.com/skycoin/skywire/pkg/transport/dmsg"
)

// Hop is a wrapper around transport hop to add functionality
type Hop struct {
	*routing.Hop
	routeID routing.RouteID
}

// Node performs routes setup operations over messaging channel.
type Node struct {
	Logger    *logging.Logger
	messenger *dmsg.Client
	srvCount  int
	metrics   metrics.Recorder
}

// NewNode constructs a new SetupNode.
func NewNode(conf *Config, metrics metrics.Recorder) (*Node, error) {
	pk := conf.PubKey
	sk := conf.SecKey

	logger := logging.NewMasterLogger()
	if lvl, err := logging.LevelFromString(conf.LogLevel); err == nil {
		logger.SetLevel(lvl)
	}
	messenger := dmsg.NewClient(pk, sk, disc.NewHTTP(conf.Messaging.Discovery), dmsg.SetLogger(logger.PackageLogger(dmsg.Type)))

	return &Node{
		Logger:    logger.PackageLogger("routesetup"),
		metrics:   metrics,
		messenger: messenger,
		srvCount:  conf.Messaging.ServerCount,
	}, nil
}

// Serve starts transport listening loop.
func (sn *Node) Serve(ctx context.Context) error {
	if sn.srvCount > 0 {
		if err := sn.messenger.InitiateServerConnections(ctx, sn.srvCount); err != nil {
			return fmt.Errorf("messaging: %s", err)
		}
		sn.Logger.Info("Connected to messaging servers")
	}

	sn.Logger.Info("Starting Setup Node")

	for {
		tp, err := sn.messenger.Accept(ctx)
		if err != nil {
			return err
		}
		go func(tp transport.Transport) {
			if err := sn.serveTransport(ctx, tp); err != nil {
				sn.Logger.Warnf("Failed to serve Transport: %s", err)
			}
		}(tp)
	}
}

func (sn *Node) serveTransport(ctx context.Context, tr transport.Transport) error {
	ctx, cancel := context.WithTimeout(ctx, ServeTransportTimeout)
	defer cancel()

	proto := NewSetupProtocol(tr)
	sp, data, err := proto.ReadPacket()
	if err != nil {
		return err
	}

	sn.Logger.Infof("Got new Setup request with type %s: %s", sp, string(data))
	defer sn.Logger.Infof("Completed Setup request with type %s: %s", sp, string(data))

	startTime := time.Now()
	switch sp {
	case PacketCreateLoop:
		var ld routing.LoopDescriptor
		if err = json.Unmarshal(data, &ld); err == nil {
			err = sn.createLoop(ctx, ld)
		}
	case PacketCloseLoop:
		var ld routing.LoopData
		if err = json.Unmarshal(data, &ld); err == nil {
			err = sn.closeLoop(ctx, ld.Loop.Remote.PubKey, routing.LoopData{
				Loop: routing.Loop{
					Remote: ld.Loop.Local,
					Local:  ld.Loop.Remote,
				},
			})
		}
	default:
		err = errors.New("unknown foundation packet")
	}
	sn.metrics.Record(time.Since(startTime), err != nil)

	if err != nil {
		sn.Logger.Infof("Setup request with type %s failed: %s", sp, err)
		return proto.WritePacket(RespFailure, err)
	}

	return proto.WritePacket(RespSuccess, nil)
}

func (sn *Node) createLoop(ctx context.Context, ld routing.LoopDescriptor) error {
	sn.Logger.Infof("Creating new Loop %s", ld)
	rRouteID, err := sn.createRoute(ctx, ld.Expiry, ld.Reverse, ld.Loop.Local.Port, ld.Loop.Remote.Port)
	if err != nil {
		return err
	}

	fRouteID, err := sn.createRoute(ctx, ld.Expiry, ld.Forward, ld.Loop.Remote.Port, ld.Loop.Local.Port)
	if err != nil {
		return err
	}

	if len(ld.Forward) == 0 || len(ld.Reverse) == 0 {
		return nil
	}

	initiator := ld.Initiator()
	responder := ld.Responder()

	ldR := routing.LoopData{
		Loop: routing.Loop{
			Remote: routing.Addr{
				PubKey: initiator,
				Port:   ld.Loop.Local.Port,
			},
			Local: routing.Addr{
				PubKey: responder,
				Port:   ld.Loop.Remote.Port,
			},
		},
		RouteID: rRouteID,
	}
	if err := sn.connectLoop(ctx, responder, ldR); err != nil {
		sn.Logger.Warnf("Failed to confirm loop with responder: %s", err)
		return fmt.Errorf("loop connect: %s", err)
	}

	ldI := routing.LoopData{
		Loop: routing.Loop{
			Remote: routing.Addr{
				PubKey: responder,
				Port:   ld.Loop.Remote.Port,
			},
			Local: routing.Addr{
				PubKey: initiator,
				Port:   ld.Loop.Local.Port,
			},
		},
		RouteID: fRouteID,
	}
	if err := sn.connectLoop(ctx, initiator, ldI); err != nil {
		sn.Logger.Warnf("Failed to confirm loop with initiator: %s", err)
		if err := sn.closeLoop(ctx, responder, ldR); err != nil {
			sn.Logger.Warnf("Failed to close loop: %s", err)
		}
		return fmt.Errorf("loop connect: %s", err)
	}

	sn.Logger.Infof("Created Loop %s", ld)
	return nil
}

func (sn *Node) createRoute(ctx context.Context, expireAt time.Time, route routing.Route, rport, lport routing.Port) (routing.RouteID, error) {
	if len(route) == 0 {
		return 0, nil
	}

	sn.Logger.Infof("Creating new Route %s", route)

	r := make(routing.Route, len(route)+1)
	r[0] = &routing.Hop{
		Transport: route[0].Transport,
		To:        route[0].From,
	}
	copy(r[1:], route)

	initiator := route[0].From

	// indicate errors occurred during rules setup
	rulesSetupErrs := make(chan error, len(r))
	// routeIDsCh is an array of chans used to pass the requested route IDs around the gorouines.
	// We do it in a fan fashion here. We create as many goroutines as there are rules to be applied.
	// Goroutine[idx] requests visor node for a route ID. It passes this route ID through a chan to a goroutine[idx-1].
	// In turn, goroutine[idx] waits for a route ID from chan[idx]. Thus, goroutine[len(r)] doesn't get a route ID and
	// uses 0 instead, goroutine[0] doesn't pass its route ID to anyone
	routeIDsCh := make([]chan routing.RouteID, 0, len(r))
	for range r {
		routeIDsCh = append(routeIDsCh, make(chan routing.RouteID, 2))
	}

	// chan to receive the resulting route ID from a goroutine
	resultingRouteIDCh := make(chan routing.RouteID, 2)

	// context to cancel rule setup in case of errors
	ctx, cancel := context.WithCancel(context.Background())
	for idx := len(r) - 1; idx >= 0; idx-- {
		var routeIDChIn, routeIDChOut chan routing.RouteID
		if idx > 0 {
			routeIDChOut = routeIDsCh[idx-1]
		}
		var nextTransport uuid.UUID
		var rule routing.Rule
		if idx != len(r)-1 {
			routeIDChIn = routeIDsCh[idx]
			nextTransport = r[idx+1].Transport
			rule = routing.ForwardRule(expireAt, 0, nextTransport, 0)
		} else {
			rule = routing.AppRule(expireAt, 0, initiator, lport, rport, 0)
		}

		go func(idx int, pubKey cipher.PubKey, rule routing.Rule, routeIDChIn <-chan routing.RouteID,
			routeIDChOut chan<- routing.RouteID) {
			routeID, err := sn.addRule(ctx, pubKey, rule, routeIDChIn, routeIDChOut)
			if err != nil {
				// filter out context cancellation errors
				if err == context.Canceled {
					rulesSetupErrs <- err
				} else {
					rulesSetupErrs <- fmt.Errorf("rule setup: %s", err)
				}

				return
			}

			// adding rule for initiator must result with a route ID
			if idx == 0 {
				resultingRouteIDCh <- routeID
			}

			rulesSetupErrs <- nil
		}(idx, r[idx].To, rule, routeIDChIn, routeIDChOut)
	}

	var rulesSetupErr error
	var cancelOnce sync.Once
	// check for any errors occurred so far
	for range r {
		// filter out context cancellation errors
		if err := <-rulesSetupErrs; err != nil && err != context.Canceled {
			// rules setup failed, cancel further setup
			cancelOnce.Do(cancel)
			rulesSetupErr = err
		}
	}

	// close chan to avoid leaks
	close(rulesSetupErrs)
	for _, ch := range routeIDsCh {
		close(ch)
	}
	if rulesSetupErr != nil {
		return 0, rulesSetupErr
	}

	routeID := <-resultingRouteIDCh
	close(resultingRouteIDCh)

	return routeID, nil
}

func (sn *Node) connectLoop(ctx context.Context, on cipher.PubKey, ld routing.LoopData) error {
	proto, err := sn.dialAndCreateProto(ctx, on)
	if err != nil {
		return err
	}
	defer sn.closeProto(proto)

	if err := ConfirmLoop(ctx, proto, ld); err != nil {
		return err
	}

	sn.Logger.Infof("Confirmed loop on %s with %s. RemotePort: %d. LocalPort: %d", on, ld.Loop.Remote.PubKey, ld.Loop.Remote.Port, ld.Loop.Local.Port)
	return nil
}

// Close closes underlying dmsg client.
func (sn *Node) Close() error {
	if sn == nil {
		return nil
	}
	return sn.messenger.Close()
}

func (sn *Node) closeLoop(ctx context.Context, on cipher.PubKey, ld routing.LoopData) error {
	proto, err := sn.dialAndCreateProto(ctx, on)
	if err != nil {
		return err
	}
	defer sn.closeProto(proto)

	if err := LoopClosed(ctx, proto, ld); err != nil {
		return err
	}

	sn.Logger.Infof("Closed loop on %s. LocalPort: %d", on, ld.Loop.Local.Port)
	return nil
}

func (sn *Node) addRule(ctx context.Context, pubKey cipher.PubKey, rule routing.Rule,
	routeIDChIn <-chan routing.RouteID, routeIDChOut chan<- routing.RouteID) (routing.RouteID, error) {
	proto, err := sn.dialAndCreateProto(ctx, pubKey)
	if err != nil {
		return 0, err
	}
	defer sn.closeProto(proto)

	registrationID, err := RequestRouteID(ctx, proto)
	if err != nil {
		return 0, err
	}

	sn.Logger.Infof("Received route ID %d from %s", registrationID, pubKey)

	if routeIDChOut != nil {
		routeIDChOut <- registrationID
	}
	var nextRouteID routing.RouteID
	if routeIDChIn != nil {
		nextRouteID = <-routeIDChIn
		rule.SetRouteID(nextRouteID)
	}

	rule.SetRegistrationID(registrationID)

	sn.Logger.Debugf("dialing to %s to setup rule: %v\n", pubKey, rule)

	if err := AddRule(ctx, proto, rule); err != nil {
		return 0, err
	}

	sn.Logger.Infof("Set rule of type %s on %s", rule.Type(), pubKey)

	return registrationID, nil
}

func (sn *Node) dialAndCreateProto(ctx context.Context, pubKey cipher.PubKey) (*Protocol, error) {
	sn.Logger.Debugf("dialing to %s\n", pubKey)
	tr, err := sn.messenger.Dial(ctx, pubKey)
	if err != nil {
		return nil, fmt.Errorf("transport: %s", err)
	}

	return NewSetupProtocol(tr), nil
}

func (sn *Node) closeProto(proto *Protocol) {
	if err := proto.Close(); err != nil {
		sn.Logger.Warn(err)
	}
}
