package routing

import (
	"errors"
	"fmt"
	"math"
	"sync"
)

// RangeFunc is used by RangeRules to iterate over rules.
type RangeFunc func(routeID RouteID, rule Rule) (next bool)

// Table represents a routing table implementation.
type Table interface {
	// AddRule adds a new RoutingRules to the table and returns assigned RouteID.
	AddRule(rule Rule) (routeID RouteID, err error)

	// SetRule sets RoutingRule for a given RouteID.
	SetRule(routeID RouteID, rule Rule) error

	// Rule returns RoutingRule with a given RouteID.
	Rule(routeID RouteID) (Rule, error)

	// DeleteRules removes RoutingRules with a given a RouteIDs.
	DeleteRules(routeIDs ...RouteID) error

	// RangeRules iterates over all rules and yields values to the rangeFunc until `next` is false.
	RangeRules(rangeFunc RangeFunc) error

	// ReserveRouteID reserves a RouteID.
	ReserveRouteID() (routeID RouteID, err error)

	// Count returns the number of RoutingRule entries stored.
	Count() int

	// Close safely closes routing table.
	Close() error
}

type inMemoryRoutingTable struct {
	sync.RWMutex

	nextID RouteID
	rules  map[RouteID]Rule
}

// InMemoryRoutingTable return in-memory RoutingTable implementation.
func InMemoryRoutingTable() Table {
	return &inMemoryRoutingTable{
		rules: map[RouteID]Rule{},
	}
}

func (rt *inMemoryRoutingTable) AddRule(rule Rule) (RouteID, error) {
	routeID, err := rt.ReserveRouteID()
	if err != nil {
		return 0, err
	}

	rt.Lock()
	defer rt.Unlock()

	rt.rules[routeID] = rule

	return rt.nextID, nil
}

func (rt *inMemoryRoutingTable) SetRule(routeID RouteID, rule Rule) error {
	rt.Lock()
	rt.rules[routeID] = rule
	rt.Unlock()

	return nil
}

func (rt *inMemoryRoutingTable) Rule(routeID RouteID) (Rule, error) {
	rt.RLock()
	rule, ok := rt.rules[routeID]
	rt.RUnlock()
	if !ok {
		return nil, fmt.Errorf("rule of id %v not found", routeID)
	}
	return rule, nil
}

func (rt *inMemoryRoutingTable) RangeRules(rangeFunc RangeFunc) error {
	rt.RLock()
	for routeID, rule := range rt.rules {
		if !rangeFunc(routeID, rule) {
			break
		}
	}
	rt.RUnlock()

	return nil
}

func (rt *inMemoryRoutingTable) DeleteRules(routeIDs ...RouteID) error {
	rt.Lock()
	for _, routeID := range routeIDs {
		delete(rt.rules, routeID)
	}
	rt.Unlock()

	return nil
}

func (rt *inMemoryRoutingTable) ReserveRouteID() (RouteID, error) {
	rt.Lock()
	defer rt.Unlock()

	if rt.nextID == math.MaxUint32 {
		return 0, errors.New("no available routeIDs")
	}

	rt.nextID++
	return rt.nextID, nil
}

func (rt *inMemoryRoutingTable) Count() int {
	rt.RLock()
	count := len(rt.rules)
	rt.RUnlock()
	return count
}

func (rt *inMemoryRoutingTable) Close() error {
	return nil
}
