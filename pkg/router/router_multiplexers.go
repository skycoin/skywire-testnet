package router

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/app"
	"github.com/skycoin/skywire/pkg/cipher"
	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/setup"
)

// handles either regular transports or setup transports for one packet.
type tpHandlerFunc func(*router, io.ReadWriter) error

// implements 'tpHandlerFunc'
// obtains and handles a transport packet from 'rw'
func multiplexTransportPacket(r *router, rw io.ReadWriter) error {
	packet := make(routing.Packet, 6)
	if _, err := io.ReadFull(rw, packet); err != nil {
		return err
	}

	payload := make([]byte, packet.Size())
	if _, err := io.ReadFull(rw, payload); err != nil {
		return err
	}

	rule, err := r.rtm.Rule(packet.RouteID())
	if err != nil {
		return err
	}

	r.log.Infof("Got new remote packet with route ID %d. Using rule: %s", packet.RouteID(), rule)

	switch rule.Type() {
	case routing.RuleForward:
		return r.ForwardPacket(rule.TransportID(), rule.RouteID(), payload)

	case routing.RuleApp:
		proc, ok := r.pm.ProcOfPort(rule.LocalPort())
		if !ok {
			return ErrProcNotFound
		}
		lm := app.LoopMeta{
			Local:  app.LoopAddr{PubKey: r.conf.PubKey, Port: rule.LocalPort()},
			Remote: app.LoopAddr{PubKey: rule.RemotePK(), Port: rule.RemotePort()},
		}
		return proc.ConsumePacket(lm, payload)

	default:
		return errors.New("associated rule has invalid type")
	}
}

// implements 'tpHandlerFunc'
// obtains and handles a setup packet from 'rw'
func multiplexSetupPacket(r *router, rw io.ReadWriter) error {
	proto := setup.NewSetupProtocol(rw)

	t, body, err := proto.ReadPacket()
	if err != nil {
		return err
	}

	reject := func(err error) error {
		r.log.Infof("Setup request with type %s failed: %s", t, err)
		return proto.WritePacket(setup.RespFailure, err.Error())
	}

	respond := func(v interface{}, err error) error {
		if err != nil {
			return reject(err)
		}
		return proto.WritePacket(setup.RespSuccess, v)
	}

	r.log.Infof("Got new Setup request with type %s", t)

	switch t {
	case setup.PacketAddRules:
		var rules []routing.Rule
		if err := json.Unmarshal(body, &rules); err != nil {
			return reject(err)
		}
		return respond(handleAddRules(r.log, r.rtm, rules))

	case setup.PacketDeleteRules:
		var rtIDs []routing.RouteID
		if err := json.Unmarshal(body, &rtIDs); err != nil {
			return reject(err)
		}
		return respond(handleDeleteRules(r.log, r.rtm, rtIDs))

	case setup.PacketConfirmLoop:
		var ld setup.LoopData
		if err := json.Unmarshal(body, &ld); err != nil {
			return reject(err)
		}
		return respond(handleConfirmLoop(r.log, r.rtm, r.pm, r.conf.PubKey, ld))

	case setup.PacketLoopClosed:
		var ld setup.LoopData
		if err := json.Unmarshal(body, &ld); err != nil {
			return reject(err)
		}
		return respond(nil, handleLoopClosed(r.pm, r.conf.PubKey, ld))

	default:
		return reject(errors.New("unknown foundation packet"))
	}
}

// triggered when a 'AddRules' packet is received from SetupNode
func handleAddRules(log *logging.Logger, rtm *RoutingTableManager, rules []routing.Rule) ([]routing.RouteID, error) {
	res := make([]routing.RouteID, len(rules))
	for idx, rule := range rules {
		routeID, err := rtm.AddRule(rule)
		if err != nil {
			return nil, fmt.Errorf("routing table: %s", err)
		}
		res[idx] = routeID
		log.Infof("Added new Routing Rule with ID %d %s", routeID, rule)
	}
	return res, nil
}

// triggered when a 'DeleteRules' packet is received from SetupNode
func handleDeleteRules(log *logging.Logger, rtm *RoutingTableManager, ids []routing.RouteID) ([]routing.RouteID, error) {
	err := rtm.DeleteRules(ids...)
	if err != nil {
		return nil, fmt.Errorf("routing table: %s", err)
	}

	log.Infof("Removed Routing Rules with IDs %s", ids)
	return ids, nil
}

// triggered when a 'ConfirmLoop' packet is received from SetupNode
func handleConfirmLoop(log *logging.Logger, rtm *RoutingTableManager, pm ProcManager, localPK cipher.PubKey, ld setup.LoopData) ([]byte, error) {
	lm := makeLoopMeta(localPK, ld)

	appRtID, appRule, ok := rtm.FindAppRule(lm)
	if !ok {
		return nil, errors.New("unknown loop")
	}
	fwdRule, err := rtm.FindFwdRule(ld.RouteID)
	if err != nil {
		return nil, err
	}

	proc, ok := pm.ProcOfPort(lm.Local.Port)
	if !ok {
		return nil, ErrProcNotFound
	}
	msg, err := proc.ConfirmLoop(lm, fwdRule.TransportID(), fwdRule.RouteID(), ld.NoiseMessage)
	if err != nil {
		return nil, fmt.Errorf("confirm: %s", err)
	}

	log.Infof("Setting reverse route ID %d for rule with ID %d", ld.RouteID, appRtID)
	appRule.SetRouteID(ld.RouteID)

	if err := rtm.SetRule(appRtID, appRule); err != nil {
		return nil, fmt.Errorf("routing table: %s", err)
	}

	log.Infof("Confirmed loop with %s", lm.Remote)
	return msg, nil
}

// triggered when a 'LoopClosed' packet is received from SetupNode
func handleLoopClosed(pm ProcManager, localPK cipher.PubKey, ld setup.LoopData) error {
	lm := makeLoopMeta(localPK, ld)

	proc, ok := pm.ProcOfPort(lm.Local.Port)
	if !ok {
		return ErrProcNotFound
	}
	return proc.ConfirmCloseLoop(lm)
}
