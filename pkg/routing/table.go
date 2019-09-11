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

// Table represents a routing table implementation.
type Table interface {
	// ReserveKey reserves a RouteID.
	ReserveKey() (RouteID, error)

	// SaveRule sets RoutingRule for a given RouteID.
	SaveRule(Rule) error

	// Rule returns RoutingRule with a given RouteID.
	Rule(RouteID) (Rule, error)

	// AllRules returns all non timed out rules with a given route descriptor.
	RulesWithDesc(RouteDescriptor) []Rule

	// AllRules returns all non timed out rules.
	AllRules() []Rule

	// DelRules removes RoutingRules with a given a RouteIDs.
	DelRules([]RouteID)

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

// NewTable returns an in-memory routing table implementation with a specified configuration.
func NewTable(config Config) Table {
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

func (mt *memTable) ReserveKey() (key RouteID, err error) {
	mt.Lock()
	defer mt.Unlock()

	if mt.nextID == math.MaxUint32 {
		return 0, ErrNoAvailableRoutes
	}

	mt.nextID++
	return mt.nextID, nil
}

func (mt *memTable) SaveRule(rule Rule) error {
	key := rule.KeyRouteID()
	now := time.Now()

	mt.Lock()
	defer mt.Unlock()

	mt.rules[key] = rule
	mt.activity[key] = now

	return nil
}

func (mt *memTable) Rule(key RouteID) (Rule, error) {
	mt.RLock()
	rule, ok := mt.rules[key]
	mt.RUnlock()

	if !ok {
		return nil, fmt.Errorf("rule of id %v not found", key)
	}

	if mt.ruleIsTimedOut(key, rule) {
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

func (mt *memTable) DelRules(keys []RouteID) {
	for _, key := range keys {
		mt.Lock()
		mt.delRule(key)
		mt.Unlock()
	}
}

func (mt *memTable) delRule(key RouteID) {
	delete(mt.rules, key)
	delete(mt.activity, key)
}

func (mt *memTable) Count() int {
	mt.RLock()
	defer mt.RUnlock()

	return len(mt.rules)
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
	mt.Lock()
	defer mt.Unlock()

	for routeID, rule := range mt.rules {
		if rule.Type() == RuleIntermediaryForward && mt.ruleIsTimedOut(routeID, rule) {
			mt.delRule(routeID)
		}
	}
}

// ruleIsExpired checks whether rule's keep alive timeout is exceeded.
// NOTE: for internal use, is NOT thread-safe, object lock should be acquired outside
func (mt *memTable) ruleIsTimedOut(key RouteID, rule Rule) bool {
	lastActivity, ok := mt.activity[key]
	idling := time.Since(lastActivity)
	keepAlive := rule.KeepAlive()
	return !ok || idling > keepAlive
}
