package routing

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/skycoin/dmsg/cipher"
	"github.com/stretchr/testify/assert"
)

func TestConsumeRule(t *testing.T) {
	keepAlive := 2 * time.Minute
	pk, _ := cipher.GenerateKeyPair()

	rule := ConsumeRule(keepAlive, 1, pk, 2, 3)

	assert.Equal(t, keepAlive, rule.KeepAlive())
	assert.Equal(t, RuleConsume, rule.Type())
	assert.Equal(t, RouteID(1), rule.KeyRouteID())
	assert.Equal(t, pk, rule.RouteDescriptor().DstPK())
	assert.Equal(t, Port(3), rule.RouteDescriptor().DstPort())
	assert.Equal(t, Port(2), rule.RouteDescriptor().SrcPort())

	rule.SetKeyRouteID(4)
	assert.Equal(t, RouteID(4), rule.KeyRouteID())
}

func TestForwardRule(t *testing.T) {
	trID := uuid.New()
	keepAlive := 2 * time.Minute
	pk, _ := cipher.GenerateKeyPair()

	rule := ForwardRule(keepAlive, 1, 2, trID, pk, 3, 4)

	assert.Equal(t, keepAlive, rule.KeepAlive())
	assert.Equal(t, RuleForward, rule.Type())
	assert.Equal(t, RouteID(1), rule.KeyRouteID())
	assert.Equal(t, RouteID(2), rule.NextRouteID())
	assert.Equal(t, trID, rule.NextTransportID())
	assert.Equal(t, pk, rule.RouteDescriptor().DstPK())
	assert.Equal(t, Port(4), rule.RouteDescriptor().DstPort())
	assert.Equal(t, Port(3), rule.RouteDescriptor().SrcPort())

	rule.SetKeyRouteID(5)
	assert.Equal(t, RouteID(5), rule.KeyRouteID())
}

func TestIntermediaryForwardRule(t *testing.T) {
	trID := uuid.New()
	keepAlive := 2 * time.Minute

	rule := IntermediaryForwardRule(keepAlive, 1, 2, trID)

	assert.Equal(t, keepAlive, rule.KeepAlive())
	assert.Equal(t, RuleIntermediaryForward, rule.Type())
	assert.Equal(t, RouteID(1), rule.KeyRouteID())
	assert.Equal(t, RouteID(2), rule.NextRouteID())
	assert.Equal(t, trID, rule.NextTransportID())

	rule.SetKeyRouteID(3)
	assert.Equal(t, RouteID(3), rule.KeyRouteID())
}
