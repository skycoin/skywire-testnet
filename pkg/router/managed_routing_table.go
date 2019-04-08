package router

import (
	"sync"
	"time"

	"github.com/skycoin/skywire/pkg/routing"
)

var routeKeepalive = 10 * time.Minute // interval to keep active expired routes

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

func (rt *managedRoutingTable) Rule(routeID routing.RouteID) (routing.Rule, error) {
	rt.mu.Lock()
	rt.activity[routeID] = time.Now()
	rt.mu.Unlock()
	return rt.Table.Rule(routeID)
}

func (rt *managedRoutingTable) Cleanup() error {
	expiredIDs := []routing.RouteID{}
	rt.mu.Lock()
	err := rt.RangeRules(func(routeID routing.RouteID, rule routing.Rule) bool {
		if rule.Expiry().Before(time.Now()) {
			if lastActivity, ok := rt.activity[routeID]; !ok || time.Since(lastActivity) > routeKeepalive {
				expiredIDs = append(expiredIDs, routeID)
			}
		}
		return true
	})
	rt.mu.Unlock()

	if err != nil {
		return err
	}

	return rt.DeleteRules(expiredIDs...)
}
