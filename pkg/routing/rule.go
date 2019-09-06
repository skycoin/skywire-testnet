package routing

import (
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/skycoin/dmsg/cipher"
)

// RuleHeaderSize represents the base size of a rule.
// All rules should be at-least this size.
// TODO(evanlinjin): Document the format of rules in comments.
const (
	RuleHeaderSize      = 8 + 1 + 4
	pkSize              = len(cipher.PubKey{})
	uuidSize            = len(uuid.UUID{})
	routeDescriptorSize = pkSize*2 + 2*2

	invalidRule = "invalid rule"
)

// RuleType defines type of a routing rule
type RuleType byte

func (rt RuleType) String() string {
	switch rt {
	case RuleConsume:
		return "Consume"
	case RuleForward:
		return "Forward"
	case RuleIntermediaryForward:
		return "IntermediaryForward"
	}

	return fmt.Sprintf("Unknown(%d)", rt)
}

const (
	// RuleConsume represents a hop to the route's destination node.
	// A packet referencing this rule is to be consumed localy.
	RuleConsume = RuleType(0)

	// ForwardRule represents a hop from the route's source node.
	// A packet referencing this rule is to be sent to a remote node.
	RuleForward = RuleType(1)

	// RuleIntermediaryForward represents a hop which is not from the route's source,
	// nor to the route's destination.
	RuleIntermediaryForward = RuleType(2)
)

// Rule represents a routing rule.
// There are two types of routing rules; App and Forward.
//
type Rule []byte

func (r Rule) assertLen(l int) {
	if len(r) < l {
		panic("bad rule length")
	}
}

// KeepAlive returns rule's keep-alive timeout.
func (r Rule) KeepAlive() time.Duration {
	r.assertLen(RuleHeaderSize)
	return time.Duration(binary.BigEndian.Uint64(r[0:8]))
}

func (r Rule) SetKeepAlive(keepAlive time.Duration) {
	r.assertLen(RuleHeaderSize)

	if keepAlive < 0 {
		keepAlive = 0
	}

	binary.BigEndian.PutUint64(r[0:8], uint64(keepAlive))
}

// Type returns type of a rule.
func (r Rule) Type() RuleType {
	r.assertLen(RuleHeaderSize)
	return RuleType(r[8])
}

func (r Rule) SetType(t RuleType) {
	r.assertLen(RuleHeaderSize)
	r[8] = byte(t)
}

// RouteID returns RouteID from the rule: reverse ID for an app rule
// and next ID for a forward rule.
func (r Rule) KeyRouteID() RouteID {
	r.assertLen(RuleHeaderSize)
	return RouteID(binary.BigEndian.Uint32(r[8+1 : 8+1+4]))
}

func (r Rule) SetKeyRouteID(id RouteID) {
	r.assertLen(RuleHeaderSize)
	binary.BigEndian.PutUint32(r[8+1:8+1+4], uint32(id))
}

// Body returns Body from the rule.
func (r Rule) Body() []byte {
	r.assertLen(RuleHeaderSize)
	return append(r[:0:0], r[RuleHeaderSize:]...)
}

func (r Rule) RouteDescriptor() RouteDescriptor {
	switch t := r.Type(); t {
	case RuleConsume, RuleForward:
		r.assertLen(RuleHeaderSize + routeDescriptorSize)

		var desc RouteDescriptor
		copy(desc[:], r[RuleHeaderSize:])
		return desc

	default:
		panic(fmt.Sprintf("%v: %v", invalidRule, t.String()))
	}
}

func (r Rule) NextRouteID() RouteID {
	offset := RuleHeaderSize
	switch t := r.Type(); t {
	case RuleForward:
		offset += routeDescriptorSize
		fallthrough

	case RuleIntermediaryForward:
		r.assertLen(offset + 4)
		return RouteID(binary.BigEndian.Uint32(r[offset : offset+4]))

	default:
		panic(fmt.Sprintf("%v: %v", invalidRule, t.String()))
	}
}

func (r Rule) SetNextRouteID(id RouteID) {
	offset := RuleHeaderSize
	switch t := r.Type(); t {
	case RuleForward:
		offset += routeDescriptorSize
		fallthrough

	case RuleIntermediaryForward:
		r.assertLen(offset + 4)
		binary.BigEndian.PutUint32(r[offset:offset+4], uint32(id))

	default:
		panic(fmt.Sprintf("%v: %v", invalidRule, t.String()))
	}
}

// NextTransportID returns next transport ID for a forward rule.
func (r Rule) NextTransportID() uuid.UUID {
	offset := RuleHeaderSize + 4
	switch t := r.Type(); t {
	case RuleForward:
		offset += routeDescriptorSize
		fallthrough

	case RuleIntermediaryForward:
		r.assertLen(offset + 4)
		return uuid.Must(uuid.FromBytes(r[offset : offset+uuidSize]))

	default:
		panic(fmt.Sprintf("%v: %v", invalidRule, t.String()))
	}
}

func (r Rule) SetNextTransportID(id uuid.UUID) {
	offset := RuleHeaderSize + 4
	switch t := r.Type(); t {
	case RuleForward:
		offset += routeDescriptorSize
		fallthrough

	case RuleIntermediaryForward:
		r.assertLen(offset + 4)
		copy(r[offset:offset+uuidSize], id[:])

	default:
		panic(fmt.Sprintf("%v: %v", invalidRule, t.String()))
	}
}

func (r Rule) SetSrcPK(pk cipher.PubKey) {
	switch t := r.Type(); t {
	case RuleConsume, RuleForward:
		copy(r[RuleHeaderSize:RuleHeaderSize+pkSize], pk[:])

	default:
		panic(fmt.Sprintf("%v: %v", invalidRule, t.String()))
	}
}

func (r Rule) SetDstPK(pk cipher.PubKey) {
	switch t := r.Type(); t {
	case RuleConsume, RuleForward:
		copy(r[RuleHeaderSize+pkSize:RuleHeaderSize+pkSize*2], pk[:])

	default:
		panic(fmt.Sprintf("%v: %v", invalidRule, t.String()))
	}
}

func (r Rule) SetSrcPort(port Port) {
	switch t := r.Type(); t {
	case RuleConsume, RuleForward:
		binary.BigEndian.PutUint16(r[RuleHeaderSize+pkSize*2:RuleHeaderSize+pkSize*2+2], uint16(port))

	default:
		panic(fmt.Sprintf("%v: %v", invalidRule, t.String()))
	}
}

func (r Rule) SetDstPort(port Port) {
	switch t := r.Type(); t {
	case RuleConsume, RuleForward:
		binary.BigEndian.PutUint16(r[RuleHeaderSize+pkSize*2+2:RuleHeaderSize+pkSize*2+2*2], uint16(port))

	default:
		panic(fmt.Sprintf("%v: %v", invalidRule, t.String()))
	}
}

type RouteDescriptor [routeDescriptorSize]byte

func (d RouteDescriptor) SrcPK() cipher.PubKey {
	var pk cipher.PubKey
	copy(pk[:], d[0:pkSize])
	return pk
}

func (d RouteDescriptor) DstPK() cipher.PubKey {
	var pk cipher.PubKey
	copy(pk[:], d[pkSize:pkSize*2])
	return pk
}

func (d RouteDescriptor) SetDskPK(pk cipher.PubKey) {
	copy(d[pkSize:pkSize*2], pk[:])
}

func (d RouteDescriptor) SrcPort() Port {
	return Port(binary.BigEndian.Uint16(d[pkSize*2 : pkSize*2+2]))
}

func (d RouteDescriptor) SetSrcPort(port Port) {
	binary.BigEndian.PutUint16(d[pkSize*2:pkSize*2+2], uint16(port))
}

func (d RouteDescriptor) DstPort() Port {
	return Port(binary.BigEndian.Uint16(d[pkSize*2+2 : pkSize*2+2*2]))
}

func (d RouteDescriptor) SetDstPort(port Port) {
	binary.BigEndian.PutUint16(d[pkSize*2+2:pkSize*2+2*2], uint16(port))
}

func (r Rule) String() string {
	switch t := r.Type(); t {
	case RuleConsume:
		return fmt.Sprintf("App: <remote-pk: %s><remote-port: %d><local-port: %d>",
			r.RouteDescriptor().DstPK(), r.RouteDescriptor().DstPort(), r.RouteDescriptor().SrcPK())
	case RuleForward:
		return fmt.Sprintf("Forward: <next-rid: %d><next-tid: %s><remote-pk: %s><remote-port: %d><local-port: %d>",
			r.NextRouteID(), r.NextTransportID(),
			r.RouteDescriptor().DstPK(), r.RouteDescriptor().DstPort(), r.RouteDescriptor().SrcPK())
	case RuleIntermediaryForward:
		return fmt.Sprintf("IntermediaryForward: <next-rid: %d><next-tid: %s>",
			r.NextRouteID(), r.NextTransportID())
	default:
		panic(fmt.Sprintf("%v: %v", invalidRule, t.String()))
	}
}

type RouteDescriptorFields struct {
	DstPK   cipher.PubKey `json:"dst_pk"`
	SrcPK   cipher.PubKey `json:"src_pk"`
	DstPort Port          `json:"dst_port"`
	SrcPort Port          `json:"src_port"`
}

// RuleConsumeFields summarizes App fields of a RoutingRule.
type RuleConsumeFields struct {
	RouteDescriptor RouteDescriptorFields `json:"route_descriptor"`
}

// RuleForwardFields summarizes Forward fields of a RoutingRule.
type RuleForwardFields struct {
	RouteDescriptor RouteDescriptorFields `json:"route_descriptor"`
	NextRID         RouteID               `json:"next_rid"`
	NextTID         uuid.UUID             `json:"next_tid"`
}

// RuleIntermediaryForwardFields summarizes IntermediaryForward fields of a RoutingRule.
type RuleIntermediaryForwardFields struct {
	NextRID RouteID   `json:"next_rid"`
	NextTID uuid.UUID `json:"next_tid"`
}

// RuleSummary provides a summary of a RoutingRule.
type RuleSummary struct {
	KeepAlive                 time.Duration                  `json:"keep_alive"`
	Type                      RuleType                       `json:"rule_type"`
	KeyRouteID                RouteID                        `json:"key_route_id"`
	ConsumeFields             *RuleConsumeFields             `json:"app_fields,omitempty"`
	ForwardFields             *RuleForwardFields             `json:"forward_fields,omitempty"`
	IntermediaryForwardFields *RuleIntermediaryForwardFields `json:"intermediary_forward_fields,omitempty"`
}

// ToRule converts RoutingRuleSummary to RoutingRule.
func (rs *RuleSummary) ToRule() (Rule, error) {
	switch {

	case rs.Type == RuleConsume && rs.ConsumeFields != nil && rs.ForwardFields == nil && rs.IntermediaryForwardFields == nil:
		f := rs.ConsumeFields
		d := f.RouteDescriptor
		return ConsumeRule(rs.KeepAlive, rs.KeyRouteID, d.DstPK, d.SrcPort, d.DstPort), nil

	case rs.Type == RuleForward && rs.ConsumeFields == nil && rs.ForwardFields != nil && rs.IntermediaryForwardFields == nil:
		f := rs.ForwardFields
		d := f.RouteDescriptor
		return ForwardRule(rs.KeepAlive, rs.KeyRouteID, f.NextRID, f.NextTID, d.DstPK, d.SrcPort, d.DstPort), nil

	case rs.Type == RuleIntermediaryForward && rs.ConsumeFields == nil && rs.ForwardFields == nil && rs.IntermediaryForwardFields != nil:
		f := rs.IntermediaryForwardFields
		return IntermediaryForwardRule(rs.KeepAlive, rs.KeyRouteID, f.NextRID, f.NextTID), nil

	default:
		return nil, errors.New("invalid routing rule summary")
	}
}

// Summary returns the RoutingRule's summary.
func (r Rule) Summary() *RuleSummary {
	summary := RuleSummary{
		KeepAlive:  r.KeepAlive(),
		Type:       r.Type(),
		KeyRouteID: r.KeyRouteID(),
	}
	switch t := summary.Type; t {
	case RuleConsume:
		summary.ConsumeFields = &RuleConsumeFields{
			RouteDescriptor: RouteDescriptorFields{
				DstPK:   r.RouteDescriptor().DstPK(),
				SrcPK:   r.RouteDescriptor().SrcPK(),
				DstPort: r.RouteDescriptor().DstPort(),
				SrcPort: r.RouteDescriptor().SrcPort(),
			},
		}
	case RuleForward:
		summary.ForwardFields = &RuleForwardFields{
			RouteDescriptor: RouteDescriptorFields{
				DstPK:   r.RouteDescriptor().DstPK(),
				SrcPK:   r.RouteDescriptor().SrcPK(),
				DstPort: r.RouteDescriptor().DstPort(),
				SrcPort: r.RouteDescriptor().SrcPort(),
			},
			NextRID: r.NextRouteID(),
			NextTID: r.NextTransportID(),
		}

	case RuleIntermediaryForward:
		summary.IntermediaryForwardFields = &RuleIntermediaryForwardFields{
			NextRID: r.NextRouteID(),
			NextTID: r.NextTransportID(),
		}
	default:
		panic(fmt.Sprintf("%v: %v", invalidRule, t.String()))
	}
	return &summary
}

func ConsumeRule(keepAlive time.Duration, keyRouteID RouteID, remotePK cipher.PubKey, localPort, remotePort Port) Rule {
	rule := Rule(make([]byte, RuleHeaderSize+routeDescriptorSize))

	rule.SetKeepAlive(keepAlive)
	rule.SetType(RuleConsume)
	rule.SetKeyRouteID(keyRouteID)

	rule.SetDstPK(remotePK)
	rule.SetSrcPK(cipher.PubKey{})
	rule.SetDstPort(remotePort)
	rule.SetSrcPort(localPort)

	return rule
}

func ForwardRule(keepAlive time.Duration, keyRouteID, nextRoute RouteID, nextTransport uuid.UUID, remotePK cipher.PubKey, localPort, remotePort Port) Rule {
	rule := Rule(make([]byte, RuleHeaderSize+routeDescriptorSize+4+pkSize))

	rule.SetKeepAlive(keepAlive)
	rule.SetType(RuleForward)
	rule.SetKeyRouteID(keyRouteID)
	rule.SetNextRouteID(nextRoute)
	rule.SetNextTransportID(nextTransport)

	rule.SetDstPK(remotePK)
	rule.SetSrcPK(cipher.PubKey{})
	rule.SetDstPort(remotePort)
	rule.SetSrcPort(localPort)

	return rule
}

func IntermediaryForwardRule(keepAlive time.Duration, keyRouteID, nextRoute RouteID, nextTransport uuid.UUID) Rule {
	rule := Rule(make([]byte, RuleHeaderSize+4+pkSize))

	rule.SetKeepAlive(keepAlive)
	rule.SetType(RuleIntermediaryForward)
	rule.SetKeyRouteID(keyRouteID)
	rule.SetNextRouteID(nextRoute)
	rule.SetNextTransportID(nextTransport)

	return rule
}
