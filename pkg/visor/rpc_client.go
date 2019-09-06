package visor

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"net/http"
	"net/rpc"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/app"
	"github.com/skycoin/skywire/pkg/router"
	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/transport"
)

// RPCClient represents a RPC Client implementation.
type RPCClient interface {
	Summary() (*Summary, error)
	Exec(command string) ([]byte, error)

	Health() (*HealthInfo, error)
	Uptime() (float64, error)

	Apps() ([]*AppState, error)
	StartApp(appName string) error
	StopApp(appName string) error
	SetAutoStart(appName string, autostart bool) error
	LogsSince(timestamp time.Time, appName string) ([]string, error)

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

// Health calls Health
func (rc *rpcClient) Health() (*HealthInfo, error) {
	hi := &HealthInfo{}
	err := rc.Call("Health", &struct{}{}, hi)
	return hi, err
}

// Uptime calls Uptime
func (rc *rpcClient) Uptime() (float64, error) {
	var out float64
	err := rc.Call("Uptime", &struct{}{}, &out)
	return out, err
}

// Exec calls Exec.
func (rc *rpcClient) Exec(command string) ([]byte, error) {
	output := make([]byte, 0)
	err := rc.Call("Exec", &command, &output)
	return output, err
}

// Apps calls Apps.
func (rc *rpcClient) Apps() ([]*AppState, error) {
	states := make([]*AppState, 0)
	err := rc.Call("Apps", &struct{}{}, &states)
	return states, err
}

// StartApp calls StartApp.
func (rc *rpcClient) StartApp(appName string) error {
	return rc.Call("StartApp", &appName, &struct{}{})
}

// StopApp calls StopApp.
func (rc *rpcClient) StopApp(appName string) error {
	return rc.Call("StopApp", &appName, &struct{}{})
}

// SetAutoStart calls SetAutoStart.
func (rc *rpcClient) SetAutoStart(appName string, autostart bool) error {
	return rc.Call("SetAutoStart", &SetAutoStartIn{
		AppName:   appName,
		AutoStart: autostart,
	}, &struct{}{})
}

// LogsSince calls LogsSince
func (rc *rpcClient) LogsSince(timestamp time.Time, appName string) ([]string, error) {
	res := make([]string, 0)

	err := rc.Call("LogsSince", &AppLogsRequest{
		TimeStamp: timestamp,
		AppName:   appName,
	}, &res)
	if err != nil {
		return nil, err
	}

	return res, nil
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
	startedAt time.Time
	s         *Summary
	tpTypes   []string
	rt        routing.Table
	appls     app.LogStore
	sync.RWMutex
}

// NewMockRPCClient creates a new mock RPCClient.
func NewMockRPCClient(r *rand.Rand, maxTps int, maxRules int) (cipher.PubKey, RPCClient, error) {
	log := logging.MustGetLogger("mock-rpc-client")

	types := []string{"messaging", "native"}
	localPK, _ := cipher.GenerateKeyPair()

	log.Infof("generating mock client with: localPK(%s) maxTps(%d) maxRules(%d)", localPK, maxTps, maxRules)

	tps := make([]*TransportSummary, r.Intn(maxTps+1))
	for i := range tps {
		remotePK, _ := cipher.GenerateKeyPair()
		tps[i] = &TransportSummary{
			ID:     transport.MakeTransportID(localPK, remotePK, types[r.Int()%len(types)]),
			Local:  localPK,
			Remote: remotePK,
			Type:   types[r.Int()%len(types)],
			Log:    new(transport.LogEntry),
		}
		log.Infof("tp[%2d]: %v", i, tps[i])
	}
	rt := routing.InMemoryRoutingTable()
	ruleKeepAlive := router.DefaultRouteKeepAlive
	for i := 0; i < r.Intn(maxRules+1); i++ {
		remotePK, _ := cipher.GenerateKeyPair()
		var lpRaw, rpRaw [2]byte
		if _, err := r.Read(lpRaw[:]); err != nil {
			return cipher.PubKey{}, nil, err
		}
		if _, err := r.Read(rpRaw[:]); err != nil {
			return cipher.PubKey{}, nil, err
		}
		lp := routing.Port(binary.BigEndian.Uint16(lpRaw[:]))
		rp := routing.Port(binary.BigEndian.Uint16(rpRaw[:]))
		fwdRID, err := rt.AddRule(nil)
		if err != nil {
			panic(err)
		}
		fwdRule := routing.IntermediaryForwardRule(ruleKeepAlive, fwdRID, routing.RouteID(r.Uint32()), uuid.New())
		if err := rt.SetRule(fwdRID, fwdRule); err != nil {
			panic(err)
		}
		appRID, err := rt.AddRule(nil)
		if err != nil {
			panic(err)
		}
		appRule := routing.ConsumeRule(ruleKeepAlive, appRID, remotePK, lp, rp)
		if err := rt.SetRule(appRID, appRule); err != nil {
			panic(err)
		}
		log.Infof("rt[%2da]: %v %v", i, fwdRID, fwdRule.Summary().ForwardFields)
		log.Infof("rt[%2db]: %v %v", i, appRID, appRule.Summary().ConsumeFields)
	}
	log.Printf("rtCount: %d", rt.Count())
	client := &mockRPCClient{
		s: &Summary{
			PubKey:          localPK,
			NodeVersion:     Version,
			AppProtoVersion: supportedProtocolVersion,
			Apps: []*AppState{
				{Name: "foo.v1.0", AutoStart: false, Port: 10},
				{Name: "bar.v2.0", AutoStart: false, Port: 20},
			},
			Transports:  tps,
			RoutesCount: rt.Count(),
		},
		tpTypes:   types,
		rt:        rt,
		startedAt: time.Now(),
	}
	return localPK, client, nil
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
		for _, app := range mc.s.Apps {
			out.Apps = append(out.Apps, &(*app))
		}
		for _, tp := range mc.s.Transports {
			out.Transports = append(out.Transports, &(*tp))
		}
		out.RoutesCount = mc.s.RoutesCount
		return nil
	})
	return &out, err
}

// Health implements RPCClient
func (mc *mockRPCClient) Health() (*HealthInfo, error) {
	hi := &HealthInfo{
		TransportDiscovery: http.StatusOK,
		RouteFinder:        http.StatusOK,
		SetupNode:          http.StatusOK,
	}

	return hi, nil
}

// Uptime implements RPCClient
func (mc *mockRPCClient) Uptime() (float64, error) {
	return time.Since(mc.startedAt).Seconds(), nil
}

// Exec implements RPCClient.
func (mc *mockRPCClient) Exec(command string) ([]byte, error) {
	return []byte("mock"), nil
}

// Apps implements RPCClient.
func (mc *mockRPCClient) Apps() ([]*AppState, error) {
	var apps []*AppState
	err := mc.do(false, func() error {
		for _, app := range mc.s.Apps {
			apps = append(apps, &(*app))
		}
		return nil
	})
	return apps, err
}

// StartApp implements RPCClient.
func (*mockRPCClient) StartApp(string) error {
	return nil
}

// StopApp implements RPCClient.
func (*mockRPCClient) StopApp(string) error {
	return nil
}

// SetAutoStart implements RPCClient.
func (mc *mockRPCClient) SetAutoStart(appName string, autostart bool) error {
	return mc.do(true, func() error {
		for _, app := range mc.s.Apps {
			if app.Name == appName {
				app.AutoStart = autostart
				return nil
			}
		}
		return fmt.Errorf("app of name '%s' does not exist", appName)
	})
}

// LogsSince implements RPCClient. Manually set (*mockRPPClient).appls before calling this function
func (mc *mockRPCClient) LogsSince(timestamp time.Time, _ string) ([]string, error) {
	return mc.appls.LogsSince(timestamp)
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
		ID:     transport.MakeTransportID(mc.s.PubKey, remote, tpType),
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
		if rule.Type() == routing.RuleConsume {
			loops = append(loops, LoopInfo{AppRule: rule})
		}
		return true
	})
	if err != nil {
		return nil, err
	}
	for i, l := range loops {
		fwdRID := l.AppRule.NextRouteID()
		rule, err := mc.rt.Rule(fwdRID)
		if err != nil {
			return nil, err
		}
		loops[i].FwdRule = rule
	}
	return loops, nil
}
