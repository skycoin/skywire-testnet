package router

import (
	"testing"
	"time"

	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/pkg/routing"
)

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
