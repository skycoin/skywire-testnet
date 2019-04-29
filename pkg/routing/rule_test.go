package routing

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/skycoin/skywire/pkg/cipher"
)

func ExampleAppRule() {
	expireAt := time.Now().Add(2 * time.Minute)
	pk, _ := cipher.GenerateKeyPair()
	appRule := AppRule(expireAt, 2, pk, 3, 4)

	fmt.Printf("len(appRule): %v\n", len(appRule))

	// Output: len(appRule): 50
}

func ExampleForwardRule() {
	trID := uuid.New()
	expireAt := time.Now().Add(2 * time.Minute)
	fwdRule := ForwardRule(expireAt, 2, trID)
	fmt.Printf("len(fwdRule): %v\n", len(fwdRule))

	// Output: len(fwdRule): 29
}

func TestAppRule(t *testing.T) {
	expireAt := time.Now().Add(2 * time.Minute)
	pk, _ := cipher.GenerateKeyPair()
	rule := AppRule(expireAt, 2, pk, 3, 4)

	assert.Equal(t, expireAt.Unix(), rule.Expiry().Unix())
	assert.Equal(t, RuleApp, rule.Type())
	assert.Equal(t, RouteID(2), rule.RouteID())
	assert.Equal(t, pk, rule.RemotePK())
	assert.Equal(t, uint16(3), rule.RemotePort())
	assert.Equal(t, uint16(4), rule.LocalPort())

	rule.SetRouteID(3)
	assert.Equal(t, RouteID(3), rule.RouteID())
}

func TestForwardRule(t *testing.T) {
	trID := uuid.New()
	expireAt := time.Now().Add(2 * time.Minute)
	rule := ForwardRule(expireAt, 2, trID)

	assert.Equal(t, expireAt.Unix(), rule.Expiry().Unix())
	assert.Equal(t, RuleForward, rule.Type())
	assert.Equal(t, RouteID(2), rule.RouteID())
	assert.Equal(t, trID, rule.TransportID())

	rule.SetRouteID(3)
	assert.Equal(t, RouteID(3), rule.RouteID())
}
