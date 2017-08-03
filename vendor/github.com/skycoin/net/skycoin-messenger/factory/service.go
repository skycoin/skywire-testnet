package factory

import (
	"sync"

	"github.com/skycoin/skycoin/src/cipher"
)

type Service struct {
	Key        cipher.PubKey
	Attributes []string
}

type serviceDiscovery struct {
	// subscription key => connection key => connection
	subscription2Subscriber      map[cipher.PubKey]map[cipher.PubKey]*Connection
	subscription2SubscriberMutex sync.RWMutex

	// attribute => subscription key
	attribute2Keys  map[string]map[cipher.PubKey]struct{}
	keys2Attributes map[cipher.PubKey]map[string]struct{}
}

func newServiceDiscovery() serviceDiscovery {
	return serviceDiscovery{
		subscription2Subscriber: make(map[cipher.PubKey]map[cipher.PubKey]*Connection),
		attribute2Keys:          make(map[string]map[cipher.PubKey]struct{}),
		keys2Attributes:         make(map[cipher.PubKey]map[string]struct{}),
	}
}

func (sd *serviceDiscovery) register(conn *Connection, subs []*Service) {
	if len(subs) < 1 {
		return
	}
	sd.subscription2SubscriberMutex.Lock()
	defer sd.subscription2SubscriberMutex.Unlock()

	for _, sub := range subs {
		m, ok := sd.subscription2Subscriber[sub.Key]
		if !ok {
			m = make(map[cipher.PubKey]*Connection)
			m[conn.GetKey()] = conn
			sd.subscription2Subscriber[sub.Key] = m
		} else {
			m[conn.GetKey()] = conn
		}

		for _, attr := range sub.Attributes {
			am, ok := sd.attribute2Keys[attr]
			if !ok {
				am = make(map[cipher.PubKey]struct{})
				am[sub.Key] = struct{}{}
				sd.attribute2Keys[attr] = am
			} else {
				am[sub.Key] = struct{}{}
			}

			km, ok := sd.keys2Attributes[sub.Key]
			if !ok {
				km = make(map[string]struct{})
				km[attr] = struct{}{}
				sd.keys2Attributes[sub.Key] = km
			} else {
				km[attr] = struct{}{}
			}
		}
	}
	conn.setServices(subs)
}

func (sd *serviceDiscovery) unregister(conn *Connection) {
	sd.subscription2SubscriberMutex.Lock()
	defer sd.subscription2SubscriberMutex.Unlock()

	for _, sub := range conn.GetServices() {
		m, ok := sd.subscription2Subscriber[sub.Key]
		if !ok {
			continue
		}
		delete(m, conn.GetKey())
		// no one subscribes to sub.Key
		if len(m) < 1 {
			delete(sd.subscription2Subscriber, sub.Key)

			as, ok := sd.keys2Attributes[sub.Key]
			if !ok {
				continue
			}
			delete(sd.keys2Attributes, sub.Key)
			for attr := range as {
				am, ok := sd.attribute2Keys[attr]
				if !ok {
					continue
				}
				delete(am, sub.Key)
				if len(am) < 1 {
					delete(sd.attribute2Keys, attr)
				}
			}
		}
	}
}

// find public keys of nodes by subscription key
func (sd *serviceDiscovery) find(key cipher.PubKey) []cipher.PubKey {
	sd.subscription2SubscriberMutex.RLock()
	defer sd.subscription2SubscriberMutex.RUnlock()

	m, ok := sd.subscription2Subscriber[key]
	if !ok {
		return nil
	}

	keys := make([]cipher.PubKey, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// find public keys of nodes by subscription attrs
// return intersect slice
func (sd *serviceDiscovery) findByAttributes(attrs []string) []cipher.PubKey {
	if len(attrs) < 1 {
		return nil
	}
	sd.subscription2SubscriberMutex.RLock()
	defer sd.subscription2SubscriberMutex.RUnlock()

	var maps []map[cipher.PubKey]struct{}
	for _, attr := range attrs {
		m, ok := sd.attribute2Keys[attr]
		if !ok {
			return nil
		}
		maps = append(maps, m)
	}

	keys := intersectKeys(maps)
	result := make(map[cipher.PubKey]struct{})
	for _, key := range keys {
		m, ok := sd.subscription2Subscriber[key]
		if !ok {
			continue
		}
		for k := range m {
			result[k] = struct{}{}
		}
	}
	return mapKeys(result)
}

func mapKeys(m map[cipher.PubKey]struct{}) (keys []cipher.PubKey) {
	if len(m) < 1 {
		return
	}
	keys = make([]cipher.PubKey, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return
}

func intersectKeys(maps []map[cipher.PubKey]struct{}) (keys []cipher.PubKey) {
	if len(maps) < 1 {
		return
	}
	m := maps[0]
	keys = make([]cipher.PubKey, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	if len(maps) == 1 {
		return
	}

	var result []cipher.PubKey
	for _, key := range keys {
		if isKeyInMaps(key, maps[1:]) {
			result = append(result, key)
		}
	}
	return result
}

func isKeyInMaps(key cipher.PubKey, maps []map[cipher.PubKey]struct{}) bool {
	for _, m := range maps {
		if _, ok := m[key]; !ok {
			return false
		}
	}
	return true
}
