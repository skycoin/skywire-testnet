package router

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/app"

	"github.com/skycoin/skywire/pkg/routing"
)

const (
	// DefaultRouteKeepalive is the default interval to keep active expired routes.
	DefaultRouteKeepalive = 10 * time.Minute

	// DefaultRouteCleanupDuration is the default interval to clean up routes.
	DefaultRouteCleanupDuration = 10 * time.Minute
)

// RoutingTableManager manages an internal routing table by cleaning up expired rules per time interval
// and by providing some additional rule-finding functions.
type RoutingTableManager struct {
	routing.Table
	log       *logging.Logger
	keepalive time.Duration
	ticker    *time.Ticker // stale routing rules are deleted when triggered
	activity  map[routing.RouteID]time.Time
	mx        sync.Mutex
}

// NewRoutingTableManager creates a new RoutingTableManager.
func NewRoutingTableManager(l *logging.Logger, rt routing.Table, keepalive, cleanup time.Duration) *RoutingTableManager {
	return &RoutingTableManager{
		Table:     rt,
		log:       l,
		keepalive: keepalive,
		ticker:    time.NewTicker(cleanup),
		activity:  make(map[routing.RouteID]time.Time),
	}
}

// Run runs the table-cleanup event loop.
func (rtm *RoutingTableManager) Run() {
	for range rtm.ticker.C {
		if err := rtm.Cleanup(); err != nil {
			rtm.log.Warnf("Failed to expiry routes: %s", err)
		}
	}
}

// Stop stops the table-cleanup event loop.
func (rtm *RoutingTableManager) Stop() {
	rtm.ticker.Stop()
}

// Rule obtains a routing rule of given rtID key.
func (rtm *RoutingTableManager) Rule(routeID routing.RouteID) (routing.Rule, error) {
	rtm.mx.Lock()
	rtm.activity[routeID] = time.Now()
	rtm.mx.Unlock()

	rule, err := rtm.Table.Rule(routeID)
	if err != nil {
		return nil, fmt.Errorf("routing table: %s", err)
	}
	if rule == nil {
		return nil, errors.New("unknown RouteID")
	}
	if rule.Expiry().Before(time.Now()) {
		return nil, errors.New("expired routing rule")
	}
	return rule, nil
}

// DeleteAppRule deletes a rule of type APP with given LoopMeta.
func (rtm *RoutingTableManager) DeleteAppRule(lm app.LoopMeta) error {
	rtID, _, ok := rtm.FindAppRule(lm)
	if !ok {
		return nil
	}
	if err := rtm.DeleteRules(rtID); err != nil {
		return fmt.Errorf("routing table: %s", err)
	}
	return nil
}

// FindAppRule finds an APP rule with given LoopMeta.
func (rtm *RoutingTableManager) FindAppRule(lm app.LoopMeta) (rtID routing.RouteID, rule routing.Rule, ok bool) {
	_ = rtm.RangeRules(func(id routing.RouteID, r routing.Rule) bool { //nolint:errcheck
		var (
			typesMatch  = r.Type() == routing.RuleApp
			rPKsMatch   = r.RemotePK() == lm.Remote.PubKey
			rPortsMatch = r.RemotePort() == lm.Remote.Port
			lPortsMatch = r.LocalPort() == lm.Local.Port
		)
		if typesMatch && rPKsMatch && rPortsMatch && lPortsMatch {
			rtID, rule, ok = id, r, true
			return false
		}
		return true
	})
	return
}

// FindFwdRule finds a FWD rule with given rtID key.
func (rtm *RoutingTableManager) FindFwdRule(rtID routing.RouteID) (routing.Rule, error) {
	rule, err := rtm.Rule(rtID)
	if err != nil {
		return nil, fmt.Errorf("routing table: %s", err)
	}
	if rule.Type() != routing.RuleForward {
		return nil, errors.New("reverse rule is not forward")
	}
	return rule, nil
}

// Cleanup removes all expired routing rules.
func (rtm *RoutingTableManager) Cleanup() error {
	var expiredIDs []routing.RouteID
	rtm.mx.Lock()
	err := rtm.RangeRules(func(routeID routing.RouteID, rule routing.Rule) bool {
		if rule.Expiry().Before(time.Now()) {
			if lastActivity, ok := rtm.activity[routeID]; !ok || time.Since(lastActivity) > rtm.keepalive {
				expiredIDs = append(expiredIDs, routeID)
			}
		}
		return true
	})
	rtm.mx.Unlock()

	if err != nil {
		return err
	}
	return rtm.DeleteRules(expiredIDs...)
}
