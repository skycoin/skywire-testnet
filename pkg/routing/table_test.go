package routing

import (
	"log"
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

	id, err := tbl.ReserveKeys(1)
	require.NoError(t, err)

	rule := IntermediaryForwardRule(15*time.Minute, id[0], 2, uuid.New())
	err = tbl.SaveRule(rule)
	require.NoError(t, err)

	assert.Equal(t, 1, tbl.Count())

	r, err := tbl.Rule(id[0])
	require.NoError(t, err)
	assert.Equal(t, rule, r)

	id2, err := tbl.ReserveKeys(1)
	require.NoError(t, err)

	rule2 := IntermediaryForwardRule(15*time.Minute, id2[0], 3, uuid.New())
	err = tbl.SaveRule(rule2)
	require.NoError(t, err)

	assert.Equal(t, 2, tbl.Count())
	require.NoError(t, tbl.SaveRule(rule))

	r, err = tbl.Rule(id[0])
	require.NoError(t, err)
	assert.Equal(t, rule, r)

	ids := make([]RouteID, 0)
	for _, rule := range tbl.AllRules() {
		ids = append(ids, rule.KeyRouteID())
	}
	require.ElementsMatch(t, []RouteID{id[0], id2[0]}, ids)

	tbl.DelRules([]RouteID{id[0], id2[0]})
	assert.Equal(t, 0, tbl.Count())
}

func TestRoutingTable(t *testing.T) {
	RoutingTableSuite(t, NewTable(DefaultConfig()))
}

func TestRoutingTableCleanup(t *testing.T) {
	rt := &memTable{
		rules:    map[RouteID]Rule{},
		activity: make(map[RouteID]time.Time),
		config:   Config{GCInterval: DefaultGCInterval},
	}

	id0, err := rt.ReserveKeys(1)
	require.NoError(t, err)
	err = rt.SaveRule(IntermediaryForwardRule(1*time.Hour, id0[0], 3, uuid.New()))
	require.NoError(t, err)

	id1, err := rt.ReserveKeys(1)
	require.NoError(t, err)
	err = rt.SaveRule(IntermediaryForwardRule(1*time.Hour, id1[0], 3, uuid.New()))
	require.NoError(t, err)

	id2, err := rt.ReserveKeys(1)
	require.NoError(t, err)
	err = rt.SaveRule(IntermediaryForwardRule(-1*time.Hour, id2[0], 3, uuid.New()))
	require.NoError(t, err)

	// rule should already be expired at this point due to the execution time.
	// However, we'll just a bit to be sure
	time.Sleep(1 * time.Millisecond)

	assert.Equal(t, 3, rt.Count())

	_, err = rt.Rule(id1[0])
	require.NoError(t, err)

	assert.NotNil(t, rt.activity[id1[0]])

	rt.gc()
	assert.Equal(t, 2, rt.Count())

	rule, err := rt.Rule(id2[0])
	require.Error(t, err)
	assert.Nil(t, rule)
}
