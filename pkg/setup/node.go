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

	fmt.Println("FINISHED SETUP OF REVERSE ROUTE")

	fRouteID, err := sn.createRoute(ctx, ld.Expiry, ld.Forward, ld.Loop.Remote.Port, ld.Loop.Local.Port)
	if err != nil {
		return err
	}

	fmt.Println("FINISHED SETUP OF FORWARD ROUTE")

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

	fmt.Println("FINISHED CONNECTLOOP REQUEST")

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
	r := make([]*Hop, len(route))

	initiator := route[0].From

	// indicate errors occurred during rules setup
	rulesSetupErrs := make(chan error, len(r))
	routeIDsCh := make([]chan routing.RouteID, 0, len(r))
	for range r {
		routeIDsCh = append(routeIDsCh, make(chan routing.RouteID, 2))
	}

	// context to cancel rule setup in case of errors
	ctx, cancel := context.WithCancel(context.Background())
	for idx := len(r) - 1; idx >= 0; idx-- {
		hop := &Hop{Hop: route[idx]}
		r[idx] = hop

		toPK := hop.To.Hex()

		var routeIDChIn, routeIDChOut chan routing.RouteID
		if idx > 0 {
			routeIDChOut = routeIDsCh[idx-1]
		}
		var nextTransport uuid.UUID
		if idx != len(r)-1 {
			routeIDChIn = routeIDsCh[idx]
			nextTransport = route[idx+1].Transport
		}

		go func(idx int, hop *Hop, routeIDChIn, routeIDChOut chan routing.RouteID, nextTransport uuid.UUID) {
			sn.Logger.Debugf("dialing to %s to request route ID\n", toPK)
			tr, err := sn.messenger.Dial(ctx, hop.To)
			if err != nil {
				// filter out context cancellation errors
				if err == context.Canceled {
					rulesSetupErrs <- err
				} else {
					rulesSetupErrs <- fmt.Errorf("rule setup: transport: %s", err)
				}
				return
			}
			defer func() {
				if err := tr.Close(); err != nil {
					sn.Logger.Warnf("Failed to close transport: %s", err)
				}
			}()

			proto := NewSetupProtocol(tr)
			routeID, err := RequestRouteID(ctx, proto)
			if err != nil {
				// filter out context cancellation errors
				if err == context.Canceled {
					rulesSetupErrs <- err
				} else {
					rulesSetupErrs <- fmt.Errorf("rule setup: %s", err)
				}
				return
			}

			sn.Logger.Infof("Received route ID %d from %s", routeID, hop.To)

			hop.routeID = routeID

			if routeIDChOut != nil {
				routeIDChOut <- routeID
			}
			var nextRouteID routing.RouteID
			if routeIDChIn != nil {
				nextRouteID = <-routeIDChIn
			}

			var rule routing.Rule
			if idx == len(r)-1 {
				rule = routing.AppRule(expireAt, 0, initiator, lport, rport, hop.routeID)
			} else {
				rule = routing.ForwardRule(expireAt, nextRouteID, nextTransport, hop.routeID)
			}

			sn.Logger.Debugf("dialing to %s to setup rule: %v\n", hop.To, rule)

			if err := AddRule(ctx, proto, rule); err != nil {
				// filter out context cancellation errors
				if err == context.Canceled {
					rulesSetupErrs <- err
				} else {
					rulesSetupErrs <- fmt.Errorf("rule setup: %s", err)
				}
				return
			}

			sn.Logger.Infof("Set rule of type %s on %s", rule.Type(), hop.To)

			// put nil to avoid block
			rulesSetupErrs <- nil
		}(idx, hop, routeIDChIn, routeIDChOut, nextTransport)
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

	sn.Logger.Debugf("dialing to %s to request route ID\n", initiator)
	tr, err := sn.messenger.Dial(ctx, initiator)
	if err != nil {
		return 0, fmt.Errorf("transport: %s", err)
	}
	defer func() {
		if err := tr.Close(); err != nil {
			sn.Logger.Warnf("Failed to close transport: %s", err)
		}
	}()

	proto := NewSetupProtocol(tr)
	routeID, err := RequestRouteID(ctx, proto)
	if err != nil {
		return 0, err
	}

	rule := routing.ForwardRule(expireAt, r[0].routeID, r[0].Transport, routeID)
	if err := AddRule(ctx, proto, rule); err != nil {
		return 0, fmt.Errorf("rule setup: %s", err)
	}

	return routeID, nil
}

func (sn *Node) connectLoop(ctx context.Context, on cipher.PubKey, ld routing.LoopData) error {
	tr, err := sn.messenger.Dial(ctx, on)
	if err != nil {
		return fmt.Errorf("transport: %s", err)
	}
	defer func() {
		if err := tr.Close(); err != nil {
			sn.Logger.Warnf("Failed to close transport: %s", err)
		}
	}()

	if err := ConfirmLoop(ctx, NewSetupProtocol(tr), ld); err != nil {
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
	fmt.Printf(">>> BEGIN: closeLoop(%s, ld)\n", on)
	defer fmt.Printf(">>>   END: closeLoop(%s, ld)\n", on)

	tr, err := sn.messenger.Dial(ctx, on)
	fmt.Println(">>> *****: closeLoop() dialed:", err)
	if err != nil {
		return fmt.Errorf("transport: %s", err)
	}
	defer func() {
		if err := tr.Close(); err != nil {
			sn.Logger.Warnf("Failed to close transport: %s", err)
		}
	}()

	proto := NewSetupProtocol(tr)
	if err := LoopClosed(ctx, proto, ld); err != nil {
		return err
	}

	sn.Logger.Infof("Closed loop on %s. LocalPort: %d", on, ld.Loop.Local.Port)
	return nil
}

func (sn *Node) requestRouteID(ctx context.Context, pubKey cipher.PubKey) (routing.RouteID, error) {
	sn.Logger.Debugf("dialing to %s to request route ID\n", pubKey)
	tr, err := sn.messenger.Dial(ctx, pubKey)
	if err != nil {
		return 0, fmt.Errorf("transport: %s", err)
	}
	defer func() {
		if err := tr.Close(); err != nil {
			sn.Logger.Warnf("Failed to close transport: %s", err)
		}
	}()

	proto := NewSetupProtocol(tr)
	routeID, err := RequestRouteID(ctx, proto)
	if err != nil {
		return 0, err
	}

	sn.Logger.Infof("Received route ID %d from %s", routeID, pubKey)
	return routeID, nil
}

func (sn *Node) setupRule(ctx context.Context, pubKey cipher.PubKey, rule routing.Rule) error {
	sn.Logger.Debugf("dialing to %s to setup rule: %v\n", pubKey, rule)
	tr, err := sn.messenger.Dial(ctx, pubKey)
	if err != nil {
		return fmt.Errorf("transport: %s", err)
	}
	defer func() {
		if err := tr.Close(); err != nil {
			sn.Logger.Warnf("Failed to close transport: %s", err)
		}
	}()

	proto := NewSetupProtocol(tr)
	if err := AddRule(ctx, proto, rule); err != nil {
		return err
	}

	sn.Logger.Infof("Set rule of type %s on %s", rule.Type(), pubKey)
	return nil
}
