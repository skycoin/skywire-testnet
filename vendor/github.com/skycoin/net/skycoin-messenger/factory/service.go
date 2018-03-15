package factory

import (
	"github.com/skycoin/skycoin/src/cipher"
	"sync"
)

type Service struct {
	Key               cipher.PubKey
	Attributes        []string `json:",omitempty"`
	Address           string   `json:",omitempty"`
	HideFromDiscovery bool     `json:",omitempty"`
	AllowNodes        []string `json:",omitempty"`
	Version           string   `json:",omitempty"`
}

type NodeServices struct {
	// Services that node provides
	Services []*Service
	// Used for node extra service listen address
	ServiceAddress string `json:",omitempty"`
	// Node location
	Location string `json:",omitempty"`
	// Node version info
	Version []string `json:",omitempty"`
}

type serviceDiscovery struct {
	subscription2Subscriber      map[cipher.PubKey]*NodeServices
	subscription2SubscriberMutex sync.RWMutex

	RegisterService         func(key cipher.PubKey, ns *NodeServices) (err error)
	UnRegisterService       func(key cipher.PubKey) (err error)
	FindServiceAddresses    func(keys []cipher.PubKey, exclude cipher.PubKey) (result []*ServiceInfo)
	FindByAttributes        func(attrs ...string) (result *AttrNodesInfo)
	FindByAttributesAndPaging func(page, limit int, attrs ...string) (result *AttrNodesInfo)
}

func newServiceDiscovery() serviceDiscovery {
	return serviceDiscovery{
		subscription2Subscriber: make(map[cipher.PubKey]*NodeServices),
	}
}

func (sd *serviceDiscovery) pack() *NodeServices {
	sd.subscription2SubscriberMutex.RLock()
	defer sd.subscription2SubscriberMutex.RUnlock()
	if len(sd.subscription2Subscriber) < 1 {
		return nil
	}
	var ss []*Service
	for _, value := range sd.subscription2Subscriber {
		for _, service := range value.Services {
			ss = append(ss, service)
		}
	}
	ns := &NodeServices{
		Services: ss,
	}
	return ns
}

func (sd *serviceDiscovery) register(conn *Connection, ns *NodeServices) {
	if !conn.IsKeySet() {
		return
	}
	filter := make(map[cipher.PubKey]*Service)
	for _, service := range ns.Services {
		filter[service.Key] = service
	}
	ns.Services = make([]*Service, 0, len(filter))
	for _, s := range filter {
		ns.Services = append(ns.Services, s)
	}
	if len(ns.Services) < 1 {
		sd.subscription2SubscriberMutex.Lock()
		sd._unregister(conn)
		sd.subscription2SubscriberMutex.Unlock()
		conn.setServices(nil)
		return
	}
	sd.subscription2SubscriberMutex.Lock()
	sd._unregister(conn)
	sd.subscription2Subscriber[conn.GetKey()] = ns
	conn.setServices(ns)
	sd.subscription2SubscriberMutex.Unlock()
}

func (sd *serviceDiscovery) discoveryRegister(conn *Connection, ns *NodeServices) {
	if !conn.IsKeySet() {
		return
	}
	filter := make(map[cipher.PubKey]*Service)
	for _, service := range ns.Services {
		filter[service.Key] = service
	}
	ns.Services = make([]*Service, 0, len(filter))
	for _, s := range filter {
		ns.Services = append(ns.Services, s)
	}
	if len(ns.Services) < 1 {
		sd.subscription2SubscriberMutex.Lock()
		sd._unregister(conn)
		sd.subscription2SubscriberMutex.Unlock()
		conn.setServices(nil)
		return
	}
	sd.subscription2SubscriberMutex.Lock()
	sd._discoveryUnregister(conn)
	err := sd.registerService(conn.GetKey(), ns)
	if err != nil {
		conn.GetContextLogger().Errorf("set service: %s", err)
	}
	conn.setServices(ns)
	sd.subscription2SubscriberMutex.Unlock()
}

func (sd *serviceDiscovery) _discoveryUnregister(conn *Connection) {
	ns := conn.GetServices()
	if ns == nil || !conn.IsKeySet() {
		return
	}
	err := sd.unRegisterService(conn.GetKey())
	if err != nil {
		conn.GetContextLogger().Errorf("unRegister service: %s", err)
	}
	conn.setServices(nil)
}

func (sd *serviceDiscovery) _unregister(conn *Connection) {
	ns := conn.GetServices()
	if ns == nil || !conn.IsKeySet() {
		return
	}
	delete(sd.subscription2Subscriber, conn.GetKey())
	conn.setServices(nil)
}

func (sd *serviceDiscovery) unDiscoveryregister(conn *Connection) {
	sd._discoveryUnregister(conn)
}

func (sd *serviceDiscovery) unregister(conn *Connection) {
	sd.subscription2SubscriberMutex.Lock()
	sd._unregister(conn)
	sd.subscription2SubscriberMutex.Unlock()
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

// find service address of nodes by subscription key
// keys is app keys
// exclude is node key
func (sd *serviceDiscovery) findServiceAddresses(keys []cipher.PubKey, exclude cipher.PubKey) (result []*ServiceInfo) {
	if sd.FindServiceAddresses != nil {
		return sd.FindServiceAddresses(keys, exclude)
	}
	return
}

type AttrNodesInfo struct {
	Nodes []*AttrNodeInfo
	Count int64
}

type AttrNodeInfo struct {
	Node     cipher.PubKey
	Apps     []cipher.PubKey
	Location string
	Version  []string
	AppInfos []*AttrAppInfo
}

type AttrAppInfo struct {
	Key     cipher.PubKey
	Version string
}

// find public keys of nodes by subscription attrs
// return intersect map of node key => sub keys
func (sd *serviceDiscovery) findByAttributes(attrs ...string) (result *AttrNodesInfo) {
	if sd.FindByAttributes != nil {
		return sd.FindByAttributes(attrs...)
	}
	return
}

func (sd *serviceDiscovery) findByAttributesAndPaging(page, limit int, attrs ...string) (result *AttrNodesInfo) {
	if sd.FindByAttributes != nil {
		return sd.FindByAttributesAndPaging(page, limit, attrs...)
	}
	return
}

func (sd *serviceDiscovery) registerService(key cipher.PubKey, ns *NodeServices) (err error) {
	if sd.RegisterService != nil {
		return sd.RegisterService(key, ns)
	}
	return
}

func (sd *serviceDiscovery) unRegisterService(key cipher.PubKey) (err error) {
	if sd.UnRegisterService != nil {
		return sd.UnRegisterService(key)
	}
	return
}
