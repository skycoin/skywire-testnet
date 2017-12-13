package factory

import (
	"sync"

	"github.com/skycoin/skycoin/src/cipher"
)

type Service struct {
	Key               cipher.PubKey
	Attributes        []string `json:",omitempty"`
	Address           string
	HideFromDiscovery bool
	AllowNodes        []string
}

type NodeServices struct {
	Services       []*Service
	ServiceAddress string
}

type ServiceNodes struct {
	Service *Service
	Nodes   map[cipher.PubKey]*NodeServices
}

type serviceDiscovery struct {
	subscription2Subscriber      map[cipher.PubKey]*ServiceNodes
	subscription2SubscriberMutex sync.RWMutex

	// attribute => subscription key
	attribute2Keys map[string]map[cipher.PubKey]struct{}
	key2Attributes map[cipher.PubKey]map[string]struct{}
}

func newServiceDiscovery() serviceDiscovery {
	return serviceDiscovery{
		subscription2Subscriber: make(map[cipher.PubKey]*ServiceNodes),
		attribute2Keys:          make(map[string]map[cipher.PubKey]struct{}),
		key2Attributes:          make(map[cipher.PubKey]map[string]struct{}),
	}
}

func (sd *serviceDiscovery) pack() *NodeServices {
	sd.subscription2SubscriberMutex.RLock()
	defer sd.subscription2SubscriberMutex.RUnlock()
	if len(sd.key2Attributes) < 1 {
		return nil
	}
	ss := make([]*Service, 0, len(sd.key2Attributes))
	for k, v := range sd.key2Attributes {
		attrs := make([]string, 0, len(v))
		for attr := range v {
			attrs = append(attrs, attr)
		}
		s := &Service{Key: k, Attributes: attrs}
		ss = append(ss, s)
	}
	ns := &NodeServices{Services: ss}
	return ns
}

func (sd *serviceDiscovery) register(conn *Connection, ns *NodeServices) {
	if len(ns.Services) < 1 {
		sd.subscription2SubscriberMutex.Lock()
		sd._unregister(conn)
		sd.subscription2SubscriberMutex.Unlock()
		conn.setServices(nil)
		return
	}
	sd.subscription2SubscriberMutex.Lock()
	defer sd.subscription2SubscriberMutex.Unlock()
	sd._unregister(conn)

	for _, service := range ns.Services {
		nodes, ok := sd.subscription2Subscriber[service.Key]
		if !ok {
			nodes = &ServiceNodes{Nodes: make(map[cipher.PubKey]*NodeServices), Service: service}
			nodes.Nodes[conn.GetKey()] = ns
			sd.subscription2Subscriber[service.Key] = nodes
		} else {
			nodes.Nodes[conn.GetKey()] = ns
		}

		for _, attr := range service.Attributes {
			am, ok := sd.attribute2Keys[attr]
			if !ok {
				am = make(map[cipher.PubKey]struct{})
				am[service.Key] = struct{}{}
				sd.attribute2Keys[attr] = am
			} else {
				am[service.Key] = struct{}{}
			}

			if service.HideFromDiscovery {
				continue
			}
			km, ok := sd.key2Attributes[service.Key]
			if !ok {
				km = make(map[string]struct{})
				km[attr] = struct{}{}
				sd.key2Attributes[service.Key] = km
			} else {
				km[attr] = struct{}{}
			}
		}
	}
	conn.setServices(ns)
}

func (sd *serviceDiscovery) _unregister(conn *Connection) {
	ns := conn.GetServices()
	if ns == nil {
		return
	}
	for _, service := range ns.Services {
		m, ok := sd.subscription2Subscriber[service.Key]
		if !ok {
			continue
		}
		delete(m.Nodes, conn.GetKey())
		// no one subscribes to service.Key
		if len(m.Nodes) < 1 {
			delete(sd.subscription2Subscriber, service.Key)

			for _, attr := range service.Attributes {
				am, ok := sd.attribute2Keys[attr]
				if !ok {
					continue
				}
				delete(am, service.Key)
				if len(am) < 1 {
					delete(sd.attribute2Keys, attr)
				}
			}
			if service.HideFromDiscovery {
				continue
			}
			_, ok = sd.key2Attributes[service.Key]
			if !ok {
				continue
			}
			delete(sd.key2Attributes, service.Key)
		}
	}
	conn.setServices(nil)
}

func (sd *serviceDiscovery) unregister(conn *Connection) {
	sd.subscription2SubscriberMutex.Lock()
	defer sd.subscription2SubscriberMutex.Unlock()

	sd._unregister(conn)
}

// find public keys of nodes by subscription key
func (sd *serviceDiscovery) find(key cipher.PubKey) []cipher.PubKey {
	sd.subscription2SubscriberMutex.RLock()
	defer sd.subscription2SubscriberMutex.RUnlock()

	m, ok := sd.subscription2Subscriber[key]
	if !ok {
		return nil
	}

	keys := make([]cipher.PubKey, 0, len(m.Nodes))
	for k := range m.Nodes {
		keys = append(keys, k)
	}
	return keys
}

// pubkey and address info of the node
type NodeInfo struct {
	// node key
	PubKey cipher.PubKey
	// node address
	Address string
}

// info of nodes for the service key
type ServiceInfo struct {
	// service key
	PubKey cipher.PubKey
	// nodes for the service key
	Nodes []*NodeInfo
}

// internal method without lock - find service address of nodes by subscription key
func (sd *serviceDiscovery) _findServiceAddress(key cipher.PubKey, exclude cipher.PubKey) []*NodeInfo {
	m, ok := sd.subscription2Subscriber[key]
	if !ok {
		return nil
	}

	result := make([]*NodeInfo, 0, len(m.Nodes))
	for k, v := range m.Nodes {
		if k == exclude {
			continue
		}
		result = append(result, &NodeInfo{
			PubKey:  k,
			Address: v.ServiceAddress,
		})
	}
	return result
}

// find service address of nodes by subscription key
func (sd *serviceDiscovery) findServiceAddresses(keys []cipher.PubKey, exclude cipher.PubKey) (result []*ServiceInfo) {
	if len(keys) < 1 {
		return nil
	}
	result = make([]*ServiceInfo, len(keys))
	check := make(map[cipher.PubKey]struct{}, len(keys))

	sd.subscription2SubscriberMutex.RLock()
	defer sd.subscription2SubscriberMutex.RUnlock()

	for _, k := range keys {
		if _, ok := check[k]; ok {
			continue
		}
		result = append(result, &ServiceInfo{
			PubKey: k,
			Nodes:  sd._findServiceAddress(k, exclude),
		})
		check[k] = struct{}{}
	}
	return
}

// find public keys of nodes by subscription attrs
// return intersect map of node key => sub keys
func (sd *serviceDiscovery) findByAttributes(attrs ...string) map[string][]cipher.PubKey {
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
	nodes := make(map[string][]cipher.PubKey)
	for _, key := range keys {
		m, ok := sd.subscription2Subscriber[key]
		if !ok {
			continue
		}
		for k := range m.Nodes {
			nodes[k.Hex()] = append(nodes[k.Hex()], key)
		}
	}
	return nodes
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

func (sd *serviceDiscovery) RangeServiceToServiceNodesMap(fn func(key cipher.PubKey, value *ServiceNodes)) {
	sd.subscription2SubscriberMutex.RLock()
	for k, v := range sd.subscription2Subscriber {
		fn(k, v)
	}
	sd.subscription2SubscriberMutex.RUnlock()
}
