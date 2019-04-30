package router

import (
	"testing"
	"time"

	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/pkg/app"
	"github.com/skycoin/skywire/pkg/cipher"
	"github.com/skycoin/skywire/pkg/routing"
)

// var rtTestCases []func(rtm *RoutingTableManager) error {

// }

func TestManagedRoutingTableCleanup(t *testing.T) {
	rt := NewRoutingTableManager(
		logging.MustGetLogger("rt_manager"),
		routing.InMemoryRoutingTable(),
		DefaultRouteKeepalive,
		DefaultRouteCleanupDuration)

	go rt.Run()
	defer func() { rt.Stop() }()

	id, err := rt.AddRule(routing.ForwardRule(time.Now().Add(time.Hour), 3, uuid.New()))
	require.NoError(t, err)

	_, err = rt.Rule(id)
	require.NoError(t, err)

	_, err = rt.AddRule(routing.ForwardRule(time.Now().Add(-time.Hour), 3, uuid.New()))
	require.NoError(t, err)

	_, err = rt.AddRule(routing.ForwardRule(time.Now().Add(-time.Hour), 3, uuid.New()))
	require.NoError(t, err)

	assert.Equal(t, 3, rt.Count())

	assert.NotNil(t, rt.activity[id])

	require.NoError(t, rt.Cleanup())
	assert.Equal(t, 1, rt.Count())
}

func TestRouteManagerAddRule(t *testing.T) {
	// rt := NewRoutingTableManager(routing.InMemoryRoutingTable())
	rt := NewRoutingTableManager(
		logging.MustGetLogger("rt_manager"),
		routing.InMemoryRoutingTable(),
		DefaultRouteKeepalive,
		DefaultRouteCleanupDuration)

	// rm := &setupManager{logging.MustGetLogger("routesetup"), rt, nil}

	expiredRule := routing.ForwardRule(time.Now().Add(-10*time.Minute), 3, uuid.New())
	expiredID, err := rt.AddRule(expiredRule)
	require.NoError(t, err)

	rule := routing.ForwardRule(time.Now().Add(10*time.Minute), 3, uuid.New())
	id, err := rt.AddRule(rule)
	require.NoError(t, err)

	_, err = rt.Rule(expiredID)
	require.Error(t, err)

	_, err = rt.Rule(123)
	require.Error(t, err)

	r, err := rt.Rule(id)
	require.NoError(t, err)
	assert.Equal(t, rule, r)
}

func TestRouteManagerRemoveLoopRule(t *testing.T) {
	rt := NewRoutingTableManager(
		logging.MustGetLogger("rt_manager"),
		routing.InMemoryRoutingTable(),
		DefaultRouteKeepalive,
		DefaultRouteCleanupDuration)

	pk, _ := cipher.GenerateKeyPair()
	rule := routing.AppRule(time.Now(), 3, pk, 3, 2)
	_, err := rt.AddRule(rule)
	require.NoError(t, err)

	addr := app.LoopMeta{Local: app.LoopAddr{Port: 3}, Remote: app.LoopAddr{PubKey: pk, Port: 3}}
	require.NoError(t, rt.DeleteAppRule(addr))
	assert.Equal(t, 1, rt.Count())

	addr = app.LoopMeta{Local: app.LoopAddr{Port: 2}, Remote: app.LoopAddr{PubKey: pk, Port: 3}}
	require.NoError(t, rt.DeleteAppRule(addr))
	assert.Equal(t, 0, rt.Count())
}
