package node

import (
	"encoding/binary"
	"fmt"
	"github.com/skycoin/skywire/pkg/app"
	"github.com/skycoin/skywire/pkg/router"
	"math/rand"
	"net/rpc"
	"sync"
	"time"

	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/google/uuid"

	"github.com/skycoin/skywire/pkg/cipher"
	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/transport"
)

// RPCClient represents a RPC Client implementation.
type RPCClient interface {
	Summary() (*Summary, error)

	Apps() ([]*app.Meta, error)
	StartProc(appName string, args[]string, port uint16) (router.ProcID, error)
	StopProc(pid router.ProcID) error
	ListProcs() []router.ProcInfo

	TransportTypes() ([]string, error)
	Transports(types []string, pks []cipher.PubKey, logs bool) ([]*TransportSummary, error)
	Transport(tid uuid.UUID) (*TransportSummary, error)
	AddTransport(remote cipher.PubKey, tpType string, public bool, timeout time.Duration) (*TransportSummary, error)
	RemoveTransport(tid uuid.UUID) error

	RoutingRules() ([]*RoutingEntry, error)
	RoutingRule(key routing.RouteID) (routing.Rule, error)
	AddRoutingRule(rule routing.Rule) (routing.RouteID, error)
	SetRoutingRule(key routing.RouteID, rule routing.Rule) error
	RemoveRoutingRule(key routing.RouteID) error

	Loops() ([]LoopInfo, error)
}

// RPCClient provides methods to call an RPC Server.
// It implements RPCClient
type rpcClient struct {
	client *rpc.Client
	prefix string
}

// NewRPCClient creates a new RPCClient.
func NewRPCClient(rc *rpc.Client, prefix string) RPCClient {
	return &rpcClient{client: rc, prefix: prefix}
}

// Call calls the internal rpc.Client with the serviceMethod arg prefixed.
func (rc *rpcClient) Call(method string, args, reply interface{}) error {
	return rc.client.Call(rc.prefix+"."+method, args, reply)
}

// Summary calls Summary.
func (rc *rpcClient) Summary() (*Summary, error) {
	out := new(Summary)
	err := rc.Call("Summary", &struct{}{}, out)
	return out, err
}

// StartProc starts a new process of an app with given configuration
func (rc *rpcClient) StartProc(appName string, args []string, port uint16) (router.ProcID, error) {
	var proc router.ProcID
	err := rc.Call("StartProc", &StartProcIn{
		appName: appName,
		args: args,
		port: port,
	}, &proc)

	return proc, err
}

// StopProc stops process by it's ID
func (rc *rpcClient) StopProc(pid router.ProcID) error {
	return rc.Call("StopProc", &pid, struct {}{})
}

// ListProcs list all the processes handled by node
func (rc *rpcClient) ListProcs() ([]router.ProcInfo, error) {
	var procsInfo []router.ProcInfo
	 err := rc.Call("ListProcs", &struct {}{}, &procsInfo)
	return procsInfo, err
}

// TransportTypes calls TransportTypes.
func (rc *rpcClient) TransportTypes() ([]string, error) {
	var types []string
	err := rc.Call("TransportTypes", &struct{}{}, &types)
	return types, err
}

// Transports calls Transports.
func (rc *rpcClient) Transports(types []string, pks []cipher.PubKey, logs bool) ([]*TransportSummary, error) {
	var transports []*TransportSummary
	err := rc.Call("Transports", &TransportsIn{
		FilterTypes:   types,
		FilterPubKeys: pks,
		ShowLogs:      logs,
	}, &transports)
	return transports, err
}

// Transport calls Transport.
func (rc *rpcClient) Transport(tid uuid.UUID) (*TransportSummary, error) {
	var summary TransportSummary
	err := rc.Call("Transport", &tid, &summary)
	return &summary, err
}

// AddTransport calls AddTransport.
func (rc *rpcClient) AddTransport(remote cipher.PubKey, tpType string, public bool, timeout time.Duration) (*TransportSummary, error) {
	var summary TransportSummary
	err := rc.Call("AddTransport", &AddTransportIn{
		RemotePK: remote,
		TpType:   tpType,
		Public:   public,
		Timeout:  timeout,
	}, &summary)
	return &summary, err
}

// RemoveTransport calls RemoveTransport.
func (rc *rpcClient) RemoveTransport(tid uuid.UUID) error {
	return rc.Call("RemoveTransport", &tid, &struct{}{})
}

// RoutingRules calls RoutingRules.
func (rc *rpcClient) RoutingRules() ([]*RoutingEntry, error) {
	var entries []*RoutingEntry
	err := rc.Call("RoutingRules", &struct{}{}, &entries)
	return entries, err
}

// RoutingRule calls RoutingRule.
func (rc *rpcClient) RoutingRule(key routing.RouteID) (routing.Rule, error) {
	var rule routing.Rule
	err := rc.Call("RoutingRule", &key, &rule)
	return rule, err
}

// AddRoutingRule calls AddRoutingRule.
func (rc *rpcClient) AddRoutingRule(rule routing.Rule) (routing.RouteID, error) {
	var tid routing.RouteID
	err := rc.Call("AddRoutingRule", &rule, &tid)
	return tid, err
}

// SetRoutingRule calls SetRoutingRule.
func (rc *rpcClient) SetRoutingRule(key routing.RouteID, rule routing.Rule) error {
	return rc.Call("SetRoutingRule", &RoutingEntry{Key: key, Value: rule}, &struct{}{})
}

// RemoveRoutingRule calls RemoveRoutingRule.
func (rc *rpcClient) RemoveRoutingRule(key routing.RouteID) error {
	return rc.Call("RemoveRoutingRule", &key, &struct{}{})
}

// Loops calls Loops.
func (rc *rpcClient) Loops() ([]LoopInfo, error) {
	var loops []LoopInfo
	err := rc.Call("Loops", &struct{}{}, &loops)
	return loops, err
}

// MockRPCClient mocks RPCClient.
type mockRPCClient struct {
	s       *Summary
	tpTypes []string
	rt      routing.Table
	sync.RWMutex
}

// NewMockRPCClient creates a new mock RPCClient.
func NewMockRPCClient(r *rand.Rand, maxTps int, maxRules int) (cipher.PubKey, RPCClient) {
	log := logging.MustGetLogger("mock-rpc-client")

	types := []string{"messaging", "native"}
	localPK, _ := cipher.GenerateKeyPair()

	log.Infof("generating mock client with: localPK(%s) maxTps(%d) maxRules(%d)", localPK, maxTps, maxRules)

	tps := make([]*TransportSummary, r.Intn(maxTps+1))
	for i := range tps {
		remotePK, _ := cipher.GenerateKeyPair()
		tps[i] = &TransportSummary{
			ID:     transport.MakeTransportID(localPK, remotePK, types[r.Int()%len(types)], true),
			Local:  localPK,
			Remote: remotePK,
			Type:   types[r.Int()%len(types)],
			Log:    new(transport.LogEntry),
		}
		log.Infof("tp[%2d]: %v", i, tps[i])
	}
	rt := routing.InMemoryRoutingTable()
	ruleExp := time.Now().Add(time.Hour * 24)
	for i := 0; i < r.Intn(maxRules+1); i++ {
		remotePK, _ := cipher.GenerateKeyPair()
		var lpRaw, rpRaw [2]byte
		r.Read(lpRaw[:])
		r.Read(rpRaw[:])
		lp := binary.BigEndian.Uint16(lpRaw[:])
		rp := binary.BigEndian.Uint16(rpRaw[:])
		fwdRule := routing.ForwardRule(ruleExp, routing.RouteID(r.Uint32()), uuid.New())
		fwdRID, err := rt.AddRule(fwdRule)
		if err != nil {
			panic(err)
		}
		appRule := routing.AppRule(ruleExp, fwdRID, remotePK, rp, lp)
		appRID, err := rt.AddRule(appRule)
		if err != nil {
			panic(err)
		}
		log.Infof("rt[%2da]: %v %v", i, fwdRID, fwdRule.Summary().ForwardFields)
		log.Infof("rt[%2db]: %v %v", i, appRID, appRule.Summary().AppFields)
	}
	log.Printf("rtCount: %d", rt.Count())
	return localPK, &mockRPCClient{
		s: &Summary{
			PubKey:          localPK,
			NodeVersion:     Version,
			AppProtoVersion: supportedProtocolVersion,
			//Apps: []router.AppInfo{
			//	{
			//		Meta:   app.Meta{AppName: "foo", AppVersion: "1.0", ProtocolVersion: app.ProtocolVersion, Host: localPK},
			//		State:  router.AppState{Running: false, Loops: 2},
			//		Config: router.AppConfig{AutoStart: false, Port: 2},
			//	},
			//	{
			//		Meta:   app.Meta{AppName: "bar", AppVersion: "2.0", ProtocolVersion: app.ProtocolVersion, Host: localPK},
			//		State:  router.AppState{Running: false, Loops: 3},
			//		Config: router.AppConfig{AutoStart: false, Port: 3},
			//	},
			//},
			Transports:  tps,
			RoutesCount: rt.Count(),
		},
		tpTypes: types,
		rt:      rt,
	}
}

func (mc *mockRPCClient) do(write bool, f func() error) error {
	if write {
		mc.Lock()
		defer mc.Unlock()
	} else {
		mc.RLock()
		defer mc.RUnlock()
	}
	return f()
}

// Summary implements RPCClient.
func (mc *mockRPCClient) Summary() (*Summary, error) {
	var out Summary
	err := mc.do(false, func() error {
		out = *mc.s
		copy(out.Apps, mc.s.Apps)
		copy(out.Transports, mc.s.Transports)
		out.RoutesCount = mc.s.RoutesCount
		return nil
	})
	return &out, err
}

// StartProc starts a new process of an app with given configuration
func (mc *mockRPCClient) StartProc(appName string, args []string, port uint16) (router.ProcID, error) {
	var proc router.ProcID
	err := rc.Call("StartProc", &StartProcIn{
		appName: appName,
		args: args,
		port: port,
	}, &proc)

	return proc, err
}

// StopProc stops process by it's ID
func (mc *mockRPCClient) StopProc(pid router.ProcID) error {
	return rc.Call("StopProc", &pid, struct {}{})
}

// ListProcs list all the processes handled by node
func (mc *mockRPCClient) ListProcs() ([]router.ProcInfo, error) {
	var procsInfo []router.ProcInfo
	err := rc.Call("ListProcs", &struct {}{}, &procsInfo)
	return procsInfo, err
}

// TransportTypes implements RPCClient.
func (mc *mockRPCClient) TransportTypes() ([]string, error) {
	return mc.tpTypes, nil
}

// Transports implements RPCClient.
func (mc *mockRPCClient) Transports(types []string, pks []cipher.PubKey, logs bool) ([]*TransportSummary, error) {
	var summaries []*TransportSummary
	err := mc.do(false, func() error {
		for _, tp := range mc.s.Transports {
			if types != nil {
				for _, reqT := range types {
					if tp.Type == reqT {
						goto TypeOK
					}
				}
				continue
			}
		TypeOK:
			if pks != nil {
				for _, reqPK := range pks {
					if tp.Remote == reqPK || tp.Local == reqPK {
						goto PubKeyOK
					}
				}
				continue
			}
		PubKeyOK:
			if !logs {
				temp := *tp
				temp.Log = nil
				summaries = append(summaries, &temp)
			} else {
				summaries = append(summaries, &(*tp))
			}
		}
		return nil
	})
	return summaries, err
}

// Transport implements RPCClient.
func (mc *mockRPCClient) Transport(tid uuid.UUID) (*TransportSummary, error) {
	var summary TransportSummary
	err := mc.do(false, func() error {
		for _, tp := range mc.s.Transports {
			if tp.ID == tid {
				summary = *tp
				return nil
			}
		}
		return fmt.Errorf("transport of id '%s' is not found", tid)
	})
	return &summary, err
}

// AddTransport implements RPCClient.
func (mc *mockRPCClient) AddTransport(remote cipher.PubKey, tpType string, public bool, _ time.Duration) (*TransportSummary, error) {
	summary := &TransportSummary{
		ID:     transport.MakeTransportID(mc.s.PubKey, remote, tpType, public),
		Local:  mc.s.PubKey,
		Remote: remote,
		Type:   tpType,
		Log:    new(transport.LogEntry),
	}
	return summary, mc.do(true, func() error {
		mc.s.Transports = append(mc.s.Transports, summary)
		return nil
	})
}

// RemoveTransport implements RPCClient.
func (mc *mockRPCClient) RemoveTransport(tid uuid.UUID) error {
	return mc.do(true, func() error {
		for i, tp := range mc.s.Transports {
			if tp.ID == tid {
				mc.s.Transports = append(mc.s.Transports[:i], mc.s.Transports[i+1:]...)
				return nil
			}
		}
		return fmt.Errorf("transport of id '%s' is not found", tid)
	})
}

// RoutingRules implements RPCClient.
func (mc *mockRPCClient) RoutingRules() ([]*RoutingEntry, error) {
	var entries []*RoutingEntry
	err := mc.rt.RangeRules(func(routeID routing.RouteID, rule routing.Rule) (next bool) {
		entries = append(entries, &RoutingEntry{Key: routeID, Value: rule})
		return true
	})
	return entries, err
}

// RoutingRule implements RPCClient.
func (mc *mockRPCClient) RoutingRule(key routing.RouteID) (routing.Rule, error) {
	return mc.rt.Rule(key)
}

// AddRoutingRule implements RPCClient.
func (mc *mockRPCClient) AddRoutingRule(rule routing.Rule) (routing.RouteID, error) {
	return mc.rt.AddRule(rule)
}

// SetRoutingRule implements RPCClient.
func (mc *mockRPCClient) SetRoutingRule(key routing.RouteID, rule routing.Rule) error {
	return mc.rt.SetRule(key, rule)
}

// RemoveRoutingRule implements RPCClient.
func (mc *mockRPCClient) RemoveRoutingRule(key routing.RouteID) error {
	return mc.rt.DeleteRules(key)
}

// Loops implements RPCClient.
func (mc *mockRPCClient) Loops() ([]LoopInfo, error) {
	var loops []LoopInfo
	err := mc.rt.RangeRules(func(_ routing.RouteID, rule routing.Rule) (next bool) {
		if rule.Type() == routing.RuleApp {
			loops = append(loops, LoopInfo{AppRule: rule})
		}
		return true
	})
	if err != nil {
		return nil, err
	}
	for i, l := range loops {
		fwdRID := l.AppRule.RouteID()
		rule, err := mc.rt.Rule(fwdRID)
		if err != nil {
			return nil, err
		}
		loops[i].FwdRule = rule
	}
	return loops, nil
}
