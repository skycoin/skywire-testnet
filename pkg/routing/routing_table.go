package routing

import (
	"errors"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/skycoin/skycoin/src/util/logging"
)

var log = logging.MustGetLogger("routing")

// DefaultGCInterval is the default duration for garbage collection of routing rules.
const DefaultGCInterval = 5 * time.Second

// ErrRuleTimedOut is being returned while trying to access the rule which timed out
var ErrRuleTimedOut = errors.New("rule keep-alive timeout exceeded")

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

	// Count returns the number of RoutingRule entries stored.
	Count() int
}

type memTable struct {
	sync.RWMutex

	nextID     uint32
	rules      map[RouteID]Rule
	activity   map[RouteID]time.Time
	gcInterval time.Duration
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
		rules:      map[RouteID]Rule{},
		activity:   make(map[RouteID]time.Time),
		gcInterval: config.GCInterval,
	}

	go mt.gcLoop()

	return mt
}

func (mt *memTable) AddRule(rule Rule) (routeID RouteID, err error) {
	if routeID == math.MaxUint32 {
		return 0, errors.New("no available routeIDs")
	}

	routeID = RouteID(atomic.AddUint32(&mt.nextID, 1))

	mt.Lock()
	mt.rules[routeID] = rule
	mt.activity[routeID] = time.Now()
	mt.Unlock()

	return routeID, nil
}

func (mt *memTable) SetRule(routeID RouteID, rule Rule) error {
	mt.Lock()
	mt.rules[routeID] = rule
	mt.Unlock()

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

func (mt *memTable) RangeRules(rangeFunc RangeFunc) error {
	mt.RLock()
	for routeID, rule := range mt.rules {
		if !rangeFunc(routeID, rule) {
			break
		}
	}
	mt.RUnlock()

	return nil
}

func (mt *memTable) DeleteRules(routeIDs ...RouteID) error {
	mt.Lock()
	for _, routeID := range routeIDs {
		delete(mt.rules, routeID)
	}
	mt.Unlock()

	return nil
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
	if mt.gcInterval <= 0 {
		return
	}
	ticker := time.NewTicker(mt.gcInterval)
	defer ticker.Stop()
	for range ticker.C {
		if err := mt.gc(); err != nil {
			log.WithError(err).Warnf("routing table gc")
		}
	}
}

func (mt *memTable) gc() error {
	expiredIDs := make([]RouteID, 0)

	err := mt.RangeRules(func(routeID RouteID, rule Rule) bool {
		if rule.Type() == RuleIntermediaryForward && mt.ruleIsTimedOut(routeID, rule) {
			expiredIDs = append(expiredIDs, routeID)
		}
		return true
	})
	if err != nil {
		return err
	}

	if err := mt.DeleteRules(expiredIDs...); err != nil {
		return err
	}

	mt.Lock()
	defer mt.Unlock()
	mt.deleteActivity(expiredIDs...)

	return nil
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
