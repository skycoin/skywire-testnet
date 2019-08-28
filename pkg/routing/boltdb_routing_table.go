package routing

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"sync/atomic"

	"github.com/skycoin/skycoin/src/util/logging"
	"go.etcd.io/bbolt"
)

var boltDBBucket = []byte("routing")
var log = logging.MustGetLogger("routing")

// BoltDBRoutingTable implements RoutingTable on top of BoltDB.
type boltDBRoutingTable struct {
	db       *bbolt.DB
	exceeded uint32
}

// BoltDBRoutingTable constructs a new BoldDBRoutingTable.
func BoltDBRoutingTable(path string) (Table, error) {
	db, err := bbolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(boltDBBucket); err != nil {
			return fmt.Errorf("failed to create bucket: %s", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &boltDBRoutingTable{db: db}, nil
}

// AddRule adds routing rule to the table and returns assigned Route ID.
func (rt *boltDBRoutingTable) AddRule(rule Rule) (routeID RouteID, err error) {
	if atomic.LoadUint32(&rt.exceeded) != 0 {
		return 0, errors.New("no available routeIDs")
	}

	err = rt.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(boltDBBucket)
		nextID, err := b.NextSequence()
		if err != nil {
			return err
		}

		if nextID >= math.MaxUint32 {
			atomic.StoreUint32(&rt.exceeded, 1)
		}

		routeID = RouteID(nextID)
		return b.Put(binaryID(routeID), rule)
	})

	return routeID, err
}

// SetRule sets RoutingRule for a given RouteID.
func (rt *boltDBRoutingTable) SetRule(routeID RouteID, rule Rule) error {
	return rt.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(boltDBBucket)

		return b.Put(binaryID(routeID), []byte(rule))
	})
}

// Rule returns RoutingRule with a given RouteID.
func (rt *boltDBRoutingTable) Rule(routeID RouteID) (Rule, error) {
	var rule Rule
	err := rt.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(boltDBBucket)
		rule = b.Get(binaryID(routeID))
		return nil
	})

	return rule, err
}

// RangeRules iterates over all rules and yields values to the rangeFunc until `next` is false.
func (rt *boltDBRoutingTable) RangeRules(rangeFunc RangeFunc) error {
	return rt.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(boltDBBucket)
		f := func(k, v []byte) error {
			if !rangeFunc(RouteID(binary.BigEndian.Uint32(k)), v) {
				return errors.New("iterator stopped")
			}

			return nil
		}
		if err := b.ForEach(f); err != nil {
			log.Warn(err)
		}
		return nil
	})
}

// AddRule adds routing rule to the table and returns assigned Route ID.
func (rt *boltDBRoutingTable) ReserveRouteID() (routeID RouteID, err error) {
	if atomic.LoadUint32(&rt.exceeded) != 0 {
		return 0, errors.New("no available routeIDs")
	}

	err = rt.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(boltDBBucket)
		nextID, err := b.NextSequence()
		if err != nil {
			return err
		}

		if nextID >= math.MaxUint32 {
			atomic.StoreUint32(&rt.exceeded, 1)
		}

		routeID = RouteID(nextID)
		return nil
	})

	return routeID, err
}

// Rules returns RoutingRules for a given RouteIDs.
func (rt *boltDBRoutingTable) Rules(routeIDs ...RouteID) (rules []Rule, err error) {
	rules = []Rule{}
	err = rt.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(boltDBBucket)

		for _, routeID := range routeIDs {
			rule := b.Get(binaryID(routeID))
			if rule != nil {
				rules = append(rules, rule)
			}
		}
		return nil
	})

	return rules, err
}

// DeleteRules removes RoutingRules with a given a RouteIDs.
func (rt *boltDBRoutingTable) DeleteRules(routeIDs ...RouteID) error {
	return rt.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(boltDBBucket)

		var dErr error
		for _, routeID := range routeIDs {
			if err := b.Delete(binaryID(routeID)); err != nil {
				dErr = err
			}
		}

		return dErr
	})
}

// Count returns the number of routing rules stored.
func (rt *boltDBRoutingTable) Count() (count int) {
	err := rt.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(boltDBBucket)

		stats := b.Stats()
		count = stats.KeyN
		return nil
	})
	if err != nil {
		return 0
	}

	return count
}

// Close closes underlying BoltDB instance.
func (rt *boltDBRoutingTable) Close() error {
	if rt == nil {
		return nil
	}
	return rt.db.Close()
}

func binaryID(v RouteID) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(v))
	return b
}
