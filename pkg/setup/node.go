package setup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/skycoin/skywire/pkg/snet"

	"github.com/skycoin/dmsg"

	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/dmsg/disc"
	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/metrics"
	"github.com/skycoin/skywire/pkg/routing"
)

// Node performs routes setup operations over messaging channel.
type Node struct {
	Logger   *logging.Logger
	dmsgC    *dmsg.Client
	dmsgL    *dmsg.Listener
	srvCount int
	metrics  metrics.Recorder
}

// NewNode constructs a new SetupNode.
func NewNode(conf *Config, metrics metrics.Recorder) (*Node, error) {
	ctx := context.Background()

	logger := logging.NewMasterLogger()
	if lvl, err := logging.LevelFromString(conf.LogLevel); err == nil {
		logger.SetLevel(lvl)
	}
	log := logger.PackageLogger("setup_node")

	// Prepare dmsg.
	dmsgC := dmsg.NewClient(
		conf.PubKey,
		conf.SecKey,
		disc.NewHTTP(conf.Messaging.Discovery),
		dmsg.SetLogger(logger.PackageLogger(dmsg.Type)),
	)
	if err := dmsgC.InitiateServerConnections(ctx, conf.Messaging.ServerCount); err != nil {
		return nil, fmt.Errorf("failed to init dmsg: %s", err)
	}
	log.Info("connected to dmsg servers")

	dmsgL, err := dmsgC.Listen(snet.SetupPort)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on dmsg port %d: %v", snet.SetupPort, dmsgL)
	}
	log.Info("started listening for dmsg connections")

	return &Node{
		Logger:   log,
		dmsgC:    dmsgC,
		dmsgL:    dmsgL,
		srvCount: conf.Messaging.ServerCount,
		metrics:  metrics,
	}, nil
}

// Serve starts transport listening loop.
func (sn *Node) Serve(ctx context.Context) error {
	if err := sn.dmsgC.InitiateServerConnections(ctx, sn.srvCount); err != nil {
		return fmt.Errorf("messaging: %s", err)
	}
	sn.Logger.Info("Connected to messaging servers")

	sn.Logger.Info("Starting Setup Node")

	for {
		conn, err := sn.dmsgL.AcceptTransport()
		if err != nil {
			return err
		}
		go func(conn *dmsg.Transport) {
			if err := sn.serveTransport(ctx, conn); err != nil {
				sn.Logger.Warnf("Failed to serve Transport: %s", err)
			}
		}(conn)
	}
}

func (sn *Node) serveTransport(ctx context.Context, tr *dmsg.Transport) error {
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
	// regIDsCh is an array of chans used to pass the requested registration IDs around the goroutines.
	// We do it in a fan fashion here. We create as many goroutines as there are rules to be applied.
	// Goroutine[idx] requests visor node for a registration ID. It passes this registration ID through a chan to
	// a goroutine[idx-1]. In turn, goroutine[idx-1] waits for a registration ID from chan[idx].
	// Thus, goroutine[len(r)] doesn't get a registration ID and uses 0 instead, goroutine[0] doesn't pass
	// its route ID to anyone
	regIDsCh := make([]chan routing.RouteID, 0, len(r))
	for range r {
		regIDsCh = append(regIDsCh, make(chan routing.RouteID, 2))
	}

	// chan to receive the resulting route ID from a goroutine
	resultingRouteIDCh := make(chan routing.RouteID, 2)

	// context to cancel rule setup in case of errors
	ctx, cancel := context.WithCancel(context.Background())
	for idx := len(r) - 1; idx >= 0; idx-- {
		var regIDChIn, regIDChOut chan routing.RouteID
		if idx > 0 {
			regIDChOut = regIDsCh[idx-1]
		}
		var nextTransport uuid.UUID
		var rule routing.Rule
		if idx != len(r)-1 {
			regIDChIn = regIDsCh[idx]
			nextTransport = r[idx+1].Transport
			rule = routing.ForwardRule(expireAt, 0, nextTransport, 0)
		} else {
			rule = routing.AppRule(expireAt, 0, initiator, lport, rport, 0)
		}

		go func(idx int, pubKey cipher.PubKey, rule routing.Rule, regIDChIn <-chan routing.RouteID,
			regIDChOut chan<- routing.RouteID) {
			routeID, err := sn.setupRule(ctx, pubKey, rule, regIDChIn, regIDChOut)
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
		}(idx, r[idx].To, rule, regIDChIn, regIDChOut)
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
	for _, ch := range regIDsCh {
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
	return sn.dmsgC.Close()
}

func (sn *Node) closeLoop(ctx context.Context, on cipher.PubKey, ld routing.LoopData) error {
	fmt.Printf(">>> BEGIN: closeLoop(%s, ld)\n", on)
	defer fmt.Printf(">>>   END: closeLoop(%s, ld)\n", on)

	proto, err := sn.dialAndCreateProto(ctx, on)
	fmt.Println(">>> *****: closeLoop() dialed:", err)
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

func (sn *Node) setupRule(ctx context.Context, pubKey cipher.PubKey, rule routing.Rule,
	regIDChIn <-chan routing.RouteID, regIDChOut chan<- routing.RouteID) (routing.RouteID, error) {
	sn.Logger.Debugf("trying to setup setup rule: %v with %s\n", rule, pubKey)
	proto, err := sn.dialAndCreateProto(ctx, pubKey)
	if err != nil {
		return 0, err
	}
	defer sn.closeProto(proto)

	registrationID, err := RequestRegistrationID(ctx, proto)
	if err != nil {
		return 0, err
	}

	sn.Logger.Infof("Received route ID %d from %s", registrationID, pubKey)

	if regIDChOut != nil {
		regIDChOut <- registrationID
	}
	var nextRouteID routing.RouteID
	if regIDChIn != nil {
		nextRouteID = <-regIDChIn
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
	tr, err := sn.dmsgC.Dial(ctx, pubKey, snet.AwaitSetupPort)
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
