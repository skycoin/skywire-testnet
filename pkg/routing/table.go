package routing

import (
	"errors"
	"fmt"
	"math"
	"sync"
	"time"
)

// DefaultGCInterval is the default duration for garbage collection of routing rules.
const DefaultGCInterval = 5 * time.Second

var (
	// ErrRuleTimedOut is being returned while trying to access the rule which timed out
	ErrRuleTimedOut = errors.New("rule keep-alive timeout exceeded")
	// ErrNoAvailableRoutes is returned when there're no more available routeIDs
	ErrNoAvailableRoutes = errors.New("no available routeIDs")
)

// RangeFunc is used by RangeRules to iterate over rules.
type RangeFunc func(routeID RouteID, rule Rule) (next bool)

// Table represents a routing table implementation.
type Table interface {
	// ReserveKey reserves a RouteID.
	ReserveKey() (RouteID, error)

	// SaveRule sets RoutingRule for a given RouteID.
	SaveRule(RouteID, Rule) error

	// Rule returns RoutingRule with a given RouteID.
	Rule(RouteID) (Rule, error)

	// AllRules returns all non timed out rules with a given route descriptor.
	RulesWithDesc(RouteDescriptor) []Rule

	// AllRules returns all non timed out rules.
	AllRules() []Rule

	// DelRules removes RoutingRules with a given a RouteIDs.
	DelRules([]RouteID)

	// RangeRules iterates over all rules and yields values to the rangeFunc until `next` is false.
	RangeRules(RangeFunc)

	// Count returns the number of RoutingRule entries stored.
	Count() int
}

type memTable struct {
	sync.RWMutex

	config   Config
	nextID   RouteID
	rules    map[RouteID]Rule
	activity map[RouteID]time.Time
}

// Config represents a routing table configuration.
type Config struct {
	GCInterval time.Duration
}

// DefaultConfig represents the default configuration of routing table.
func DefaultConfig() Config {
	return Config{
		GCInterval: DefaultGCInterval,
	}
}

// New returns an in-memory routing table implementation.
func New() Table {
	return NewWithConfig(DefaultConfig())
}

// NewWithConfig returns an in-memory routing table implementation with a specified configuration.
func NewWithConfig(config Config) Table {
	if config.GCInterval <= 0 {
		config.GCInterval = DefaultGCInterval
	}

	mt := &memTable{
		config:   config,
		rules:    map[RouteID]Rule{},
		activity: make(map[RouteID]time.Time),
	}

	go mt.gcLoop()

	return mt
}

func (mt *memTable) ReserveKey() (RouteID, error) {
	mt.Lock()
	defer mt.Unlock()

	if mt.nextID == math.MaxUint32 {
		return 0, ErrNoAvailableRoutes
	}

	mt.nextID++
	return mt.nextID, nil
}

func (mt *memTable) SaveRule(routeID RouteID, rule Rule) error {
	mt.Lock()
	defer mt.Unlock()

	mt.rules[routeID] = rule
	mt.activity[routeID] = time.Now()

	return nil
}

func (mt *memTable) Rule(routeID RouteID) (Rule, error) {
	mt.RLock()
	rule, ok := mt.rules[routeID]
	mt.RUnlock()

	if !ok {
		return nil, fmt.Errorf("rule of id %v not found", routeID)
	}

	if mt.ruleIsTimedOut(routeID, rule) {
		return nil, ErrRuleTimedOut
	}

	return rule, nil
}

func (mt *memTable) RulesWithDesc(desc RouteDescriptor) []Rule {
	mt.RLock()
	defer mt.RUnlock()

	rules := make([]Rule, 0, len(mt.rules))
	for k, v := range mt.rules {
		if !mt.ruleIsTimedOut(k, v) && v.RouteDescriptor() == desc {
			rules = append(rules, v)
		}
	}

	return rules
}

func (mt *memTable) AllRules() []Rule {
	mt.RLock()
	defer mt.RUnlock()

	rules := make([]Rule, 0, len(mt.rules))
	for k, v := range mt.rules {
		if !mt.ruleIsTimedOut(k, v) {
			rules = append(rules, v)
		}
	}

	return rules
}

func (mt *memTable) RangeRules(rangeFunc RangeFunc) {
	mt.RLock()
	defer mt.RUnlock()

	for routeID, rule := range mt.rules {
		if !rangeFunc(routeID, rule) {
			break
		}
	}
}

func (mt *memTable) DelRules(routeIDs []RouteID) {
	mt.Lock()
	defer mt.Unlock()

	for _, routeID := range routeIDs {
		delete(mt.rules, routeID)
	}
}

func (mt *memTable) Count() int {
	mt.RLock()
	count := len(mt.rules)
	mt.RUnlock()
	return count
}

func (mt *memTable) Close() error {
	return nil
}

// Routing table garbage collect loop.
func (mt *memTable) gcLoop() {
	ticker := time.NewTicker(mt.config.GCInterval)
	defer ticker.Stop()

	for range ticker.C {
		mt.gc()
	}
}

func (mt *memTable) gc() {
	expiredIDs := make([]RouteID, 0)

	mt.RangeRules(func(routeID RouteID, rule Rule) bool {
		if rule.Type() == RuleIntermediaryForward && mt.ruleIsTimedOut(routeID, rule) {
			expiredIDs = append(expiredIDs, routeID)
		}
		return true
	})

	mt.DelRules(expiredIDs)

	mt.Lock()
	defer mt.Unlock()
	mt.deleteActivity(expiredIDs...)
}

// ruleIsExpired checks whether rule's keep alive timeout is exceeded.
// NOTE: for internal use, is NOT thread-safe, object lock should be acquired outside
func (mt *memTable) ruleIsTimedOut(routeID RouteID, rule Rule) bool {
	lastActivity, ok := mt.activity[routeID]
	return !ok || time.Since(lastActivity) > rule.KeepAlive()
}

// deleteActivity removes activity records for the specified set of `routeIDs`.
// NOTE: for internal use, is NOT thread-safe, object lock should be acquired outside
func (mt *memTable) deleteActivity(routeIDs ...RouteID) {
	for _, rID := range routeIDs {
		delete(mt.activity, rID)
	}
}
