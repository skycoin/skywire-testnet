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

// createRoute setups the route. Route setup involves applying routing rules to each visor node along the route.
// Each rule applying procedure consists of two steps:
// 1. Request free route ID from the visor node
// 2. Apply the rule, using route ID from the step 1 to register this rule inside the visor node
//
// Route ID received as a response after 1st step is used in two rules. 1st, it's used in the rule being applied
// to the current visor node as a route ID to register this rule within the visor node.
// 2nd, it's used in the rule being applied to the previous visor node along the route as a `respRouteID/nextRouteID`.
// For this reason, each 2nd step must wait for completion of its 1st step and the 1st step of the next visor node
// along the route to be able to obtain route ID from there. IDs serving as `respRouteID/nextRouteID` are being
// passed in a fan-like fashion.
//
// Example. Let's say, we have N visor nodes along the route. Visor[0] is the initiator. Setup node sends N requests to
// each visor along the route according to the 1st step and gets N route IDs in response. Then we assemble N rules to
// be applied. We construct each rule as the following:
// - Rule[0..N-1] are of type `ForwardRule`;
// - Rule[N] is of type `AppRule`;
// - For i = 0..N-1 rule[i] takes `nextTransportID` from the rule[i+1];
// - For i = 0..N-1 rule[i] takes `respRouteID/nextRouteID` from rule[i+1] (after [i+1] request for free route ID
// completes;
// - Rule[N] has `respRouteID/nextRouteID` equal to 0;
// Rule[0..N] use their route ID retrieved from the 1st step to be registered within the corresponding visor node.
//
// During the setup process each error received along the way causes all the procedure to be cancelled. RouteID received
// from the 1st step connecting to the initiating node is used as the ID for the overall rule, thus being returned.
func (sn *Node) createRoute(ctx context.Context, expireAt time.Time, route routing.Route, rport, lport routing.Port) (routing.RouteID, error) {
	if len(route) == 0 {
		return 0, nil
	}

	sn.Logger.Infof("Creating new Route %s", route)

	// add the initiating node to the start of the route. We need to loop over all the visor nodes
	// along the route to apply rules including the initiating one
	r := make(routing.Route, len(route)+1)
	r[0] = &routing.Hop{
		Transport: route[0].Transport,
		To:        route[0].From,
	}
	copy(r[1:], route)

	init := route[0].From

	// indicate errors occurred during rules setup
	rulesSetupErrs := make(chan error, len(r))
	// reqIDsCh is an array of chans used to pass the requested route IDs around the goroutines.
	// We do it in a fan fashion here. We create as many goroutines as there are rules to be applied.
	// Goroutine[i] requests visor node for a free route ID. It passes this route ID through a chan to
	// a goroutine[i-1]. In turn, goroutine[i-1] waits for a route ID from chan[i].
	// Thus, goroutine[len(r)] doesn't get a route ID and uses 0 instead, goroutine[0] doesn't pass
	// its route ID to anyone
	reqIDsCh := make([]chan routing.RouteID, 0, len(r))
	for range r {
		reqIDsCh = append(reqIDsCh, make(chan routing.RouteID, 2))
	}

	// chan to receive the resulting route ID from a goroutine
	resultingRouteIDCh := make(chan routing.RouteID, 2)

	// context to cancel rule setup in case of errors
	cancellableCtx, cancel := context.WithCancel(ctx)
	for i := len(r) - 1; i >= 0; i-- {
		var reqIDChIn, reqIDChOut chan routing.RouteID
		// goroutine[0] doesn't need to pass the route ID from the 1st step to anyone
		if i > 0 {
			reqIDChOut = reqIDsCh[i-1]
		}
		var (
			nextTpID uuid.UUID
			rule     routing.Rule
		)
		// goroutine[len(r)-1] uses 0 as the route ID from the 1st step
		if i != len(r)-1 {
			reqIDChIn = reqIDsCh[i]
			nextTpID = r[i+1].Transport
			rule = routing.ForwardRule(expireAt, 0, nextTpID, 0)
		} else {
			rule = routing.AppRule(expireAt, 0, init, lport, rport, 0)
		}

		go func(i int, pk cipher.PubKey, rule routing.Rule, reqIDChIn <-chan routing.RouteID,
			reqIDChOut chan<- routing.RouteID) {
			routeID, err := sn.setupRule(cancellableCtx, pk, rule, reqIDChIn, reqIDChOut)
			if err != nil {
				// filter out context cancellation errors
				if err == context.Canceled {
					rulesSetupErrs <- err
				} else {
					rulesSetupErrs <- fmt.Errorf("rule setup: %s", err)
				}

				return
			}

			// adding rule for initiator must result with a route ID for the overall route
			if i == 0 {
				resultingRouteIDCh <- routeID
			}

			rulesSetupErrs <- nil
		}(i, r[i].To, rule, reqIDChIn, reqIDChOut)
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
	for _, ch := range reqIDsCh {
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

func (sn *Node) setupRule(ctx context.Context, pk cipher.PubKey, rule routing.Rule,
	reqIDChIn <-chan routing.RouteID, reqIDChOut chan<- routing.RouteID) (routing.RouteID, error) {
	sn.Logger.Debugf("trying to setup setup rule: %v with %s\n", rule, pk)
	requestRouteID, err := sn.requestRouteID(ctx, pk)
	if err != nil {
		return 0, err
	}

	if reqIDChOut != nil {
		reqIDChOut <- requestRouteID
	}
	var nextRouteID routing.RouteID
	if reqIDChIn != nil {
		nextRouteID = <-reqIDChIn
		rule.SetRouteID(nextRouteID)
	}

	rule.SetRequestRouteID(requestRouteID)

	sn.Logger.Debugf("dialing to %s to setup rule: %v\n", pk, rule)

	if err := sn.addRule(ctx, pk, rule); err != nil {
		return 0, err
	}

	sn.Logger.Infof("Set rule of type %s on %s", rule.Type(), pk)

	return requestRouteID, nil
}

func (sn *Node) requestRouteID(ctx context.Context, pk cipher.PubKey) (routing.RouteID, error) {
	proto, err := sn.dialAndCreateProto(ctx, pk)
	if err != nil {
		return 0, err
	}
	defer sn.closeProto(proto)

	requestRouteID, err := RequestRouteID(ctx, proto)
	if err != nil {
		return 0, err
	}

	sn.Logger.Infof("Received route ID %d from %s", requestRouteID, pk)

	return requestRouteID, nil
}

func (sn *Node) addRule(ctx context.Context, pk cipher.PubKey, rule routing.Rule) error {
	proto, err := sn.dialAndCreateProto(ctx, pk)
	if err != nil {
		return err
	}
	defer sn.closeProto(proto)

	return AddRule(ctx, proto, rule)
}

func (sn *Node) dialAndCreateProto(ctx context.Context, pk cipher.PubKey) (*Protocol, error) {
	sn.Logger.Debugf("dialing to %s\n", pk)
	tr, err := sn.dmsgC.Dial(ctx, pk, snet.AwaitSetupPort)
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
