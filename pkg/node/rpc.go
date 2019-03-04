package node

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/skycoin/skywire/pkg/cipher"

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

// RPCClient represents a RPC Client implementation.
type RPCClient interface {
	Summary() (*Summary, error)

	Apps() ([]*AppState, error)
	StartApp(appName string) error
	StopApp(appName string) error
	SetAutoStart(appName string, autostart bool) error

	TransportTypes() ([]string, error)
	Transports(types []string, pks []cipher.PubKey, logs bool) ([]*TransportSummary, error)
	Transport(tid uuid.UUID) (*TransportSummary, error)
	AddTransport(remote cipher.PubKey, tpType string, public bool) (*TransportSummary, error)
	RemoveTransport(tid uuid.UUID) error

	RoutingRules() ([]*RoutingEntry, error)
	RoutingRule(key routing.RouteID) (routing.Rule, error)
	AddRoutingRule(rule routing.Rule) (routing.RouteID, error)
	SetRoutingRule(key routing.RouteID, rule routing.Rule) error
	RemoveRoutingRule(key routing.RouteID) error
}

/*
	<<< NODE SUMMARY >>>
*/

// TransportSummary summarizes a Transport.
type TransportSummary struct {
	ID     uuid.UUID           `json:"id"`
	Local  cipher.PubKey       `json:"local_pk"`
	Remote cipher.PubKey       `json:"remote_pk"`
	Type   string              `json:"type"`
	Log    *transport.LogEntry `json:"log,omitempty"`
}

func newTransportSummary(tp *transport.ManagedTransport, includeLogs bool) *TransportSummary {
	summary := TransportSummary{
		ID:     tp.ID,
		Local:  tp.Local(),
		Remote: tp.Remote(),
		Type:   tp.Type(),
	}
	if includeLogs {
		summary.Log = tp.LogEntry
	}
	return &summary
}

// Summary provides a summary of an AppNode.
type Summary struct {
	PubKey      cipher.PubKey       `json:"local_pk"`
	Apps        []*AppState         `json:"apps"`
	Transports  []*TransportSummary `json:"transports"`
	RoutesCount int                 `json:"routes_count"`
}

// Summary provides a summary of the AppNode.
func (r *RPC) Summary(_ *struct{}, out *Summary) error {
	var summaries []*TransportSummary
	r.node.tm.WalkTransports(func(tp *transport.ManagedTransport) bool {
		summaries = append(summaries, newTransportSummary(tp, false))
		return true
	})
	*out = Summary{
		PubKey:      r.node.config.Node.StaticPubKey,
		Apps:        r.node.Apps(),
		Transports:  summaries,
		RoutesCount: r.node.rt.Count(),
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
		if typeIncluded(tp.Type()) && pkIncluded(tp.Local(), tp.Remote()) {
			*out = append(*out, newTransportSummary(tp, in.ShowLogs))
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
	*out = *newTransportSummary(tp, true)
	return nil
}

// AddTransportIn is input for AddTransport.
type AddTransportIn struct {
	RemotePK cipher.PubKey
	TpType   string
	Public   bool
}

// AddTransport creates a transport for the node.
func (r *RPC) AddTransport(in *AddTransportIn, out *TransportSummary) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	tp, err := r.node.tm.CreateTransport(ctx, in.RemotePK, in.TpType, in.Public)
	if err != nil {
		return err
	}
	*out = *newTransportSummary(tp, false)
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
