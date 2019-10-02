package setup

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/router/routerclient"
	"github.com/skycoin/skywire/pkg/routing"
)

type Gateway struct {
	logger *logging.Logger
	reqPK  cipher.PubKey
	sn     *Node
}

func NewGateway(reqPK cipher.PubKey, sn *Node) *Gateway {
	return &Gateway{
		logger: logging.MustGetLogger("setup-gateway"),
		reqPK:  reqPK,
		sn:     sn,
	}
}

func (g *Gateway) DialRouteGroup(route routing.BidirectionalRoute, rules *routing.EdgeRules) (failure error) {
	startTime := time.Now()
	defer func() {
		g.sn.metrics.Record(time.Since(startTime), failure != nil)
	}()

	g.logger.Infof("Received RPC DialRouteGroup request")

	ctx := context.Background()
	idr, err := g.reserveRouteIDs(ctx, route)
	if err != nil {
		return err
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
		return err
	}
	g.logger.Infof("generated forward rules: %v", forwardRules)
	g.logger.Infof("generated consume rules: %v", consumeRules)
	g.logger.Infof("generated intermediary rules: %v", intermediaryRules)

	errCh := make(chan error, len(intermediaryRules))
	var wg sync.WaitGroup

	for pk, rules := range intermediaryRules {
		wg.Add(1)
		pk, rules := pk, rules
		go func() {
			defer wg.Done()
			if _, err := routerclient.AddIntermediaryRules(ctx, g.logger, g.sn.dmsgC, pk, rules); err != nil {
				g.logger.WithField("remote", pk).WithError(err).Warn("failed to add rules")
				errCh <- err
			}
		}()
	}

	wg.Wait()
	close(errCh)

	if err := finalError(len(intermediaryRules), errCh); err != nil {
		return err
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
	ok, err := routerclient.AddEdgeRules(ctx, g.logger, g.sn.dmsgC, route.Desc.DstPK(), respRouteRules)
	if err != nil || !ok {
		return fmt.Errorf("failed to confirm loop with destination visor: %v", err)
	}

	// Confirm routes with initiating visor.
	*rules = initRouteRules
	return nil
}

func (g *Gateway) reserveRouteIDs(ctx context.Context, route routing.BidirectionalRoute) (*idReservoir, error) {
	reservoir, total := newIDReservoir(route.Forward, route.Reverse)
	g.logger.Infof("There are %d route IDs to reserve.", total)

	err := reservoir.ReserveIDs(ctx, g.logger, g.sn.dmsgC, routerclient.ReserveIDs)
	if err != nil {
		g.logger.WithError(err).Warnf("Failed to reserve route IDs.")
		return nil, err
	}
	g.logger.Infof("Successfully reserved route IDs.")
	return reservoir, err
}
