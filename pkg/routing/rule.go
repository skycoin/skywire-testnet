package routing

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/SkycoinProject/dmsg/cipher"
)

// RuleHeaderSize represents the base size of a rule.
// All rules should be at-least this size.
// TODO(evanlinjin): Document the format of rules in comments.
const RuleHeaderSize = 13

// RuleType defines type of a routing rule
type RuleType byte

func (rt RuleType) String() string {
	switch rt {
	case RuleApp:
		return "App"
	case RuleForward:
		return "Forward"
	}

	return fmt.Sprintf("Unknown(%d)", rt)
}

const (
	// RuleApp defines App routing rule type.
	RuleApp RuleType = iota
	// RuleForward defines Forward routing rule type.
	RuleForward
)

// Rule represents a routing rule.
// There are two types of routing rules; App and Forward.
//
type Rule []byte

// KeepAlive returns rule's keep-alive timeout.
func (r Rule) KeepAlive() time.Duration {
	return time.Duration(binary.BigEndian.Uint64(r))
}

// Type returns type of a rule.
func (r Rule) Type() RuleType {
	return RuleType(r[8])
}

// RouteID returns RouteID from the rule: reverse ID for an app rule
// and next ID for a forward rule.
func (r Rule) RouteID() RouteID {
	return RouteID(binary.BigEndian.Uint32(r[9:]))
}

// SetRouteID sets RouteID for the rule: reverse ID for an app rule
// and next ID for a forward rule.
func (r Rule) SetRouteID(routeID RouteID) {
	binary.BigEndian.PutUint32(r[9:], uint32(routeID))
}

// TransportID returns next transport ID for a forward rule.
func (r Rule) TransportID() uuid.UUID {
	if r.Type() != RuleForward {
		panic("invalid rule")
	}
	return uuid.Must(uuid.FromBytes(r[13:29]))
}

// RemotePK returns remove PK for an app rule.
func (r Rule) RemotePK() cipher.PubKey {
	if r.Type() != RuleApp {
		panic("invalid rule")
	}

	pk := cipher.PubKey{}
	if err := pk.UnmarshalBinary(r[13:46]); err != nil {
		log.WithError(err).Warn("Failed to unmarshal public key")
	}
	return pk
}

// RemotePort returns remote Port for an app rule.
func (r Rule) RemotePort() Port {
	if r.Type() != RuleApp {
		panic("invalid rule")
	}
	return Port(binary.BigEndian.Uint16(r[46:]))
}

// LocalPort returns local Port for an app rule.
func (r Rule) LocalPort() Port {
	if r.Type() != RuleApp {
		panic("invalid rule")
	}
	return Port(binary.BigEndian.Uint16(r[48:]))
}

// RequestRouteID returns route ID which will be used to register this rule within
// the visor node.
func (r Rule) RequestRouteID() RouteID {
	return RouteID(binary.BigEndian.Uint32(r[50:]))
}

// SetRequestRouteID sets the route ID which will be used to register this rule within
// the visor node.
func (r Rule) SetRequestRouteID(id RouteID) {
	binary.BigEndian.PutUint32(r[50:], uint32(id))
}

func (r Rule) String() string {
	switch r.Type() {
	case RuleApp:
		return fmt.Sprintf("APP(keyRtID:%d, resRtID:%d, rPK:%s, rPort:%d, lPort:%d)",
			r.RequestRouteID(), r.RouteID(), r.RemotePK(), r.RemotePort(), r.LocalPort())
	case RuleForward:
		return fmt.Sprintf("FWD(keyRtID:%d, nxtRtID:%d, nxtTpID:%s)",
			r.RequestRouteID(), r.RouteID(), r.TransportID())
	default:
		return "invalid rule"
	}
}

//func (r Rule) MarshalJSON() ([]byte, error) {
//	return json.Marshal(r.String())
//}

// RuleAppFields summarizes App fields of a RoutingRule.
type RuleAppFields struct {
	RespRID    RouteID       `json:"resp_rid"`
	RemotePK   cipher.PubKey `json:"remote_pk"`
	RemotePort Port          `json:"remote_port"`
	LocalPort  Port          `json:"local_port"`
}

// RuleForwardFields summarizes Forward fields of a RoutingRule.
type RuleForwardFields struct {
	NextRID RouteID   `json:"next_rid"`
	NextTID uuid.UUID `json:"next_tid"`
}

// RuleSummary provides a summary of a RoutingRule.
type RuleSummary struct {
	KeepAlive      time.Duration      `json:"keep_alive"`
	Type           RuleType           `json:"rule_type"`
	AppFields      *RuleAppFields     `json:"app_fields,omitempty"`
	ForwardFields  *RuleForwardFields `json:"forward_fields,omitempty"`
	RequestRouteID RouteID            `json:"request_route_id"`
}

// ToRule converts RoutingRuleSummary to RoutingRule.
func (rs *RuleSummary) ToRule() (Rule, error) {
	if rs.Type == RuleApp && rs.AppFields != nil && rs.ForwardFields == nil {
		f := rs.AppFields
		return AppRule(rs.KeepAlive, rs.RequestRouteID, f.RespRID, f.RemotePK, f.LocalPort, f.RemotePort), nil
	}
	if rs.Type == RuleForward && rs.AppFields == nil && rs.ForwardFields != nil {
		f := rs.ForwardFields
		return ForwardRule(rs.KeepAlive, f.NextRID, f.NextTID, rs.RequestRouteID), nil
	}
	return nil, errors.New("invalid routing rule summary")
}

// Summary returns the RoutingRule's summary.
func (r Rule) Summary() *RuleSummary {
	summary := RuleSummary{
		KeepAlive:      r.KeepAlive(),
		Type:           r.Type(),
		RequestRouteID: r.RequestRouteID(),
	}
	if summary.Type == RuleApp {
		summary.AppFields = &RuleAppFields{
			RespRID:    r.RouteID(),
			RemotePK:   r.RemotePK(),
			RemotePort: r.RemotePort(),
			LocalPort:  r.LocalPort(),
		}
	} else {
		summary.ForwardFields = &RuleForwardFields{
			NextRID: r.RouteID(),
			NextTID: r.TransportID(),
		}
	}
	return &summary
}

// AppRule constructs a new consume RoutingRule.
func AppRule(keepAlive time.Duration, reqRoute, respRoute RouteID, remotePK cipher.PubKey, localPort, remotePort Port) Rule {
	rule := make([]byte, RuleHeaderSize)

	if keepAlive < 0 {
		keepAlive = 0
	}

	binary.BigEndian.PutUint64(rule, uint64(keepAlive))

	rule[8] = byte(RuleApp)
	binary.BigEndian.PutUint32(rule[9:], uint32(respRoute))
	rule = append(rule, remotePK[:]...)
	rule = append(rule, bytes.Repeat([]byte{0}, 8)...)
	binary.BigEndian.PutUint16(rule[46:], uint16(remotePort))
	binary.BigEndian.PutUint16(rule[48:], uint16(localPort))
	binary.BigEndian.PutUint32(rule[50:], uint32(reqRoute))
	return rule
}

// ForwardRule constructs a new forward RoutingRule.
func ForwardRule(keepAlive time.Duration, nextRoute RouteID, nextTrID uuid.UUID, requestRouteID RouteID) Rule {
	rule := make([]byte, RuleHeaderSize)

	if keepAlive < 0 {
		keepAlive = 0
	}

	binary.BigEndian.PutUint64(rule, uint64(keepAlive))

	rule[8] = byte(RuleForward)
	binary.BigEndian.PutUint32(rule[9:], uint32(nextRoute))
	rule = append(rule, nextTrID[:]...)
	rule = append(rule, bytes.Repeat([]byte{0}, 25)...)
	binary.BigEndian.PutUint32(rule[50:], uint32(requestRouteID))
	return rule
}
