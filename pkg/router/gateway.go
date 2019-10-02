package router

import (
	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/setup"
)

type Gateway struct {
	logger *logging.Logger
	router *router // TODO(nkryuchkov): move part of Router methods to Gateway
}

func NewGateway(router *router) *Gateway {
	return &Gateway{
		logger: logging.MustGetLogger("router-gateway"),
		router: router,
	}
}

func (r *Gateway) AddEdgeRules(rules routing.EdgeRules, ok *bool) error {
	go func() {
		r.router.accept <- rules
	}()

	if err := r.router.saveRoutingRules(rules.Forward, rules.Reverse); err != nil {
		*ok = false
		r.logger.WithError(err).Warnf("Request completed with error.")
		return setup.Failure{Code: setup.FailureAddRules, Msg: err.Error()}
	}

	*ok = true
	return nil
}

func (r *Gateway) AddIntermediaryRules(rules []routing.Rule, ok *bool) error {
	if err := r.router.saveRoutingRules(rules...); err != nil {
		*ok = false
		r.logger.WithError(err).Warnf("Request completed with error.")
		return setup.Failure{Code: setup.FailureAddRules, Msg: err.Error()}
	}

	*ok = true
	return nil
}

func (r *Gateway) ReserveIDs(n uint8, routeIDs *[]routing.RouteID) error {
	ids, err := r.router.occupyRouteID(n)
	if err != nil {
		r.logger.WithError(err).Warnf("Request completed with error.")
		return setup.Failure{Code: setup.FailureReserveRtIDs, Msg: err.Error()}
	}

	*routeIDs = ids
	return nil
}
