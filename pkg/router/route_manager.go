package router

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/app"
	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/setup"
)

type setupCallbacks struct {
	ConfirmLoop func(addr *app.LoopAddr, rule routing.Rule, noiseMsg []byte) (noiseRes []byte, err error)
	LoopClosed  func(addr *app.LoopAddr) error
}

type routeManager struct {
	Logger *logging.Logger

	rt        *managedRoutingTable
	callbacks *setupCallbacks
}

func (rm *routeManager) GetRule(routeID routing.RouteID) (routing.Rule, error) {
	rule, err := rm.rt.Rule(routeID)
	if err != nil {
		return nil, fmt.Errorf("routing table: %s", err)
	}

	if rule == nil {
		return nil, errors.New("unknown RouteID")
	}

	if rule.Expiry().Before(time.Now()) {
		return nil, errors.New("expired routing rule")
	}

	return rule, nil
}

func (rm *routeManager) RemoveLoopRule(addr *app.LoopAddr) error {
	var appRouteID routing.RouteID
	var appRule routing.Rule
	err := rm.rt.RangeRules(func(routeID routing.RouteID, rule routing.Rule) bool {
		if rule.Type() != routing.RuleApp || rule.RemotePK() != addr.Remote.PubKey ||
			rule.RemotePort() != addr.Remote.Port || rule.LocalPort() != addr.Port {
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

func (rm *routeManager) Serve(rw io.ReadWriter) error {
	proto := setup.NewProtocol(rw)

	t, body, err := proto.ReadPacket()
	if err != nil {
		fmt.Println("err:", err)
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
		respBody, err = rm.confirmLoop(body)
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

func (rm *routeManager) addRoutingRules(data []byte) ([]routing.RouteID, error) {
	rules := []routing.Rule{}
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
	ruleIDs := []routing.RouteID{}
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

func (rm *routeManager) confirmLoop(data []byte) (noiseRes []byte, err error) {
	ld := setup.LoopData{}
	if err = json.Unmarshal(data, &ld); err != nil {
		return
	}

	raddr := &app.Addr{PubKey: ld.RemotePK, Port: ld.RemotePort}

	var appRouteID routing.RouteID
	var appRule routing.Rule
	err = rm.rt.RangeRules(func(routeID routing.RouteID, rule routing.Rule) bool {
		if rule.Type() != routing.RuleApp || rule.RemotePK() != ld.RemotePK ||
			rule.RemotePort() != ld.RemotePort || rule.LocalPort() != ld.LocalPort {
			return true
		}

		appRouteID = routeID
		appRule = make(routing.Rule, len(rule))
		copy(appRule, rule)
		return false
	})
	if err != nil {
		err = fmt.Errorf("routing table: %s", err)
		return
	}

	if appRule == nil {
		err = errors.New("unknown loop")
		return
	}

	rule, err := rm.rt.Rule(ld.RouteID)
	if err != nil {
		err = fmt.Errorf("routing table: %s", err)
		return
	}

	if rule.Type() != routing.RuleForward {
		err = errors.New("reverse rule is not forward")
		return
	}

	msg, err := rm.callbacks.ConfirmLoop(&app.LoopAddr{Port: ld.LocalPort, Remote: *raddr}, rule, ld.NoiseMessage)
	if err != nil {
		err = fmt.Errorf("confirm: %s", err)
		return
	}

	rm.Logger.Infof("Setting reverse route ID %d for rule with ID %d", ld.RouteID, appRouteID)
	appRule.SetRouteID(ld.RouteID)
	if rErr := rm.rt.SetRule(appRouteID, appRule); rErr != nil {
		err = fmt.Errorf("routing table: %s", rErr)
		return
	}

	rm.Logger.Infof("Confirmed loop with %s:%d", ld.RemotePK, ld.RemotePort)
	return msg, nil
}

func (rm *routeManager) loopClosed(data []byte) error {
	ld := &setup.LoopData{}
	if err := json.Unmarshal(data, ld); err != nil {
		return err
	}

	raddr := &app.Addr{PubKey: ld.RemotePK, Port: ld.RemotePort}
	addr := &app.LoopAddr{Port: ld.LocalPort, Remote: *raddr}
	return rm.callbacks.LoopClosed(addr)
}
