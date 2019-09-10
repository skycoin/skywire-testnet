package router

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/skycoin/dmsg/cipher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/setup"
	"github.com/skycoin/skywire/pkg/snet/snettest"
)

func TestNewRouteManager(t *testing.T) {
	pk, sk := cipher.GenerateKeyPair()

	env := snettest.NewEnv(t, []snettest.KeyPair{{PK: pk, SK: sk}})
	defer env.Teardown()

	rt := routing.InMemoryRoutingTable()

	rm, err := newRouteManager(env.Nets[0], rt, RMConfig{})
	require.NoError(t, err)
	defer func() { require.NoError(t, rm.Close()) }()

	// CLOSURE: Delete all routing rules.
	clearRules := func() {
		var rules []routing.RouteID
		assert.NoError(t, rt.RangeRules(func(routeID routing.RouteID, _ routing.Rule) (next bool) {
			rules = append(rules[0:], routeID)
			return true
		}))
		assert.NoError(t, rt.DeleteRules(rules...))
	}

	// TEST: Set and get expired and unexpired rule.
	t.Run("GetRule", func(t *testing.T) {
		defer clearRules()

		expiredRule := routing.IntermediaryForwardRule(-10*time.Minute, 1, 3, uuid.New())
		expiredID, err := rm.rt.AddRule(expiredRule)
		require.NoError(t, err)

		rule := routing.IntermediaryForwardRule(10*time.Minute, 2, 3, uuid.New())
		id, err := rm.rt.AddRule(rule)
		require.NoError(t, err)

		// rule should already be expired at this point due to the execution time.
		// However, we'll just a bit to be sure
		time.Sleep(1 * time.Millisecond)

		_, err = rm.GetRule(expiredID)
		require.Error(t, err)

		_, err = rm.GetRule(123)
		require.Error(t, err)

		r, err := rm.GetRule(id)
		require.NoError(t, err)
		assert.Equal(t, rule, r)
	})

	// TEST: Ensure removing loop rules work properly.
	t.Run("RemoveLoopRule", func(t *testing.T) {
		defer clearRules()

		pk, _ := cipher.GenerateKeyPair()
		rule := routing.ConsumeRule(10*time.Minute, 1, pk, 2, 3)
		_, err := rt.AddRule(rule)
		require.NoError(t, err)

		loop := routing.Loop{Local: routing.Addr{Port: 3}, Remote: routing.Addr{PubKey: pk, Port: 3}}
		require.NoError(t, rm.RemoveLoopRule(loop))
		assert.Equal(t, 1, rt.Count())

		loop = routing.Loop{Local: routing.Addr{Port: 2}, Remote: routing.Addr{PubKey: pk, Port: 3}}
		require.NoError(t, rm.RemoveLoopRule(loop))
		assert.Equal(t, 0, rt.Count())
	})

	// TEST: Ensure AddRule and DeleteRule requests from a SetupNode does as expected.
	t.Run("AddRemoveRule", func(t *testing.T) {
		defer clearRules()

		// Add/Remove rules multiple times.
		for i := 0; i < 5; i++ {
			// As setup connections close after a single request completes
			// So we need two pairs of connections.
			requestIDIn, requestIDOut := net.Pipe()
			addIn, addOut := net.Pipe()
			delIn, delOut := net.Pipe()
			errCh := make(chan error, 2)
			go func() {
				errCh <- rm.handleSetupConn(requestIDOut) // Receive RequestRegistrationID request.
				errCh <- rm.handleSetupConn(addOut)       // Receive AddRule request.
				errCh <- rm.handleSetupConn(delOut)       // Receive DeleteRule request.
				close(errCh)
			}()

			// TODO: remove defer from for loop
			defer func() {
				require.NoError(t, requestIDIn.Close())
				require.NoError(t, addIn.Close())
				require.NoError(t, delIn.Close())
				for err := range errCh {
					require.NoError(t, err)
				}
			}()

			// Emulate SetupNode sending RequestRegistrationID request.
			ids, err := setup.RequestRouteIDs(context.TODO(), setup.NewSetupProtocol(requestIDIn), 1)
			require.NoError(t, err)

			// Emulate SetupNode sending AddRule request.
			rule := routing.IntermediaryForwardRule(10*time.Minute, ids[0], 3, uuid.New())
			err = setup.AddRules(context.TODO(), setup.NewSetupProtocol(addIn), []routing.Rule{rule})
			require.NoError(t, err)

			// Check routing table state after AddRule.
			assert.Equal(t, 1, rt.Count())
			r, err := rt.Rule(ids[0])
			require.NoError(t, err)
			assert.Equal(t, rule, r)

			// Emulate SetupNode sending RemoveRule request.
			require.NoError(t, setup.DeleteRule(context.TODO(), setup.NewSetupProtocol(delIn), ids[0]))

			// Check routing table state after DeleteRule.
			assert.Equal(t, 0, rt.Count())
			r, err = rt.Rule(ids[0])
			assert.Error(t, err)
			assert.Nil(t, r)
		}
	})

	// TEST: Ensure DeleteRule requests from SetupNode is handled properly.
	t.Run("DeleteRules", func(t *testing.T) {
		defer clearRules()

		in, out := net.Pipe()
		errCh := make(chan error, 1)
		go func() {
			errCh <- rm.handleSetupConn(out)
			close(errCh)
		}()
		defer func() {
			require.NoError(t, in.Close())
			require.NoError(t, <-errCh)
		}()

		proto := setup.NewSetupProtocol(in)

		rule := routing.IntermediaryForwardRule(10*time.Minute, 1, 3, uuid.New())
		id, err := rt.AddRule(rule)
		require.NoError(t, err)
		assert.Equal(t, 1, rt.Count())

		require.NoError(t, setup.DeleteRule(context.TODO(), proto, id))
		assert.Equal(t, 0, rt.Count())
	})

	// TEST: Ensure ConfirmLoop request from SetupNode is handled properly.
	t.Run("ConfirmLoop", func(t *testing.T) {
		defer clearRules()

		var inLoop routing.Loop
		var inRule routing.Rule

		rm.conf.OnConfirmLoop = func(loop routing.Loop, rule routing.Rule) (err error) {
			inLoop = loop
			inRule = rule
			return nil
		}
		defer func() { rm.conf.OnConfirmLoop = nil }()

		in, out := net.Pipe()
		errCh := make(chan error, 1)
		go func() {
			errCh <- rm.handleSetupConn(out)
			close(errCh)
		}()
		defer func() {
			require.NoError(t, in.Close())
			require.NoError(t, <-errCh)
		}()

		proto := setup.NewSetupProtocol(in)
		pk, _ := cipher.GenerateKeyPair()
		rule := routing.ConsumeRule(10*time.Minute, 2, pk, 2, 3)
		require.NoError(t, rt.SetRule(2, rule))

		rule = routing.IntermediaryForwardRule(10*time.Minute, 1, 3, uuid.New())
		require.NoError(t, rt.SetRule(1, rule))

		ld := routing.LoopData{
			Loop: routing.Loop{
				Remote: routing.Addr{
					PubKey: pk,
					Port:   3,
				},
				Local: routing.Addr{
					Port: 2,
				},
			},
			RouteID: 1,
		}
		err := setup.ConfirmLoop(context.TODO(), proto, ld)
		require.NoError(t, err)
		assert.Equal(t, rule, inRule)
		assert.Equal(t, routing.Port(2), inLoop.Local.Port)
		assert.Equal(t, routing.Port(3), inLoop.Remote.Port)
		assert.Equal(t, pk, inLoop.Remote.PubKey)
	})

	// TEST: Ensure LoopClosed request from SetupNode is handled properly.
	t.Run("LoopClosed", func(t *testing.T) {
		defer clearRules()

		var inLoop routing.Loop

		rm.conf.OnLoopClosed = func(loop routing.Loop) error {
			inLoop = loop
			return nil
		}
		defer func() { rm.conf.OnLoopClosed = nil }()

		in, out := net.Pipe()
		errCh := make(chan error, 1)
		go func() {
			errCh <- rm.handleSetupConn(out)
			close(errCh)
		}()
		defer func() {
			require.NoError(t, in.Close())
			require.NoError(t, <-errCh)
		}()

		proto := setup.NewSetupProtocol(in)
		pk, _ := cipher.GenerateKeyPair()

		rule := routing.ConsumeRule(10*time.Minute, 0, pk, 2, 3)
		require.NoError(t, rt.SetRule(2, rule))

		rule = routing.IntermediaryForwardRule(10*time.Minute, 1, 3, uuid.New())
		require.NoError(t, rt.SetRule(1, rule))

		ld := routing.LoopData{
			Loop: routing.Loop{
				Remote: routing.Addr{
					PubKey: pk,
					Port:   3,
				},
				Local: routing.Addr{
					Port: 2,
				},
			},
			RouteID: 1,
		}
		require.NoError(t, setup.LoopClosed(context.TODO(), proto, ld))
		assert.Equal(t, routing.Port(2), inLoop.Local.Port)
		assert.Equal(t, routing.Port(3), inLoop.Remote.Port)
		assert.Equal(t, pk, inLoop.Remote.PubKey)
	})
}
