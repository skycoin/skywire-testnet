package router

import (
	"errors"
	"sync"
	"time"

	"github.com/SkycoinProject/skywire-mainnet/pkg/routing"
)

var (
	// ErrRuleTimedOut is being returned while trying to access the rule which timed out
	ErrRuleTimedOut = errors.New("rule keep-alive timeout exceeded")
)

type managedRoutingTable struct {
	routing.Table

	activity map[routing.RouteID]time.Time
	mu       sync.Mutex
}

func manageRoutingTable(rt routing.Table) *managedRoutingTable {
	return &managedRoutingTable{
		Table:    rt,
		activity: make(map[routing.RouteID]time.Time),
	}
}

func (rt *managedRoutingTable) AddRule(rule routing.Rule) (routing.RouteID, error) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	routeID, err := rt.Table.AddRule(rule)
	if err != nil {
		return 0, err
	}

	// set the initial activity for rule not to be timed out instantly
	rt.activity[routeID] = time.Now()

	return routeID, nil
}

func (rt *managedRoutingTable) Rule(routeID routing.RouteID) (routing.Rule, error) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	rule, err := rt.Table.Rule(routeID)
	if err != nil {
		return nil, err
	}

	if rt.ruleIsTimedOut(routeID, rule) {
		return nil, ErrRuleTimedOut
	}

	rt.activity[routeID] = time.Now()

	return rule, nil
}

func (rt *managedRoutingTable) Cleanup() error {
	expiredIDs := make([]routing.RouteID, 0)
	rt.mu.Lock()
	defer rt.mu.Unlock()

	err := rt.RangeRules(func(routeID routing.RouteID, rule routing.Rule) bool {
		if rt.ruleIsTimedOut(routeID, rule) {
			expiredIDs = append(expiredIDs, routeID)
		}
		return true
	})
	if err != nil {
		return err
	}

	if err := rt.DeleteRules(expiredIDs...); err != nil {
		return err
	}

	rt.deleteActivity(expiredIDs...)

	return nil
}

// ruleIsExpired checks whether rule's keep alive timeout is exceeded.
// NOTE: for internal use, is NOT thread-safe, object lock should be acquired outside
func (rt *managedRoutingTable) ruleIsTimedOut(routeID routing.RouteID, rule routing.Rule) bool {
	lastActivity, ok := rt.activity[routeID]
	return !ok || time.Since(lastActivity) > rule.KeepAlive()
}

// deleteActivity removes activity records for the specified set of `routeIDs`.
// NOTE: for internal use, is NOT thread-safe, object lock should be acquired outside
func (rt *managedRoutingTable) deleteActivity(routeIDs ...routing.RouteID) {
	for _, rID := range routeIDs {
		delete(rt.activity, rID)
	}
}
