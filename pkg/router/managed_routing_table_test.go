package router

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/pkg/routing"
)

func TestManagedRoutingTableCleanup(t *testing.T) {
	rt := manageRoutingTable(routing.InMemoryRoutingTable())

	_, err := rt.AddRule(routing.ForwardRule(time.Now().Add(time.Hour), 3, uuid.New(), 1))
	require.NoError(t, err)

	id, err := rt.AddRule(routing.ForwardRule(time.Now().Add(-time.Hour), 3, uuid.New(), 2))
	require.NoError(t, err)

	id2, err := rt.AddRule(routing.ForwardRule(time.Now().Add(-time.Hour), 3, uuid.New(), 3))
	require.NoError(t, err)

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
