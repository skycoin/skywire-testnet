package routing

import (
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/skycoin/skycoin/src/util/logging"
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

	rule := IntermediaryForwardRule(15*time.Minute, 1, 2, uuid.New())
	id, err := tbl.AddRule(rule)
	require.NoError(t, err)

	assert.Equal(t, 1, tbl.Count())

	r, err := tbl.Rule(id)
	require.NoError(t, err)
	assert.Equal(t, rule, r)

	rule2 := IntermediaryForwardRule(15*time.Minute, 2, 3, uuid.New())
	id2, err := tbl.AddRule(rule2)
	require.NoError(t, err)

	assert.Equal(t, 2, tbl.Count())

	require.NoError(t, tbl.SaveRule(id2, rule))
	r, err = tbl.Rule(id2)
	require.NoError(t, err)
	assert.Equal(t, rule, r)

	ids := make([]RouteID, 0)
	tbl.RangeRules(func(routeID RouteID, _ Rule) bool {
		ids = append(ids, routeID)
		return true
	})
	require.ElementsMatch(t, []RouteID{id, id2}, ids)

	tbl.DelRules([]RouteID{id, id2})
	assert.Equal(t, 0, tbl.Count())
}

func TestRoutingTable(t *testing.T) {
	RoutingTableSuite(t, New())
}

func TestRoutingTableCleanup(t *testing.T) {
	rt := &memTable{
		rules:    map[RouteID]Rule{},
		activity: make(map[RouteID]time.Time),
		config:   Config{GCInterval: DefaultGCInterval},
	}

	_, err := rt.AddRule(IntermediaryForwardRule(1*time.Hour, 1, 3, uuid.New()))
	require.NoError(t, err)

	id, err := rt.AddRule(IntermediaryForwardRule(1*time.Hour, 2, 3, uuid.New()))
	require.NoError(t, err)

	id2, err := rt.AddRule(IntermediaryForwardRule(-1*time.Hour, 3, 3, uuid.New()))
	require.NoError(t, err)

	// rule should already be expired at this point due to the execution time.
	// However, we'll just a bit to be sure
	time.Sleep(1 * time.Millisecond)

	assert.Equal(t, 3, rt.Count())

	_, err = rt.Rule(id)
	require.NoError(t, err)

	assert.NotNil(t, rt.activity[id])

	rt.gc()
	assert.Equal(t, 2, rt.Count())

	rule, err := rt.Rule(id2)
	require.Error(t, err)
	assert.Nil(t, rule)
}
