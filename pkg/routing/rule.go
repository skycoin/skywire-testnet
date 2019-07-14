package routing

import (
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/skycoin/dmsg/cipher"
)

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

// Expiry returns rule's expiration time.
func (r Rule) Expiry() time.Time {
	ts := binary.BigEndian.Uint64(r)
	return time.Unix(int64(ts), 0)
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
	return uuid.Must(uuid.FromBytes(r[13:]))
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
func (r Rule) RemotePort() uint16 {
	if r.Type() != RuleApp {
		panic("invalid rule")
	}
	return binary.BigEndian.Uint16(r[46:])
}

// LocalPort returns local Port for an app rule.
func (r Rule) LocalPort() uint16 {
	if r.Type() != RuleApp {
		panic("invalid rule")
	}
	return binary.BigEndian.Uint16(r[48:])
}

func (r Rule) String() string {
	if r.Type() == RuleApp {
		return fmt.Sprintf("App: <resp-rid: %d><remote-pk: %s><remote-port: %d><local-port: %d>",
			r.RouteID(), r.RemotePK(), r.RemotePort(), r.LocalPort())
	}

	return fmt.Sprintf("Forward: <next-rid: %d><next-tid: %s>", r.RouteID(), r.TransportID())
}

// RuleAppFields summarizes App fields of a RoutingRule.
type RuleAppFields struct {
	RespRID    RouteID       `json:"resp_rid"`
	RemotePK   cipher.PubKey `json:"remote_pk"`
	RemotePort uint16        `json:"remote_port"`
	LocalPort  uint16        `json:"local_port"`
}

// RuleForwardFields summarizes Forward fields of a RoutingRule.
type RuleForwardFields struct {
	NextRID RouteID   `json:"next_rid"`
	NextTID uuid.UUID `json:"next_tid"`
}

// RuleSummary provides a summary of a RoutingRule.
type RuleSummary struct {
	ExpireAt      time.Time          `json:"expire_at"`
	Type          RuleType           `json:"rule_type"`
	AppFields     *RuleAppFields     `json:"app_fields,omitempty"`
	ForwardFields *RuleForwardFields `json:"forward_fields,omitempty"`
}

// ToRule converts RoutingRuleSummary to RoutingRule.
func (rs *RuleSummary) ToRule() (Rule, error) {
	if rs.Type == RuleApp && rs.AppFields != nil && rs.ForwardFields == nil {
		f := rs.AppFields
		return AppRule(rs.ExpireAt, f.RespRID, f.RemotePK, f.RemotePort, f.LocalPort), nil
	}
	if rs.Type == RuleForward && rs.AppFields == nil && rs.ForwardFields != nil {
		f := rs.ForwardFields
		return ForwardRule(rs.ExpireAt, f.NextRID, f.NextTID), nil
	}
	return nil, errors.New("invalid routing rule summary")
}

// Summary returns the RoutingRule's summary.
func (r Rule) Summary() *RuleSummary {
	summary := RuleSummary{
		ExpireAt: r.Expiry(),
		Type:     r.Type(),
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
func AppRule(expireAt time.Time, respRoute RouteID, remotePK cipher.PubKey, remotePort, localPort uint16) Rule {
	rule := make([]byte, 13)
	if expireAt.Unix() <= time.Now().Unix() {
		binary.BigEndian.PutUint64(rule[0:], 0)
	} else {
		binary.BigEndian.PutUint64(rule[0:], uint64(expireAt.Unix()))
	}

	rule[8] = byte(RuleApp)
	binary.BigEndian.PutUint32(rule[9:], uint32(respRoute))
	rule = append(rule, remotePK[:]...)
	rule = append(rule, 0, 0, 0, 0)
	binary.BigEndian.PutUint16(rule[46:], remotePort)
	binary.BigEndian.PutUint16(rule[48:], localPort)
	return Rule(rule)
}

// ForwardRule constructs a new forward RoutingRule.
func ForwardRule(expireAt time.Time, nextRoute RouteID, nextTrID uuid.UUID) Rule {
	rule := make([]byte, 13)
	if expireAt.Unix() <= time.Now().Unix() {
		binary.BigEndian.PutUint64(rule[0:], 0)
	} else {
		binary.BigEndian.PutUint64(rule[0:], uint64(expireAt.Unix()))
	}

	rule[8] = byte(RuleForward)
	binary.BigEndian.PutUint32(rule[9:], uint32(nextRoute))
	rule = append(rule, nextTrID[:]...)
	return Rule(rule)
}
