package setup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/skycoin/dmsg"
	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/routing"
)

var ErrNoKey = errors.New("id reservoir has no key")

type idReservoir struct {
	rec map[cipher.PubKey]uint8
	ids map[cipher.PubKey][]routing.RouteID
	mx  sync.Mutex
}

func newIDReservoir(paths ...routing.Path) (*idReservoir, int) {
	rec := make(map[cipher.PubKey]uint8)
	var total int

	for _, path := range paths {
		if len(path) == 0 {
			continue
		}
		rec[path[0].From]++
		for _, hop := range path {
			rec[hop.To]++
		}
		total += len(path) + 1
	}

	return &idReservoir{
		rec: rec,
		ids: make(map[cipher.PubKey][]routing.RouteID),
	}, total
}

type reserveFunc func(ctx context.Context, log *logging.Logger, dmsgC *dmsg.Client, pk cipher.PubKey, n uint8) ([]routing.RouteID, error)

func (idr *idReservoir) ReserveIDs(ctx context.Context, log *logging.Logger, dmsgC *dmsg.Client, reserve reserveFunc) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, len(idr.rec))
	defer close(errCh)

	for pk, n := range idr.rec {
		pk, n := pk, n
		go func() {
			ids, err := reserve(ctx, log, dmsgC, pk, n)
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

// TODO: fix comment, refactor
// GenerateRules2 generates rules for a given route.
// The outputs are as follows:
// - a map that relates a slice of routing rules to a given visor's public key.
// - an error (if any).
func (idr *idReservoir) GenerateRules(forward, reverse routing.Route) (forwardRules, consumeRules map[cipher.PubKey]routing.Rule, intermediaryRules RulesMap, err error) {
	forwardRules = make(map[cipher.PubKey]routing.Rule)
	consumeRules = make(map[cipher.PubKey]routing.Rule)
	intermediaryRules = make(RulesMap)

	for _, route := range []routing.Route{forward, reverse} {
		// 'firstRID' is the first visor's key routeID
		firstRID, ok := idr.PopID(route.Path[0].From)
		if !ok {
			return nil, nil, nil, ErrNoKey
		}

		desc := route.Desc
		dstPK := desc.DstPK()
		srcPort := desc.SrcPort()
		dstPort := desc.DstPort()

		var rID = firstRID
		for i, hop := range route.Path {
			nxtRID, ok := idr.PopID(hop.To)
			if !ok {
				return nil, nil, nil, ErrNoKey
			}

			if i == 0 {
				rule := routing.ForwardRule(route.KeepAlive, rID, nxtRID, hop.TpID, dstPK, srcPort, dstPort)
				forwardRules[hop.From] = rule
			} else {
				rule := routing.IntermediaryForwardRule(route.KeepAlive, rID, nxtRID, hop.TpID)
				intermediaryRules[hop.From] = append(intermediaryRules[hop.From], rule)
			}

			rID = nxtRID
		}

		rule := routing.ConsumeRule(route.KeepAlive, rID, dstPK, srcPort, dstPort)
		consumeRules[dstPK] = rule
	}

	return forwardRules, consumeRules, intermediaryRules, nil
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
