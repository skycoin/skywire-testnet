package router

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"net"
	"time"

	"github.com/skycoin/skywire/pkg/snet"

	"github.com/skycoin/dmsg/cipher"

	"github.com/skycoin/skywire/pkg/setup"

	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/routing"
)

type RMConfig struct {
	SetupPKs               []cipher.PubKey // Trusted setup PKs.
	GarbageCollectDuration time.Duration
	OnConfirmLoop          func(loop routing.Loop, rule routing.Rule) (err error)
	OnLoopClosed           func(loop routing.Loop) error
}

func (sc RMConfig) SetupIsTrusted(sPK cipher.PubKey) bool {
	for _, pk := range sc.SetupPKs {
		if sPK == pk {
			return true
		}
	}
	return false
}

type routeManager struct {
	Logger *logging.Logger
	conf   RMConfig
	n      *snet.Network
	sl     *snet.Listener // Listens for setup node requests.
	rt     *managedRoutingTable
	done   chan struct{}
}

// NewRouteManager creates a new route manager.
func NewRouteManager(n *snet.Network, rt routing.Table, config RMConfig) (*routeManager, error) {
	sl, err := n.Listen(snet.DmsgType, snet.AwaitSetupPort)
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
	defer func() { _ = conn.Close() }() //nolint:errcheck

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
		respBody, err = rm.occupyRouteID()
	default:
		err = errors.New("unknown foundation packet")
	}

	if err != nil {
		rm.Logger.Infof("Setup request with type %s failed: %s", t, err)
		return proto.WritePacket(setup.RespFailure, err.Error())
	}
	return proto.WritePacket(setup.RespSuccess, respBody)
}

func (rm *routeManager) rtGarbageCollectLoop() {
	if rm.conf.GarbageCollectDuration <= 0 {
		return
	}
	ticker := time.NewTicker(rm.conf.GarbageCollectDuration)
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

func (rm *routeManager) dialSetupConn(ctx context.Context) (*snet.Conn, error) {
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

func (rm *routeManager) setRoutingRules(data []byte) error {
	var rules []routing.Rule
	if err := json.Unmarshal(data, &rules); err != nil {
		return err
	}

	for _, rule := range rules {
		routeID := rule.RequestRouteID()
		if err := rm.rt.SetRule(routeID, rule); err != nil {
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

func (rm *routeManager) occupyRouteID() ([]routing.RouteID, error) {
	routeID, err := rm.rt.AddRule(routing.ForwardRule(DefaultRouteKeepAlive, 0, uuid.UUID{}, 0))
	if err != nil {
		return nil, err
	}

	return []routing.RouteID{routeID}, nil
}
