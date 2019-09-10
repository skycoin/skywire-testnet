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
	SaveRule(RouteID, Rule) error

	// Rule returns RoutingRule with a given RouteID.
	Rule(RouteID) (Rule, error)

	// AllRules returns all non timed out rules with a given route descriptor.
	RulesWithDesc(RouteDescriptor) []Rule

	// AllRules returns all non timed out rules.
	AllRules() []RuleEntry

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

// RuleEntry is a pair of a RouteID and a Rule.
type RuleEntry struct {
	RouteID RouteID
	Rule    Rule
}

func (mt *memTable) AllRules() []RuleEntry {
	mt.RLock()
	defer mt.RUnlock()

	rules := make([]RuleEntry, 0, len(mt.rules))
	for k, v := range mt.rules {
		if !mt.ruleIsTimedOut(k, v) {
			entry := RuleEntry{
				RouteID: k,
				Rule:    v,
			}
			rules = append(rules, entry)
		}
	}

	return rules
}

func (mt *memTable) DelRules(routeIDs []RouteID) {
	for _, routeID := range routeIDs {
		mt.Lock()
		mt.delRule(routeID)
		mt.Unlock()
	}
}

func (mt *memTable) delRule(routeID RouteID) {
	delete(mt.rules, routeID)
	delete(mt.activity, routeID)
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
func (mt *memTable) ruleIsTimedOut(routeID RouteID, rule Rule) bool {
	lastActivity, ok := mt.activity[routeID]
	idling := time.Since(lastActivity)
	keepAlive := rule.KeepAlive()
	return !ok || idling > keepAlive
}
