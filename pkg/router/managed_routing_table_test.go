package router

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SkycoinProject/skywire-mainnet/pkg/routing"
)

func TestManagedRoutingTableCleanup(t *testing.T) {
	rt := manageRoutingTable(routing.InMemoryRoutingTable())

	_, err := rt.AddRule(routing.ForwardRule(1*time.Hour, 3, uuid.New(), 1))
	require.NoError(t, err)

	id, err := rt.AddRule(routing.ForwardRule(1*time.Hour, 3, uuid.New(), 2))
	require.NoError(t, err)

	id2, err := rt.AddRule(routing.ForwardRule(-1*time.Hour, 3, uuid.New(), 3))
	require.NoError(t, err)

	// rule should already be expired at this point due to the execution time.
	// However, we'll just a bit to be sure
	time.Sleep(1 * time.Millisecond)

	assert.Equal(t, 3, rt.Count())

	_, err = rt.Rule(id)
	require.NoError(t, err)

	assert.NotNil(t, rt.activity[id])

	require.NoError(t, rt.Cleanup())
	assert.Equal(t, 2, rt.Count())

	rule, err := rt.Rule(id2)
	require.Error(t, err)
	assert.Nil(t, rule)
}
