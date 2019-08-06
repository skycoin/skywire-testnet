package visor

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/skycoin/dmsg/cipher"

	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/transport"
)

const (
	// RPCPrefix is the prefix used with all RPC calls.
	RPCPrefix = "app-node"
)

var (
	// ErrInvalidInput occurs when an input is invalid.
	ErrInvalidInput = errors.New("invalid input")

	// ErrNotImplemented occurs when a method is not implemented.
	ErrNotImplemented = errors.New("not implemented")

	// ErrNotFound is returned when a requested resource is not found.
	ErrNotFound = errors.New("not found")
)

// RPC defines RPC methods for Node.
type RPC struct {
	node *Node
}

/*
	<<< NODE SUMMARY >>>
*/

// TransportSummary summarizes a Transport.
type TransportSummary struct {
	ID      uuid.UUID           `json:"id"`
	Local   cipher.PubKey       `json:"local_pk"`
	Remote  cipher.PubKey       `json:"remote_pk"`
	Type    string              `json:"type"`
	Log     *transport.LogEntry `json:"log,omitempty"`
	IsSetup bool                `json:"is_setup"`
}

func newTransportSummary(tm *transport.Manager, tp *transport.ManagedTransport,
	includeLogs bool, isSetup bool) *TransportSummary {

	summary := &TransportSummary{
		ID:      tp.Entry.ID,
		Local:   tm.Local(),
		Remote:  tp.RemotePK(),
		Type:    tp.Type(),
		IsSetup: isSetup,
	}
	if includeLogs {
		summary.Log = tp.LogEntry
	}
	return summary
}

// Summary provides a summary of an AppNode.
type Summary struct {
	PubKey          cipher.PubKey       `json:"local_pk"`
	NodeVersion     string              `json:"node_version"`
	AppProtoVersion string              `json:"app_protocol_version"`
	Apps            []*AppState         `json:"apps"`
	Transports      []*TransportSummary `json:"transports"`
	RoutesCount     int                 `json:"routes_count"`
}

// Summary provides a summary of the AppNode.
func (r *RPC) Summary(_ *struct{}, out *Summary) error {
	var summaries []*TransportSummary
	r.node.tm.WalkTransports(func(tp *transport.ManagedTransport) bool {
		summaries = append(summaries,
			newTransportSummary(r.node.tm, tp, false, r.node.router.IsSetupTransport(tp)))
		return true
	})
	*out = Summary{
		PubKey:          r.node.config.Node.StaticPubKey,
		NodeVersion:     Version,
		AppProtoVersion: supportedProtocolVersion,
		Apps:            r.node.Apps(),
		Transports:      summaries,
		RoutesCount:     r.node.rt.Count(),
	}
	return nil
}

/*
	<<< APP MANAGEMENT >>>
*/

// Apps returns list of Apps registered on the Node.
func (r *RPC) Apps(_ *struct{}, reply *[]*AppState) error {
	*reply = r.node.Apps()
	return nil
}

// StartApp start App with provided name.
func (r *RPC) StartApp(name *string, _ *struct{}) error {
	return r.node.StartApp(*name)
}

// StopApp stops App with provided name.
func (r *RPC) StopApp(name *string, _ *struct{}) error {
	return r.node.StopApp(*name)
}

// SetAutoStartIn is input for SetAutoStart.
type SetAutoStartIn struct {
	AppName   string
	AutoStart bool
}

// SetAutoStart sets auto-start settings for an app.
func (r *RPC) SetAutoStart(in *SetAutoStartIn, _ *struct{}) error {
	return r.node.SetAutoStart(in.AppName, in.AutoStart)
}

/*
	<<< TRANSPORT MANAGEMENT >>>
*/

// TransportTypes lists all transport types supported by the Node.
func (r *RPC) TransportTypes(_ *struct{}, out *[]string) error {
	*out = r.node.tm.Factories()
	return nil
}

// TransportsIn is input for Transports.
type TransportsIn struct {
	FilterTypes   []string
	FilterPubKeys []cipher.PubKey
	ShowLogs      bool
}

// Transports lists Transports of the Node and provides a summary of each.
func (r *RPC) Transports(in *TransportsIn, out *[]*TransportSummary) error {
	typeIncluded := func(tType string) bool {
		if in.FilterTypes != nil {
			for _, ft := range in.FilterTypes {
				if tType == ft {
					return true
				}
			}
			return false
		}
		return true
	}
	pkIncluded := func(localPK, remotePK cipher.PubKey) bool {
		if in.FilterPubKeys != nil {
			for _, fpk := range in.FilterPubKeys {
				if localPK == fpk || remotePK == fpk {
					return true
				}
			}
			return false
		}
		return true
	}
	r.node.tm.WalkTransports(func(tp *transport.ManagedTransport) bool {
		if typeIncluded(tp.Type()) && pkIncluded(r.node.tm.Local(), tp.RemotePK()) {
			*out = append(*out, newTransportSummary(r.node.tm, tp, in.ShowLogs, r.node.router.IsSetupTransport(tp)))
		}
		return true
	})
	return nil
}

// Transport obtains a Transport Summary of Transport of given Transport ID.
func (r *RPC) Transport(in *uuid.UUID, out *TransportSummary) error {
	tp := r.node.tm.Transport(*in)
	if tp == nil {
		return ErrNotFound
	}
	*out = *newTransportSummary(r.node.tm, tp, true, r.node.router.IsSetupTransport(tp))
	return nil
}

// AddTransportIn is input for AddTransport.
type AddTransportIn struct {
	RemotePK cipher.PubKey
	TpType   string
	Public   bool
	Timeout  time.Duration
}

// AddTransport creates a transport for the node.
func (r *RPC) AddTransport(in *AddTransportIn, out *TransportSummary) error {
	ctx := context.Background()
	if in.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Second*20)
		defer cancel()
	}

	tp, err := r.node.tm.CreateDataTransport(ctx, in.RemotePK, in.TpType, in.Public)
	if err != nil {
		return err
	}
	*out = *newTransportSummary(r.node.tm, tp, false, r.node.router.IsSetupTransport(tp))
	return nil
}

// RemoveTransport removes a Transport from the node.
func (r *RPC) RemoveTransport(tid *uuid.UUID, _ *struct{}) error {
	return r.node.tm.DeleteTransport(*tid)
}

/*
	<<< ROUTES MANAGEMENT >>>
*/

// RoutingEntry represents an RoutingTable's entry.
type RoutingEntry struct {
	Key   routing.RouteID
	Value routing.Rule
}

// RoutingRules obtains all routing rules of the RoutingTable.
func (r *RPC) RoutingRules(_ *struct{}, out *[]*RoutingEntry) error {
	return r.node.rt.RangeRules(func(routeID routing.RouteID, rule routing.Rule) (next bool) {
		*out = append(*out, &RoutingEntry{Key: routeID, Value: rule})
		return true
	})
}

// RoutingRule obtains a routing rule of given RouteID.
func (r *RPC) RoutingRule(key *routing.RouteID, rule *routing.Rule) error {
	var err error
	*rule, err = r.node.rt.Rule(*key)
	return err
}

// AddRoutingRule adds a RoutingRule and returns a Key in which the rule is stored under.
func (r *RPC) AddRoutingRule(rule *routing.Rule, routeID *routing.RouteID) error {
	var err error
	*routeID, err = r.node.rt.AddRule(*rule)
	return err
}

// SetRoutingRule sets a routing rule.
func (r *RPC) SetRoutingRule(in *RoutingEntry, out *struct{}) error {
	return r.node.rt.SetRule(in.Key, in.Value)
}

// RemoveRoutingRule removes a RoutingRule based on given RouteID key.
func (r *RPC) RemoveRoutingRule(key *routing.RouteID, _ *struct{}) error {
	return r.node.rt.DeleteRules(*key)
}

/*
	<<< LOOPS MANAGEMENT >>>
	>>> TODO(evanlinjin): Implement.
*/

// LoopInfo is a human-understandable representation of a loop.
type LoopInfo struct {
	AppRule routing.Rule
	FwdRule routing.Rule
}

// Loops retrieves loops via rules of the routing table.
func (r *RPC) Loops(_ *struct{}, out *[]LoopInfo) error {
	var loops []LoopInfo
	err := r.node.rt.RangeRules(func(_ routing.RouteID, rule routing.Rule) (next bool) {
		if rule.Type() == routing.RuleApp {
			loops = append(loops, LoopInfo{AppRule: rule})
		}
		return true
	})
	if err != nil {
		return err
	}
	for i, l := range loops {
		fwdRID := l.AppRule.RouteID()
		rule, err := r.node.rt.Rule(fwdRID)
		if err != nil {
			return err
		}
		loops[i].FwdRule = rule
	}
	*out = loops
	return nil
}
