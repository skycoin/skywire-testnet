package routing

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func RoutingTableSuite(t *testing.T, tbl Table) {
	t.Helper()

	rule := ForwardRule(time.Now(), 2, uuid.New())
	id, err := tbl.AddRule(rule)
	require.NoError(t, err)

	assert.Equal(t, 1, tbl.Count())

	r, err := tbl.Rule(id)
	require.NoError(t, err)
	assert.Equal(t, rule, r)

	rule2 := ForwardRule(time.Now(), 3, uuid.New())
	id2, err := tbl.AddRule(rule2)
	require.NoError(t, err)

	assert.Equal(t, 2, tbl.Count())

	require.NoError(t, tbl.SetRule(id2, rule))
	r, err = tbl.Rule(id2)
	require.NoError(t, err)
	assert.Equal(t, rule, r)

	ids := []RouteID{}
	err = tbl.RangeRules(func(routeID RouteID, _ Rule) bool {
		ids = append(ids, routeID)
		return true
	})
	require.NoError(t, err)
	require.ElementsMatch(t, []RouteID{id, id2}, ids)

	require.NoError(t, tbl.DeleteRules(id, id2))
	assert.Equal(t, 0, tbl.Count())
}

func TestRoutingTable(t *testing.T) {
	RoutingTableSuite(t, InMemoryRoutingTable())
}
