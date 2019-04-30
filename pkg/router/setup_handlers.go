package router

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/setup"
)

type setupHandlers struct {
	r          *router
	am         ProcManager
	sproto     *setup.Protocol
	packetType setup.PacketType
	packetBody []byte
}

func makeSetupHandlers(r *router, pm ProcManager, rw io.ReadWriter) (setupHandlers, error) {
	sproto := setup.NewSetupProtocol(rw)

	packetType, packetBody, err := sproto.ReadPacket()
	if err != nil {
		return setupHandlers{}, err
	}

	r.log.Infof("Got new Setup request with type %s", packetType)

	return setupHandlers{r, pm, sproto, packetType, packetBody}, nil
}

// triggered when a 'AddRules' packet is received from SetupNode
func (sh setupHandlers) addRules(rules []routing.Rule) ([]routing.RouteID, error) {
	res := make([]routing.RouteID, len(rules))
	for idx, rule := range rules {
		routeID, err := sh.r.rtm.AddRule(rule)
		if err != nil {
			return nil, fmt.Errorf("routing table: %s", err)
		}
		res[idx] = routeID
		sh.r.log.Infof("Added new Routing Rule with ID %d %s", routeID, rule)
	}
	return res, nil
}

// triggered when a 'DeleteRules' packet is received from SetupNode
func (sh setupHandlers) deleteRules(rtIDs []routing.RouteID) ([]routing.RouteID, error) {
	err := sh.r.rtm.DeleteRules(rtIDs...)
	if err != nil {
		return nil, fmt.Errorf("routing table: %s", err)
	}

	sh.r.log.Infof("Removed Routing Rules with IDs %s", rtIDs)
	return rtIDs, nil
}

// triggered when a 'ConfirmLoop' packet is received from SetupNode
func (sh setupHandlers) confirmLoop(ld setup.LoopData) ([]byte, error) {
	lm := makeLoopMeta(sh.r.conf.PubKey, ld)

	appRtID, appRule, ok := sh.r.rtm.FindAppRule(lm)
	if !ok {
		return nil, errors.New("unknown loop")
	}
	fwdRule, err := sh.r.rtm.FindFwdRule(ld.RouteID)
	if err != nil {
		return nil, err
	}

	proc, ok := sh.am.ProcOfPort(lm.Local.Port)
	if !ok {
		return nil, ErrProcNotFound
	}
	msg, err := proc.ConfirmLoop(lm, fwdRule.TransportID(), fwdRule.RouteID(), ld.NoiseMessage)
	if err != nil {
		return nil, fmt.Errorf("confirm: %s", err)
	}

	sh.r.log.Infof("Setting reverse route ID %d for rule with ID %d", ld.RouteID, appRtID)
	appRule.SetRouteID(ld.RouteID)

	if err := sh.r.rtm.SetRule(appRtID, appRule); err != nil {
		return nil, fmt.Errorf("routing table: %s", err)
	}

	sh.r.log.Infof("Confirmed loop with %s", lm.Remote)
	return msg, nil
}

// triggered when a 'LoopClosed' packet is received from SetupNode
func (sh setupHandlers) loopClosed(ld setup.LoopData) error {
	lm := makeLoopMeta(sh.r.conf.PubKey, ld)

	proc, ok := sh.am.ProcOfPort(lm.Local.Port)
	if !ok {
		return ErrProcNotFound
	}
	return proc.ConfirmCloseLoop(lm)
}

func (sh setupHandlers) reject(err error) error {
	sh.r.log.Infof("Setup request with type %s failed: %s", sh.packetType, err)
	return sh.sproto.WritePacket(setup.RespFailure, err.Error())
}

func (sh setupHandlers) respondWith(v interface{}, err error) error {
	if err != nil {
		return sh.reject(err)
	}
	return sh.sproto.WritePacket(setup.RespSuccess, v)
}

func (sh setupHandlers) handle() error {
	switch sh.packetType {
	case setup.PacketAddRules:
		var rules []routing.Rule
		if err := json.Unmarshal(sh.packetBody, &rules); err != nil {
			return sh.reject(err)
		}
		return sh.respondWith(sh.addRules(rules))

	case setup.PacketDeleteRules:
		var rtIDs []routing.RouteID
		if err := json.Unmarshal(sh.packetBody, &rtIDs); err != nil {
			return sh.reject(err)
		}
		return sh.respondWith(sh.deleteRules(rtIDs))

	case setup.PacketConfirmLoop:
		var ld setup.LoopData
		if err := json.Unmarshal(sh.packetBody, &ld); err != nil {
			return sh.reject(err)
		}
		return sh.respondWith(sh.confirmLoop(ld))

	case setup.PacketLoopClosed:
		var ld setup.LoopData
		if err := json.Unmarshal(sh.packetBody, &ld); err != nil {
			return sh.reject(err)
		}
		return sh.respondWith(nil, sh.loopClosed(ld))

	default:
		return sh.reject(errors.New("unknown foundation packet"))
	}
}
