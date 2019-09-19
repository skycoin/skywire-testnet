package routing

import (
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/SkycoinProject/skycoin/src/util/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	loggingLevel, ok := os.LookupEnv("TEST_LOGGING_LEVEL")
	if ok {
		lvl, err := logging.LevelFromString(loggingLevel)
		if err != nil {
			log.Fatal(err)
		}
		logging.SetLevel(lvl)
	} else {
		logging.Disable()
	}

	os.Exit(m.Run())
}

func RoutingTableSuite(t *testing.T, tbl Table) {
	t.Helper()

	rule := ForwardRule(15*time.Minute, 2, uuid.New(), 1)
	id, err := tbl.AddRule(rule)
	require.NoError(t, err)

	assert.Equal(t, 1, tbl.Count())

	r, err := tbl.Rule(id)
	require.NoError(t, err)
	assert.Equal(t, rule, r)

	rule2 := ForwardRule(15*time.Minute, 3, uuid.New(), 2)
	id2, err := tbl.AddRule(rule2)
	require.NoError(t, err)

	assert.Equal(t, 2, tbl.Count())

	require.NoError(t, tbl.SetRule(id2, rule))
	r, err = tbl.Rule(id2)
	require.NoError(t, err)
	assert.Equal(t, rule, r)

	ids := make([]RouteID, 0)
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
