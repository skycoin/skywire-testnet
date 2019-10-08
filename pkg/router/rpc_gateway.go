package router

import (
	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/setup"
)

type RPCGateway struct {
	logger *logging.Logger
	router *router // TODO(nkryuchkov): move part of Router methods to RPCGateway
}

func NewRPCGateway(router *router) *RPCGateway {
	return &RPCGateway{
		logger: logging.MustGetLogger("router-gateway"),
		router: router,
	}
}

func (r *RPCGateway) AddEdgeRules(rules routing.EdgeRules, ok *bool) error {
	if err := r.router.IntroduceRules(rules); err != nil {
		return err
	}

	if err := r.router.saveRoutingRules(rules.Forward, rules.Reverse); err != nil {
		*ok = false
		r.logger.WithError(err).Warnf("Request completed with error.")
		return setup.Failure{Code: setup.FailureAddRules, Msg: err.Error()}
	}

	*ok = true
	return nil
}

func (r *RPCGateway) AddIntermediaryRules(rules []routing.Rule, ok *bool) error {
	if err := r.router.saveRoutingRules(rules...); err != nil {
		*ok = false
		r.logger.WithError(err).Warnf("Request completed with error.")
		return setup.Failure{Code: setup.FailureAddRules, Msg: err.Error()}
	}

	*ok = true
	return nil
}

func (r *RPCGateway) ReserveIDs(n uint8, routeIDs *[]routing.RouteID) error {
	ids, err := r.router.rt.ReserveKeys(int(n))
	if err != nil {
		r.logger.WithError(err).Warnf("Request completed with error.")
		return setup.Failure{Code: setup.FailureReserveRtIDs, Msg: err.Error()}
	}

	*routeIDs = ids
	return nil
}
