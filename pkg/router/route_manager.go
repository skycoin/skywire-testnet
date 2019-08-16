package router

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/skycoin/dmsg/cipher"

	"github.com/skycoin/skywire/pkg/network"
	"github.com/skycoin/skywire/pkg/setup"

	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/routing"
)

const (
	rtGarbageCollectDuration = time.Minute * 5
)

type setupConfig struct {
	SetupPKs      []cipher.PubKey // Trusted setup PKs.
	OnConfirmLoop func(loop routing.Loop, rule routing.Rule) (err error)
	OnLoopClosed  func(loop routing.Loop) error
}

func (sc setupConfig) SetupIsTrusted(sPK cipher.PubKey) bool {
	for _, pk := range sc.SetupPKs {
		if sPK == pk {
			return true
		}
	}
	return false
}

type routeManager struct {
	Logger *logging.Logger
	conf   setupConfig
	n      *network.Network
	sl     *network.Listener // Listens for setup node requests.
	rt     *managedRoutingTable
	done   chan struct{}
}

func newRouteManager(n *network.Network, rt routing.Table, config setupConfig) (*routeManager, error) {
	sl, err := n.Listen(network.DmsgNet, network.AwaitSetupPort)
	if err != nil {
		return nil, err
	}
	return &routeManager{
		Logger: logging.MustGetLogger("route_manager"),
		conf:   config,
		n:      n,
		sl:     sl,
		rt:     manageRoutingTable(rt),
		done:   make(chan struct{}),
	}, nil
}

func (rm *routeManager) Close() error {
	close(rm.done)
	return rm.sl.Close()
}

func (rm *routeManager) Serve() {

	// Routing table garbage collect loop.
	go rm.rtGarbageCollectLoop()

	// Accept setup node request loop.
	for {
		conn, err := rm.sl.AcceptConn()
		if err != nil {
			rm.Logger.WithError(err).Warnf("stopped serving")
			return
		}
		if !rm.conf.SetupIsTrusted(conn.RemotePK()) {
			rm.Logger.Warnf("closing conn from untrusted setup node: %v", conn.Close())
			continue
		}
		go func(conn *network.Conn) {
			rm.Logger.Infof("handling setup request: setupPK(%s)", conn.RemotePK())
			defer func() { _ = conn.Close() }() //nolint:errcheck

			if err := rm.handleSetupConn(conn); err != nil {
				rm.Logger.WithError(err).Warnf("setup request failed: setupPK(%s)", conn.RemotePK())
			}
			rm.Logger.Infof("successfully handled setup request: setupPK(%s)", conn.RemotePK())
		}(conn)
	}
}

func (rm *routeManager) rtGarbageCollectLoop() {
	ticker := time.NewTicker(rtGarbageCollectDuration)
	defer ticker.Stop()
	for {
		select {
		case <-rm.done:
			return
		case <-ticker.C:
			if err := rm.rt.Cleanup(); err != nil {
				rm.Logger.WithError(err).Warnf("routing table cleanup returned error")
			}
		}
	}
}

func (rm *routeManager) handleSetupConn(conn *network.Conn) error {
	proto := setup.NewSetupProtocol(conn)
	t, body, err := proto.ReadPacket()

	if err != nil {
		return err
	}
	rm.Logger.Infof("Got new Setup request with type %s", t)

	var respBody interface{}
	switch t {
	case setup.PacketAddRules:
		respBody, err = rm.addRoutingRules(body)
	case setup.PacketDeleteRules:
		respBody, err = rm.deleteRoutingRules(body)
	case setup.PacketConfirmLoop:
		err = rm.confirmLoop(body)
	case setup.PacketLoopClosed:
		err = rm.loopClosed(body)
	default:
		err = errors.New("unknown foundation packet")
	}

	if err != nil {
		rm.Logger.Infof("Setup request with type %s failed: %s", t, err)
		return proto.WritePacket(setup.RespFailure, err.Error())
	}
	return proto.WritePacket(setup.RespSuccess, respBody)
}

func (rm *routeManager) dialSetupConn(ctx context.Context) (*network.Conn, error) {
	for _, sPK := range rm.conf.SetupPKs {
		conn, err := rm.n.Dial(network.DmsgNet, sPK, network.SetupPort)
		if err != nil {
			rm.Logger.WithError(err).Warnf("failed to dial to setup node: setupPK(%s)", sPK)
			continue
		}
		return conn, nil
	}
	return nil, errors.New("failed to dial to a setup node")
}

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

	if rule.Expiry().Before(time.Now()) {
		return nil, errors.New("expired routing rule")
	}

	return rule, nil
}

func (rm *routeManager) RemoveLoopRule(loop routing.Loop) error {
	var appRouteID routing.RouteID
	var appRule routing.Rule
	err := rm.rt.RangeRules(func(routeID routing.RouteID, rule routing.Rule) bool {
		if rule.Type() != routing.RuleApp || rule.RemotePK() != loop.Remote.PubKey ||
			rule.RemotePort() != loop.Remote.Port || rule.LocalPort() != loop.Local.Port {
			return true
		}

		appRouteID = routeID
		appRule = make(routing.Rule, len(rule))
		copy(appRule, rule)

		return false
	})
	if err != nil {
		return fmt.Errorf("routing table: %s", err)
	}

	if len(appRule) == 0 {
		return nil
	}

	if err = rm.rt.DeleteRules(appRouteID); err != nil {
		return fmt.Errorf("routing table: %s", err)
	}

	return nil
}

func (rm *routeManager) addRoutingRules(data []byte) ([]routing.RouteID, error) {
	var rules []routing.Rule
	if err := json.Unmarshal(data, &rules); err != nil {
		return nil, err
	}

	res := make([]routing.RouteID, len(rules))
	for idx, rule := range rules {
		routeID, err := rm.rt.AddRule(rule)
		if err != nil {
			return nil, fmt.Errorf("routing table: %s", err)
		}

		res[idx] = routeID
		rm.Logger.Infof("Added new Routing Rule with ID %d %s", routeID, rule)
	}

	return res, nil
}

func (rm *routeManager) deleteRoutingRules(data []byte) ([]routing.RouteID, error) {
	var ruleIDs []routing.RouteID
	if err := json.Unmarshal(data, &ruleIDs); err != nil {
		return nil, err
	}

	err := rm.rt.DeleteRules(ruleIDs...)
	if err != nil {
		return nil, fmt.Errorf("routing table: %s", err)
	}

	rm.Logger.Infof("Removed Routing Rules with IDs %s", ruleIDs)
	return ruleIDs, nil
}

func (rm *routeManager) confirmLoop(data []byte) error {
	var ld routing.LoopData
	if err := json.Unmarshal(data, &ld); err != nil {
		return err
	}

	var appRouteID routing.RouteID
	var appRule routing.Rule
	err := rm.rt.RangeRules(func(routeID routing.RouteID, rule routing.Rule) bool {
		if rule.Type() != routing.RuleApp || rule.RemotePK() != ld.Loop.Remote.PubKey ||
			rule.RemotePort() != ld.Loop.Remote.Port || rule.LocalPort() != ld.Loop.Local.Port {
			return true
		}

		appRouteID = routeID
		appRule = make(routing.Rule, len(rule))
		copy(appRule, rule)
		return false
	})
	if err != nil {
		return fmt.Errorf("routing table: %s", err)
	}

	if appRule == nil {
		return errors.New("unknown loop")
	}

	rule, err := rm.rt.Rule(ld.RouteID)
	if err != nil {
		return fmt.Errorf("routing table: %s", err)
	}

	if rule.Type() != routing.RuleForward {
		return errors.New("reverse rule is not forward")
	}

	if err = rm.conf.OnConfirmLoop(ld.Loop, rule); err != nil {
		return fmt.Errorf("confirm: %s", err)
	}

	rm.Logger.Infof("Setting reverse route ID %d for rule with ID %d", ld.RouteID, appRouteID)
	appRule.SetRouteID(ld.RouteID)
	if rErr := rm.rt.SetRule(appRouteID, appRule); rErr != nil {
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
