package routing

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/skycoin/dmsg/cipher"
	"github.com/stretchr/testify/assert"
)

func TestAppRule(t *testing.T) {
	keepAlive := 2 * time.Minute
	pk, _ := cipher.GenerateKeyPair()
	rule := AppRule(keepAlive, 1, 2, pk, 4, 3)

	assert.Equal(t, keepAlive, rule.KeepAlive())
	assert.Equal(t, RuleApp, rule.Type())
	assert.Equal(t, RouteID(2), rule.RouteID())
	assert.Equal(t, pk, rule.RemotePK())
	assert.Equal(t, Port(3), rule.RemotePort())
	assert.Equal(t, Port(4), rule.LocalPort())

	rule.SetRouteID(3)
	assert.Equal(t, RouteID(3), rule.RouteID())
}

func TestForwardRule(t *testing.T) {
	trID := uuid.New()
	keepAlive := 2 * time.Minute
	rule := ForwardRule(keepAlive, 2, trID, 1)

	assert.Equal(t, keepAlive, rule.KeepAlive())
	assert.Equal(t, RuleForward, rule.Type())
	assert.Equal(t, RouteID(2), rule.RouteID())
	assert.Equal(t, trID, rule.TransportID())

	rule.SetRouteID(3)
	assert.Equal(t, RouteID(3), rule.RouteID())
}
