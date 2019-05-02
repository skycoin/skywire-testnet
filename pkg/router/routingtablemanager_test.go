package router

import (
	"fmt"
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

func mockRTM() *RoutingTableManager {
	return NewRoutingTableManager(
		logging.MustGetLogger("rt_manager"),
		routing.InMemoryRoutingTable(),
		time.Second,
		DefaultRouteCleanupDuration)
}

func TestManagedRoutingTableCleanup(t *testing.T) {
	rtm := mockRTM()

	go rtm.Run()
	defer func() { rtm.Stop() }()

	id, err := rtm.AddRule(routing.ForwardRule(time.Now().Add(time.Hour), 3, uuid.New()))
	require.NoError(t, err)

	_, err = rtm.Rule(id)
	require.NoError(t, err)

	_, err = rtm.AddRule(routing.ForwardRule(time.Now().Add(-time.Hour), 3, uuid.New()))
	require.NoError(t, err)

	_, err = rtm.AddRule(routing.ForwardRule(time.Now().Add(-time.Hour), 3, uuid.New()))
	require.NoError(t, err)

	assert.Equal(t, 3, rtm.Count())

	assert.NotNil(t, rtm.activity[id])

	require.NoError(t, rtm.Cleanup())
	assert.Equal(t, 1, rtm.Count())
}

func TestRouteManagerAddRule(t *testing.T) {
	rt := mockRTM()

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

func ExampleRoutingTableManager_FindFwdRule() {
	rtm := mockRTM()

	//TODO(alex): tests for existing loops
	rule, err := rtm.FindFwdRule(routing.RouteID(0))
	// r, err: = rtm.FindFwdRule(routing.RouteID{0})
	fmt.Printf("rule, err: %v, %v\n", rule, err)

	// Output: rule, err: %!v(PANIC=String method: runtime error: index out of range), routing table: unknown RouteID
}

func ExampleRoutingTableManager_Run() {

	rtm := mockRTM()
	rtm.ticker = time.NewTicker(10 * time.Millisecond)

	go rtm.Run()
	time.Sleep(1 * time.Second)
	fmt.Println("Success")

	// Output: Success

}
