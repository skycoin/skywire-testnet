package router

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"

	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/setup"
	"github.com/skycoin/skywire/pkg/snet"
)

// RMConfig represents route manager configuration.
type RMConfig struct {
	SetupPKs      []cipher.PubKey // Trusted setup PKs.
	OnConfirmLoop func(loop routing.Loop, rule routing.Rule) (err error)
	OnLoopClosed  func(loop routing.Loop) error
}

// SetupIsTrusted checks if setup node is trusted.
func (sc RMConfig) SetupIsTrusted(sPK cipher.PubKey) bool {
	for _, pk := range sc.SetupPKs {
		if sPK == pk {
			return true
		}
	}
	return false
}

// routeManager represents route manager.
type routeManager struct {
	Logger *logging.Logger
	conf   RMConfig
	n      *snet.Network
	sl     *snet.Listener // Listens for setup node requests.
	rt     routing.Table
	done   chan struct{}
}

// newRouteManager creates a new route manager.
func newRouteManager(n *snet.Network, rt routing.Table, config RMConfig) (*routeManager, error) {
	sl, err := n.Listen(snet.DmsgType, snet.AwaitSetupPort)
	if err != nil {
		return nil, err
	}
	return &routeManager{
		Logger: logging.MustGetLogger("route_manager"),
		conf:   config,
		n:      n,
		sl:     sl,
		rt:     rt,
		done:   make(chan struct{}),
	}, nil
}

// Close closes route manager.
func (rm *routeManager) Close() error {
	close(rm.done)
	return rm.sl.Close()
}

// Serve initiates serving connections by route manager.
func (rm *routeManager) Serve() {
	// Accept setup node request loop.
	for {
		if err := rm.serveConn(); err != nil {
			rm.Logger.WithError(err).Warnf("stopped serving")
			return
		}
	}
}

func (rm *routeManager) serveConn() error {
	conn, err := rm.sl.AcceptConn()
	if err != nil {
		rm.Logger.WithError(err).Warnf("stopped serving")
		return err
	}
	if !rm.conf.SetupIsTrusted(conn.RemotePK()) {
		rm.Logger.Warnf("closing conn from untrusted setup node: %v", conn.Close())
		return nil
	}
	go func() {
		rm.Logger.Infof("handling setup request: setupPK(%s)", conn.RemotePK())
		if err := rm.handleSetupConn(conn); err != nil {
			rm.Logger.WithError(err).Warnf("setup request failed: setupPK(%s)", conn.RemotePK())
		}
		rm.Logger.Infof("successfully handled setup request: setupPK(%s)", conn.RemotePK())
	}()
	return nil
}

func (rm *routeManager) handleSetupConn(conn net.Conn) error {
	defer func() {
		if err := conn.Close(); err != nil {
			log.WithError(err).Warn("Failed to close connection")
		}
	}()

	proto := setup.NewSetupProtocol(conn)
	t, body, err := proto.ReadPacket()

	if err != nil {
		return err
	}
	rm.Logger.Infof("Got new Setup request with type %s", t)

	var respBody interface{}
	switch t {
	case setup.PacketAddRules:
		err = rm.setRoutingRules(body)
	case setup.PacketDeleteRules:
		respBody, err = rm.deleteRoutingRules(body)
	case setup.PacketConfirmLoop:
		err = rm.confirmLoop(body)
	case setup.PacketLoopClosed:
		err = rm.loopClosed(body)
	case setup.PacketRequestRouteID:
		respBody, err = rm.occupyRouteID(body)
	default:
		err = errors.New("unknown foundation packet")
	}

	if err != nil {
		rm.Logger.Infof("Setup request with type %s failed: %s", t, err)
		return proto.WritePacket(setup.RespFailure, err.Error())
	}
	return proto.WritePacket(setup.RespSuccess, respBody)
}

func (rm *routeManager) dialSetupConn(_ context.Context) (*snet.Conn, error) {
	for _, sPK := range rm.conf.SetupPKs {
		conn, err := rm.n.Dial(snet.DmsgType, sPK, snet.SetupPort)
		if err != nil {
			rm.Logger.WithError(err).Warnf("failed to dial to setup node: setupPK(%s)", sPK)
			continue
		}
		return conn, nil
	}
	return nil, errors.New("failed to dial to a setup node")
}

// GetRule gets routing rule.
func (rm *routeManager) GetRule(routeID routing.RouteID) (routing.Rule, error) {
	rule, err := rm.rt.Rule(routeID)
	if err != nil {
		return nil, fmt.Errorf("routing table: %s", err)
	}

	if rule == nil {
		return nil, errors.New("unknown RouteID")
	}

	// TODO(evanlinjin): This is a workaround for ensuring the read-in rule is of the correct size.
	// Sometimes it is not, causing a segfault later down the line.
	if len(rule) < routing.RuleHeaderSize {
		return nil, errors.New("corrupted rule")
	}

	return rule, nil
}

// RemoveLoopRule removes loop rule.
func (rm *routeManager) RemoveLoopRule(loop routing.Loop) {
	var appRouteID routing.RouteID
	var consumeRule routing.Rule
	rm.rt.RangeRules(func(routeID routing.RouteID, rule routing.Rule) bool {
		if rule.Type() != routing.RuleConsume || rule.RouteDescriptor().DstPK() != loop.Remote.PubKey ||
			rule.RouteDescriptor().DstPort() != loop.Remote.Port ||
			rule.RouteDescriptor().SrcPort() != loop.Local.Port {
			return true
		}

		appRouteID = routeID
		consumeRule = make(routing.Rule, len(rule))
		copy(consumeRule, rule)

		return false
	})

	if len(consumeRule) != 0 {
		rm.rt.DelRules([]routing.RouteID{appRouteID})
	}
}

func (rm *routeManager) setRoutingRules(data []byte) error {
	var rules []routing.Rule
	if err := json.Unmarshal(data, &rules); err != nil {
		return err
	}

	for _, rule := range rules {
		routeID := rule.KeyRouteID()
		if err := rm.rt.SaveRule(routeID, rule); err != nil {
			return fmt.Errorf("routing table: %s", err)
		}

		rm.Logger.Infof("Set new Routing Rule with ID %d %s", routeID, rule)
	}

	return nil
}

func (rm *routeManager) deleteRoutingRules(data []byte) ([]routing.RouteID, error) {
	var ruleIDs []routing.RouteID
	if err := json.Unmarshal(data, &ruleIDs); err != nil {
		return nil, err
	}

	rm.rt.DelRules(ruleIDs)
	rm.Logger.Infof("Removed Routing Rules with IDs %s", ruleIDs)

	return ruleIDs, nil
}

func (rm *routeManager) confirmLoop(data []byte) error {
	var ld routing.LoopData
	if err := json.Unmarshal(data, &ld); err != nil {
		return err
	}

	var appRouteID routing.RouteID
	var consumeRule routing.Rule
	rm.rt.RangeRules(func(routeID routing.RouteID, rule routing.Rule) bool {
		if rule.Type() != routing.RuleConsume || rule.RouteDescriptor().DstPK() != ld.Loop.Remote.PubKey ||
			rule.RouteDescriptor().DstPort() != ld.Loop.Remote.Port ||
			rule.RouteDescriptor().SrcPort() != ld.Loop.Local.Port {
			return true
		}

		appRouteID = routeID
		consumeRule = make(routing.Rule, len(rule))
		copy(consumeRule, rule)
		return false
	})

	if consumeRule == nil {
		return errors.New("unknown loop")
	}

	rule, err := rm.rt.Rule(ld.RouteID)
	if err != nil {
		return fmt.Errorf("routing table: %s", err)
	}

	if rule.Type() != routing.RuleIntermediaryForward {
		return errors.New("reverse rule is not forward")
	}

	if err = rm.conf.OnConfirmLoop(ld.Loop, rule); err != nil {
		return fmt.Errorf("confirm: %s", err)
	}

	rm.Logger.Infof("Setting reverse route ID %d for rule with ID %d", ld.RouteID, appRouteID)
	consumeRule.SetKeyRouteID(ld.RouteID)
	if rErr := rm.rt.SaveRule(appRouteID, consumeRule); rErr != nil {
		return fmt.Errorf("routing table: %s", rErr)
	}

	rm.Logger.Infof("Confirmed loop with %s:%d", ld.Loop.Remote.PubKey, ld.Loop.Remote.Port)
	return nil
}

func (rm *routeManager) loopClosed(data []byte) error {
	var ld routing.LoopData
	if err := json.Unmarshal(data, &ld); err != nil {
		return err
	}

	return rm.conf.OnLoopClosed(ld.Loop)
}

func (rm *routeManager) occupyRouteID(data []byte) ([]routing.RouteID, error) {
	var n uint8
	if err := json.Unmarshal(data, &n); err != nil {
		return nil, err
	}

	var ids = make([]routing.RouteID, n)
	for i := range ids {
		routeID, err := rm.rt.ReserveKey()
		if err != nil {
			return nil, err
		}
		ids[i] = routeID
	}
	return ids, nil
}
