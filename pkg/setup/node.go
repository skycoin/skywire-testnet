package setup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/skycoin/skywire/pkg/metrics"

	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/cipher"
	"github.com/skycoin/skywire/pkg/messaging"
	mClient "github.com/skycoin/skywire/pkg/messaging-discovery/client"
	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/transport"
	trClient "github.com/skycoin/skywire/pkg/transport-discovery/client"
)

// Hop is a wrapper around transport hop to add functionality
type Hop struct {
	*routing.Hop
	routeID routing.RouteID
}

// Node performs routes setup operations over messaging channel.
type Node struct {
	Logger *logging.Logger

	tm        *transport.Manager
	messenger *messaging.Client
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
	messenger := messaging.NewClient(&messaging.Config{
		PubKey:     pk,
		SecKey:     sk,
		Discovery:  mClient.NewHTTP(conf.Messaging.Discovery),
		Retries:    10,
		RetryDelay: time.Second,
	})
	messenger.Logger = logger.PackageLogger("messenger")

	trDiscovery, err := trClient.NewHTTP(conf.TransportDiscovery, pk, sk)
	if err != nil {
		return nil, fmt.Errorf("trdiscovery: %s", err)
	}

	tmConf := &transport.ManagerConfig{
		PubKey:          pk,
		SecKey:          sk,
		DiscoveryClient: trDiscovery,
		LogStore:        transport.InMemoryTransportLogStore(),
	}

	tm, err := transport.NewManager(tmConf, messenger)
	if err != nil {
		log.Fatal("Failed to setup Transport Manager: ", err)
	}
	tm.Logger = logger.PackageLogger("trmanager")

	return &Node{
		Logger:    logger.PackageLogger("routesetup"),
		metrics:   metrics,
		tm:        tm,
		messenger: messenger,
		srvCount:  conf.Messaging.ServerCount,
	}, nil
}

// Serve starts transport listening loop.
func (sn *Node) Serve(ctx context.Context) error {
	if sn.srvCount > 0 {
		if err := sn.messenger.ConnectToInitialServers(ctx, sn.srvCount); err != nil {
			return fmt.Errorf("messaging: %s", err)
		}
		sn.Logger.Info("Connected to messaging servers")
	}

	acceptCh, dialCh := sn.tm.Observe()
	go func() {
		for tr := range acceptCh {
			go func(t transport.Transport) {
				for {
					if err := sn.serveTransport(t); err != nil {
						sn.Logger.Warnf("Failed to serve Transport: %s", err)
						return
					}
				}
			}(tr)
		}
	}()

	go func() {
		for range dialCh {
		}
	}()

	sn.Logger.Info("Starting Setup Node")
	return sn.tm.Serve(ctx)
}

func (sn *Node) createLoop(l *routing.Loop) error {
	sn.Logger.Infof("Creating new Loop %s", l)
	rRouteID, err := sn.createRoute(l.Expiry, l.Reverse, l.LocalPort, l.RemotePort)
	if err != nil {
		return err
	}

	fRouteID, err := sn.createRoute(l.Expiry, l.Forward, l.RemotePort, l.LocalPort)
	if err != nil {
		return err
	}

	if len(l.Forward) == 0 || len(l.Reverse) == 0 {
		return nil
	}

	initiator := l.Initiator()
	responder := l.Responder()

	ldR := &LoopData{RemotePK: initiator, RemotePort: l.LocalPort, LocalPort: l.RemotePort, RouteID: rRouteID, NoiseMessage: l.NoiseMessage}
	noiseRes, err := sn.connectLoop(responder, ldR)
	if err != nil {
		sn.Logger.Warnf("Failed to confirm loop with responder: %s", err)
		return fmt.Errorf("loop connect: %s", err)
	}

	ldI := &LoopData{RemotePK: responder, RemotePort: l.RemotePort, LocalPort: l.LocalPort, RouteID: fRouteID, NoiseMessage: noiseRes}
	if _, err := sn.connectLoop(initiator, ldI); err != nil {
		sn.Logger.Warnf("Failed to confirm loop with initiator: %s", err)
		if err := sn.closeLoop(responder, ldR); err != nil {
			sn.Logger.Warnf("Failed to close loop: %s", err)
		}

		return fmt.Errorf("loop connect: %s", err)
	}

	sn.Logger.Infof("Created Loop %s", l)
	return nil
}

func (sn *Node) createRoute(expireAt time.Time, route routing.Route, rport, lport uint16) (routing.RouteID, error) {
	if len(route) == 0 {
		return 0, nil
	}

	sn.Logger.Info("1")
	sn.Logger.Infof("Creating new Route %s", route)
	r := make([]*Hop, len(route))

	initiator := route[0].From
	for idx := len(r) - 1; idx >= 0; idx-- {
		hop := &Hop{Hop: route[idx]}
		r[idx] = hop
		var rule routing.Rule
		if idx == len(r)-1 {
			rule = routing.AppRule(expireAt, 0, initiator, lport, rport)
		} else {
			nextHop := r[idx+1]
			rule = routing.ForwardRule(expireAt, nextHop.routeID, nextHop.Transport)
		}

		routeID, err := sn.setupRule(hop.To, rule)
		if err != nil {
			return 0, fmt.Errorf("rule setup: %s", err)
		}

		hop.routeID = routeID
	}

	rule := routing.ForwardRule(expireAt, r[0].routeID, r[0].Transport)
	routeID, err := sn.setupRule(initiator, rule)
	if err != nil {
		return 0, fmt.Errorf("rule setup: %s", err)
	}

	return routeID, nil
}

// Close closes underlying transport manager.
func (sn *Node) Close() error {
	return sn.tm.Close()
}

func (sn *Node) serveTransport(tr transport.Transport) error {
	proto := NewSetupProtocol(tr)
	sp, data, err := proto.ReadPacket()
	if err != nil {
		return err
	}

	sn.Logger.Infof("Got new Setup request with type %s: %s", sp, string(data))

	startTime := time.Now()
	switch sp {
	case PacketCreateLoop:
		loop := &routing.Loop{}
		if err = json.Unmarshal(data, loop); err == nil {
			err = sn.createLoop(loop)
		}
	case PacketCloseLoop:
		ld := &LoopData{}
		if err = json.Unmarshal(data, ld); err == nil {
			remote, ok := sn.tm.Remote(tr.Edges())
			if !ok {
				return errors.New("configured PubKey not found in edges")
			}
			err = sn.closeLoop(ld.RemotePK, &LoopData{RemotePK: remote, RemotePort: ld.LocalPort, LocalPort: ld.RemotePort})
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

func (sn *Node) connectLoop(on cipher.PubKey, ld *LoopData) (noiseRes []byte, err error) {
	tr, err := sn.tm.CreateTransport(context.Background(), on, "messaging", false)
	if err != nil {
		err = fmt.Errorf("transport: %s", err)
		return
	}

	proto := NewSetupProtocol(tr)
	res, err := ConfirmLoop(proto, ld)
	if err != nil {
		return nil, err
	}

	sn.Logger.Infof("Confirmed loop on %s with %s. RemotePort: %d. LocalPort: %d", on, ld.RemotePK, ld.RemotePort, ld.LocalPort)
	err = sn.tm.DeleteTransport(tr.ID)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (sn *Node) closeLoop(on cipher.PubKey, ld *LoopData) error {
	tr, err := sn.tm.CreateTransport(context.Background(), on, "messaging", false)
	if err != nil {
		return fmt.Errorf("transport: %s", err)
	}

	proto := NewSetupProtocol(tr)
	if err := LoopClosed(proto, ld); err != nil {
		return err
	}

	sn.Logger.Infof("Closed loop on %s. LocalPort: %d", on, ld.LocalPort)
	err = sn.tm.DeleteTransport(tr.ID)
	if err != nil {
		return err
	}
	return nil
}

func (sn *Node) setupRule(pubKey cipher.PubKey, rule routing.Rule) (routeID routing.RouteID, err error) {
	defer fmt.Println("returning from setupRule")
	sn.Logger.Info("setupRule before CreateTransport...")
	tr, err := sn.tm.CreateTransport(context.Background(), pubKey, "messaging", false)
	if err != nil {
		err = fmt.Errorf("transport: %s", err)
		return
	}
	sn.Logger.Info("setup rule after CreateTransport")

	proto := NewSetupProtocol(tr)
	fmt.Println("AddRule")
	routeID, err = AddRule(proto, rule)
	if err != nil {
		return
	}
	fmt.Println("RuleAdded")

	sn.Logger.Infof("Set rule of type %s on %s with ID %d", rule.Type(), pubKey, routeID)
	err = sn.tm.DeleteTransport(tr.ID)
	if err != nil {
		return routeID, err
	}
	return routeID, nil
}
