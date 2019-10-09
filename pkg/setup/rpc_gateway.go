package setup

import (
	"context"
	"fmt"
	"time"

	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/routing"
)

type RPCGateway struct {
	logger *logging.Logger
	reqPK  cipher.PubKey
	sn     *Node
}

func NewRPCGateway(reqPK cipher.PubKey, sn *Node) *RPCGateway {
	return &RPCGateway{
		logger: logging.MustGetLogger(fmt.Sprintf("setup-gateway (%s)", reqPK)),
		reqPK:  reqPK,
		sn:     sn,
	}
}

func (g *RPCGateway) DialRouteGroup(route routing.BidirectionalRoute, rules *routing.EdgeRules) (err error) {
	startTime := time.Now()
	defer func() {
		g.sn.metrics.Record(time.Since(startTime), err != nil)
	}()

	g.logger.Infof("Received RPC DialRouteGroup request")

	// TODO(nkryuchkov): Is there a better way to do timeout?
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	initRules, err := g.sn.handleDialRouteGroup(ctx, route)
	if err != nil {
		return err
	}

	// Confirm routes with initiating visor.
	*rules = initRules
	return nil
}
