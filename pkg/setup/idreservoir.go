package setup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/skycoin/dmsg/cipher"

	"github.com/skycoin/skywire/pkg/routing"
)

type idReservoir struct {
	rec map[cipher.PubKey]uint8
	ids map[cipher.PubKey][]routing.RouteID
	mx  sync.Mutex
}

func newIDReservoir(routes ...routing.Path) (*idReservoir, int) {
	rec := make(map[cipher.PubKey]uint8)
	var total int

	for _, rt := range routes {
		if len(rt) == 0 {
			continue
		}
		rec[rt[0].From]++
		for _, hop := range rt {
			rec[hop.To]++
		}
		total += len(rt) + 1
	}

	return &idReservoir{
		rec: rec,
		ids: make(map[cipher.PubKey][]routing.RouteID),
	}, total
}

func (idr *idReservoir) ReserveIDs(ctx context.Context, reserve func(ctx context.Context, pk cipher.PubKey, n uint8) ([]routing.RouteID, error)) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, len(idr.rec))
	defer close(errCh)

	for pk, n := range idr.rec {
		pk, n := pk, n
		go func() {
			ids, err := reserve(ctx, pk, n)
			if err != nil {
				errCh <- fmt.Errorf("reserve routeID from %s failed: %v", pk, err)
				return
			}
			idr.mx.Lock()
			idr.ids[pk] = ids
			idr.mx.Unlock()
			errCh <- nil
		}()
	}

	return finalError(len(idr.rec), errCh)
}

func (idr *idReservoir) PopID(pk cipher.PubKey) (routing.RouteID, bool) {
	idr.mx.Lock()
	defer idr.mx.Unlock()

	ids, ok := idr.ids[pk]
	if !ok || len(ids) == 0 {
		return 0, false
	}

	idr.ids[pk] = ids[1:]
	return ids[0], true
}

// RulesMap associates a slice of rules to a visor's public key.
type RulesMap map[cipher.PubKey][]routing.Rule

func (rm RulesMap) String() string {
	out := make(map[cipher.PubKey][]string, len(rm))
	for pk, rules := range rm {
		str := make([]string, len(rules))
		for i, rule := range rules {
			str[i] = rule.String()
		}
		out[pk] = str
	}
	jb, err := json.MarshalIndent(out, "", "\t")
	if err != nil {
		panic(err)
	}
	return string(jb)
}

// GenerateRules generates rules for a given LoopDescriptor.
// The outputs are as follows:
// - rules: a map that relates a slice of routing rules to a given visor's public key.
// - srcAppRID: the initiating node's route ID that references the FWD rule.
// - dstAppRID: the responding node's route ID that references the FWD rule.
// - err: an error (if any).
func GenerateRules(idc *idReservoir, ld routing.LoopDescriptor) (rules RulesMap, srcFwdRID, dstFwdRID routing.RouteID, err error) {
	rules = make(RulesMap)
	src, dst := ld.Loop.Local, ld.Loop.Remote

	firstFwdRID, lastFwdRID, err := SaveForwardRules(rules, idc, ld.KeepAlive, ld.Forward)
	if err != nil {
		return nil, 0, 0, err
	}
	firstRevRID, lastRevRID, err := SaveForwardRules(rules, idc, ld.KeepAlive, ld.Reverse)
	if err != nil {
		return nil, 0, 0, err
	}

	rules[src.PubKey] = append(rules[src.PubKey],
		routing.AppRule(ld.KeepAlive, firstRevRID, lastFwdRID, dst.PubKey, src.Port, dst.Port))
	rules[dst.PubKey] = append(rules[dst.PubKey],
		routing.AppRule(ld.KeepAlive, firstFwdRID, lastRevRID, src.PubKey, dst.Port, src.Port))

	return rules, firstFwdRID, firstRevRID, nil
}

// SaveForwardRules creates the rules of the given route, and saves them in the 'rules' input.
// Note that the last rule for the route is always an APP rule, and so is not created here.
// The outputs are as follows:
// - firstRID: the first visor's route ID.
// - lastRID: the last visor's route ID (note that there is no rule set for this ID yet).
// - err: an error (if any).
func SaveForwardRules(rules RulesMap, idc *idReservoir, keepAlive time.Duration, route routing.Path) (firstRID, lastRID routing.RouteID, err error) {

	// 'firstRID' is the first visor's key routeID - this is to be returned.
	var ok bool
	if firstRID, ok = idc.PopID(route[0].From); !ok {
		return 0, 0, errors.New("fucked up")
	}

	var rID = firstRID
	for _, hop := range route {
		nxtRID, ok := idc.PopID(hop.To)
		if !ok {
			return 0, 0, errors.New("fucked up")
		}
		rule := routing.ForwardRule(keepAlive, nxtRID, hop.Transport, rID)
		rules[hop.From] = append(rules[hop.From], rule)

		rID = nxtRID
	}

	return firstRID, rID, nil
}

func finalError(n int, errCh <-chan error) error {
	var finalErr error
	for i := 0; i < n; i++ {
		if err := <-errCh; err != nil {
			finalErr = err
		}
	}
	return finalErr
}
