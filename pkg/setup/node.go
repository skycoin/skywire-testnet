package setup

import (
	"context"
	"fmt"
	"net/rpc"
	"sync"

	"github.com/skycoin/dmsg"
	"github.com/skycoin/dmsg/disc"
	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/metrics"
	"github.com/skycoin/skywire/pkg/router/routerclient"
	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/snet"
)

// Node performs routes setup operations over messaging channel.
type Node struct {
	logger   *logging.Logger
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

	node := &Node{
		logger:   log,
		dmsgC:    dmsgC,
		dmsgL:    dmsgL,
		srvCount: conf.Messaging.ServerCount,
		metrics:  metrics,
	}

	return node, nil
}

// Close closes underlying dmsg client.
func (sn *Node) Close() error {
	if sn == nil {
		return nil
	}

	return sn.dmsgC.Close()
}

// Serve starts transport listening loop.
func (sn *Node) Serve() error {
	sn.logger.Info("Serving setup node")

	for {
		conn, err := sn.dmsgL.AcceptTransport()
		if err != nil {
			return err
		}

		sn.logger.WithField("requester", conn.RemotePK()).Infof("Received request.")

		rpcS := rpc.NewServer()
		if err := rpcS.Register(NewRPCGateway(conn.RemotePK(), sn)); err != nil {
			return err
		}
		go rpcS.ServeConn(conn)
	}
}

func (sn *Node) handleDialRouteGroup(ctx context.Context, route routing.BidirectionalRoute) (routing.EdgeRules, error) {
	idr, err := sn.reserveRouteIDs(ctx, route)
	if err != nil {
		return routing.EdgeRules{}, err
	}

	forwardRoute := routing.Route{
		Desc:      route.Desc,
		Path:      route.Forward,
		KeepAlive: route.KeepAlive,
	}
	reverseRoute := routing.Route{
		Desc:      route.Desc.Invert(),
		Path:      route.Reverse,
		KeepAlive: route.KeepAlive,
	}

	// Determine the rules to send to visors using loop descriptor and reserved route IDs.
	forwardRules, consumeRules, intermediaryRules, err := idr.GenerateRules(forwardRoute, reverseRoute)
	if err != nil {
		return routing.EdgeRules{}, err
	}

	sn.logger.Infof("generated forward rules: %v", forwardRules)
	sn.logger.Infof("generated consume rules: %v", consumeRules)
	sn.logger.Infof("generated intermediary rules: %v", intermediaryRules)

	errCh := make(chan error, len(intermediaryRules))
	var wg sync.WaitGroup
	for pk, rules := range intermediaryRules {
		wg.Add(1)
		pk, rules := pk, rules
		go func() {
			defer wg.Done()
			if _, err := routerclient.AddIntermediaryRules(ctx, sn.logger, sn.dmsgC, pk, rules); err != nil {
				sn.logger.WithField("remote", pk).WithError(err).Warn("failed to add rules")
				errCh <- err
			}
		}()
	}

	wg.Wait()
	close(errCh)

	if err := finalError(len(intermediaryRules), errCh); err != nil {
		return routing.EdgeRules{}, err
	}

	initRouteRules := routing.EdgeRules{
		Desc:    forwardRoute.Desc,
		Forward: forwardRules[route.Desc.SrcPK()],
		Reverse: consumeRules[route.Desc.SrcPK()],
	}
	respRouteRules := routing.EdgeRules{
		Desc:    reverseRoute.Desc,
		Forward: forwardRules[route.Desc.DstPK()],
		Reverse: consumeRules[route.Desc.DstPK()],
	}

	// Confirm routes with responding visor.
	ok, err := routerclient.AddEdgeRules(ctx, sn.logger, sn.dmsgC, route.Desc.DstPK(), respRouteRules)
	if err != nil || !ok {
		return routing.EdgeRules{}, fmt.Errorf("failed to confirm loop with destination visor: %v", err)
	}

	return initRouteRules, nil
}

func (sn *Node) reserveRouteIDs(ctx context.Context, route routing.BidirectionalRoute) (*idReservoir, error) {
	reservoir, total := newIDReservoir(route.Forward, route.Reverse)
	sn.logger.Infof("There are %d route IDs to reserve.", total)

	err := reservoir.ReserveIDs(ctx, sn.logger, sn.dmsgC, routerclient.ReserveIDs)
	if err != nil {
		sn.logger.WithError(err).Warnf("Failed to reserve route IDs.")
		return nil, err
	}

	sn.logger.Infof("Successfully reserved route IDs.")
	return reservoir, err
}
