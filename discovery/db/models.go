package db

import (
	"github.com/skycoin/net/skycoin-messenger/factory"
	"github.com/skycoin/skycoin/src/cipher"
	"time"
)

type Node struct {
	Id             int64
	Key            string
	ServiceAddress string
	Location       string
	Version        []string
	Priority       int
	Created        time.Time `xorm:"created"`
	Updated        time.Time `xorm:"updated"`
}

func (n *Node) TableName() string {
	return "node"
}

type Service struct {
	Id                int64
	Key               string
	Address           string
	HideFromDiscovery bool
	AllowNodes        []string
	Version           string
	NodeId            int64
	Created           time.Time `xorm:"created"`
	Updated           time.Time `xorm:"updated"`
}

func (s *Service) TableName() string {
	return "service"
}

type Attributes struct {
	Name      string `xorm:"varchar(20)"`
	ServiceId int64
}

func (a *Attributes) TableName() string {
	return "attributes"
}

func UnRegisterService(key cipher.PubKey) (err error) {
	sess := engine.NewSession()
	defer sess.Close()
	err = sess.Begin()
	if err != nil {
		return
	}
	node := &Node{}
	_, err = engine.Where("key = ?", key.Hex()).Get(node)
	if err != nil {
		sess.Rollback()
		return
	}
	services := make([]Service, 0)
	err = engine.Where("node_id = ?", node.Id).Find(&services)
	if err != nil {
		sess.Rollback()
		return
	}
	_, err = engine.Where("key = ?", key.Hex()).Delete(&Node{})
	if err != nil {
		sess.Rollback()
		return
	}
	_, err = engine.Where("node_id = ?", node.Id).Delete(&Service{})
	if err != nil {
		sess.Rollback()
		return
	}
	for _, v := range services {
		_, err = engine.Where("service_id == ?", v.Id).Delete(&Attributes{})
		if err != nil {
			sess.Rollback()
			return err
		}
	}
	sess.Commit()
	return
}

func DelByKeyString(key string) (err error) {
	_, err = engine.Where("key = ?", key).Delete(&Node{})
	return
}

func RegisterService(key cipher.PubKey, ns *factory.NodeServices) (err error) {
	sess := engine.NewSession()
	defer sess.Close()
	err = sess.Begin()
	if err != nil {
		return
	}
	exist, err := engine.Where("key = ?", key.Hex()).Exist(&Node{})
	if err != nil {
		return
	}
	node := &Node{
		Key:            key.Hex(),
		ServiceAddress: ns.ServiceAddress,
		Location:       ns.Location,
		Version:        ns.Version,
	}
	if exist {
		_, err = engine.Where("key = ?", key.Hex()).Update(node)
		if err != nil {
			sess.Rollback()
			return
		}
	} else {
		_, err = engine.Insert(node)
		if err != nil {
			sess.Rollback()
			return
		}
	}
	for _, v := range ns.Services {
		tmpService := &Service{}
		ok, err := engine.Where("key = ?", v.Key.Hex()).Get(tmpService)
		if err != nil {
			sess.Rollback()
			return err
		}
		service := &Service{
			Key:               v.Key.Hex(),
			Address:           v.Address,
			HideFromDiscovery: v.HideFromDiscovery,
			AllowNodes:        v.AllowNodes,
			Version:           v.Version,
			NodeId:            node.Id,
		}
		if ok {
			_, err = engine.Where("key = ?", v.Key.Hex()).Update(service)
			if err != nil {
				sess.Rollback()
				return err
			}
			_,err = engine.Where("service_id == ?",service.Id).Delete(&Attributes{})
			if err != nil {
				sess.Rollback()
				return err
			}
		} else {
			_, err = engine.Insert(service)
			if err != nil {
				sess.Rollback()
				return err
			}
		}
		for _, attr := range v.Attributes {
			_, err = engine.Insert(&Attributes{
				Name:      attr,
				ServiceId: service.Id,
			})
			if err != nil {
				sess.Rollback()
				return err
			}
		}

	}
	sess.Commit()
	return
}

type NodeDetail struct {
	Node       `xorm:"extends"`
	Service    `xorm:"extends"`
	Attributes `xorm:"extends"`
}

func (NodeDetail) TableName() string {
	return "node"
}

func FindResultByAttrs(attr ...string) (result *factory.AttrNodesInfo) {
	sas := make([]NodeDetail, 0)
	err := engine.Join("INNER", "service", "service.node_id = node.id").
		Join("INNER", "attributes", "attributes.service_id = service.id").
		In("attributes.name", attr).Desc("node.priority").Find(&sas)
	if err != nil {
		return
	}

	atis := make(map[string]*factory.AttrNodeInfo)
	for _, v := range sas {
		nodeKey, err := cipher.PubKeyFromHex(v.Node.Key)
		if err != nil {
			continue
		}
		appKey, err := cipher.PubKeyFromHex(v.Service.Key)
		if err != nil {
			continue
		}
		ati, ok := atis[v.Service.Key]
		if ok {
			ati.AppInfos = append(ati.AppInfos, &factory.AttrAppInfo{})
			atis[v.Service.Key] = ati
		} else {

			appinfos := make([]*factory.AttrAppInfo, 0)
			appinfos = append(appinfos, &factory.AttrAppInfo{
				Key:     appKey,
				Version: v.Service.Version,
			})
			apps := make([]cipher.PubKey, 0)
			appsKey, err := cipher.PubKeyFromHex(v.Service.Key)
			if err != nil {
				continue
			}
			apps = append(apps, appsKey)
			info := &factory.AttrNodeInfo{
				Node:     nodeKey,
				Apps:     apps,
				Location: v.Node.Location,
				Version:  v.Node.Version,
				AppInfos: appinfos,
			}
			atis[v.Service.Key] = info
		}
	}
	result = &factory.AttrNodesInfo{
		Nodes: make([]*factory.AttrNodeInfo, 0),
	}
	for _, v := range atis {
		result.Nodes = append(result.Nodes, v)
	}
	return
}

func FindResultByAttrsAndPaging(pages, limit int, attr ...string) (result *factory.AttrNodesInfo) {
	sas := make([]NodeDetail, 0)
	err := engine.Join("INNER", "service", "service.node_id = node.id").
		Join("INNER", "attributes", "attributes.service_id = service.id").
		In("attributes.name", attr).Limit(limit, (pages-1)*limit).Desc("node.priority").Find(&sas)
	if err != nil {
		return
	}
	atis := make(map[string]*factory.AttrNodeInfo)
	for _, v := range sas {
		nodeKey, err := cipher.PubKeyFromHex(v.Node.Key)
		if err != nil {
			continue
		}
		appKey, err := cipher.PubKeyFromHex(v.Service.Key)
		if err != nil {
			continue
		}
		ati, ok := atis[v.Service.Key]
		if ok {
			ati.AppInfos = append(ati.AppInfos, &factory.AttrAppInfo{})
			atis[v.Service.Key] = ati
		} else {
			appinfos := make([]*factory.AttrAppInfo, 0)
			appinfos = append(appinfos, &factory.AttrAppInfo{
				Key:     appKey,
				Version: v.Service.Version,
			})
			apps := make([]cipher.PubKey, 0)
			appsKey, err := cipher.PubKeyFromHex(v.Service.Key)
			if err != nil {
				continue
			}
			apps = append(apps, appsKey)
			info := &factory.AttrNodeInfo{
				Node:     nodeKey,
				Apps:     apps,
				Location: v.Node.Location,
				Version:  v.Node.Version,
				AppInfos: appinfos,
			}
			atis[v.Service.Key] = info
		}
	}
	count, err := engine.Join("INNER", "service", "service.node_id = node.id").
		Join("INNER", "attributes", "attributes.service_id = service.id").
		In("attributes.name", attr).Count(new(Node))
	if err != nil {
		return
	}
	result = &factory.AttrNodesInfo{
		Nodes: make([]*factory.AttrNodeInfo, 0),
		Count: count,
	}
	for _, v := range atis {
		result.Nodes = append(result.Nodes, v)
	}
	return
}

type NodeAndService struct {
	Node    `xorm:"extends"`
	Service `xorm:"extends"`
}

func FindServiceAddresses(keys []cipher.PubKey, exclude cipher.PubKey) (result []*factory.ServiceInfo) {
	appKeys := make([]string, len(keys))
	for _, v := range keys {
		appKeys = append(appKeys, v.Hex())
	}
	excludeNodeKey := exclude.Hex()
	ns := make([]NodeAndService, 0)
	err := engine.Join("INNER", "service", "service.node_id = node.id").
		Where("node.key != ?", excludeNodeKey).In("service.key", appKeys).Find(&ns)
	if err != nil {
		return
	}
	ss := make(map[string][]*factory.NodeInfo)
	for _, v := range ns {
		nodeKey, err := cipher.PubKeyFromHex(v.Node.Key)
		if err != nil {
			continue
		}
		node := &factory.NodeInfo{
			PubKey:  nodeKey,
			Address: v.Node.ServiceAddress,
		}
		s, ok := ss[v.Service.Key]
		if ok {
			s = append(s, node)
			ss[v.Service.Key] = s
		} else {
			nodes := make([]*factory.NodeInfo, 0)
			nodes = append(nodes, node)
			ss[v.Service.Key] = nodes
		}
	}
	result = make([]*factory.ServiceInfo, 0)
	for k, v := range ss {
		serviceKey, err := cipher.PubKeyFromHex(k)
		if err != nil {
			continue
		}
		result = append(result, &factory.ServiceInfo{
			PubKey: serviceKey,
			Nodes:  v,
		})
	}
	return
}
