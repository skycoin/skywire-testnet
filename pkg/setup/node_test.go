package setup

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/dmsg/disc"
	"github.com/skycoin/skycoin/src/util/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/pkg/metrics"
	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/transport"
	"github.com/skycoin/skywire/pkg/transport/dmsg"
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

func TestCreateLoop(t *testing.T) {
	dc := disc.NewMock()

	pk1, sk1 := cipher.GenerateKeyPair()
	pk2, sk2 := cipher.GenerateKeyPair()
	pk3, sk3 := cipher.GenerateKeyPair()
	pk4, sk4 := cipher.GenerateKeyPair()
	pkS, _ := cipher.GenerateKeyPair()

	c1 := dmsg.NewClient(pk1, sk1, dc)
	c2 := dmsg.NewClient(pk2, sk2, dc)
	c3 := dmsg.NewClient(pk3, sk3, dc)
	c4 := dmsg.NewClient(pk4, sk4, dc)

	n1 := newMockNode(c1)
	go n1.serve()
	n2 := newMockNode(c2)
	go n2.serve()
	n3 := newMockNode(c3)
	go n3.serve()

	_, err := c1.Dial(context.TODO(), pk2)
	require.NoError(t, err)

	_, err = c3.Dial(context.TODO(), pk2)
	require.NoError(t, err)

	l := &routing.Loop{LocalPort: 1, RemotePort: 2, Expiry: time.Now().Add(time.Hour),
		Forward: routing.Route{
			&routing.Hop{From: pk1, To: pk2, Transport: uuid.New()},
			&routing.Hop{From: pk2, To: pk3, Transport: uuid.New()},
		},
		Reverse: routing.Route{
			&routing.Hop{From: pk3, To: pk2, Transport: uuid.New()},
			&routing.Hop{From: pk2, To: pk1, Transport: uuid.New()},
		},
	}

	time.Sleep(100 * time.Millisecond)

	sn := &Node{logging.MustGetLogger("routesetup"), nil, 0, metrics.NewDummy()}
	errChan := make(chan error)
	go func() {
		errChan <- sn.Serve(context.TODO())
	}()

	tr, err := c4.Dial(context.TODO(), pkS)
	require.NoError(t, err)

	proto := NewSetupProtocol(tr)
	require.NoError(t, CreateLoop(proto, l))

	rules := n1.getRules()
	require.Len(t, rules, 2)
	rule := rules[1]
	assert.Equal(t, routing.RuleApp, rule.Type())
	assert.Equal(t, routing.RouteID(2), rule.RouteID())
	assert.Equal(t, pk3, rule.RemotePK())
	assert.Equal(t, uint16(2), rule.RemotePort())
	assert.Equal(t, uint16(1), rule.LocalPort())
	rule = rules[2]
	assert.Equal(t, routing.RuleForward, rule.Type())
	assert.Equal(t, routing.RouteID(2), rule.RouteID())

	rules = n2.getRules()
	require.Len(t, rules, 2)
	rule = rules[1]
	assert.Equal(t, routing.RuleForward, rule.Type())
	assert.Equal(t, routing.RouteID(1), rule.RouteID())
	rule = rules[2]
	assert.Equal(t, routing.RuleForward, rule.Type())
	assert.Equal(t, routing.RouteID(2), rule.RouteID())

	rules = n3.getRules()
	require.Len(t, rules, 2)
	rule = rules[1]
	assert.Equal(t, routing.RuleForward, rule.Type())
	assert.Equal(t, routing.RouteID(1), rule.RouteID())
	rule = rules[2]
	assert.Equal(t, routing.RuleApp, rule.Type())
	assert.Equal(t, routing.RouteID(1), rule.RouteID())
	assert.Equal(t, pk1, rule.RemotePK())
	assert.Equal(t, uint16(1), rule.RemotePort())
	assert.Equal(t, uint16(2), rule.LocalPort())

	require.NoError(t, sn.Close())
	require.NoError(t, <-errChan)
}

func TestCloseLoop(t *testing.T) {
	dc := disc.NewMock()

	pk1, sk1 := cipher.GenerateKeyPair()
	pk3, sk3 := cipher.GenerateKeyPair()
	pkS, _ := cipher.GenerateKeyPair()

	c1 := dmsg.NewClient(pk1, sk1, dc)
	c3 := dmsg.NewClient(pk3, sk3, dc)

	n3 := newMockNode(c3)
	go n3.serve()

	time.Sleep(100 * time.Millisecond)

	sn := &Node{logging.MustGetLogger("routesetup"), c3, 0, metrics.NewDummy()}
	errChan := make(chan error)
	go func() {
		errChan <- sn.Serve(context.TODO())
	}()

	n3.setRule(1, routing.AppRule(time.Now(), 2, pk1, 1, 2))
	rules := n3.getRules()
	require.Len(t, rules, 1)

	tr, err := c1.Dial(context.TODO(), pkS)
	require.NoError(t, err)

	proto := NewSetupProtocol(tr)
	require.NoError(t, CloseLoop(proto, &LoopData{RemotePK: pk3, RemotePort: 2, LocalPort: 1}))

	rules = n3.getRules()
	require.Len(t, rules, 0)
	require.Nil(t, rules[1])

	require.NoError(t, sn.Close())
	require.NoError(t, <-errChan)
}

type mockNode struct {
	sync.Mutex
	rules     map[routing.RouteID]routing.Rule
	messenger *dmsg.Client
}

func newMockNode(messenger *dmsg.Client) *mockNode {
	return &mockNode{messenger: messenger, rules: make(map[routing.RouteID]routing.Rule)}
}

func (n *mockNode) serve() {
	ctx := context.Background()

	for {
		tp, _ := n.messenger.Accept(ctx) // nolint:errcheck
		go func(tp transport.Transport) {
			for {
				n.serveTransport(tp) // nolint:errcheck
			}
		}(tp)
	}
}

func (n *mockNode) setRule(id routing.RouteID, rule routing.Rule) {
	n.Lock()
	n.rules[id] = rule
	n.Unlock()
}

func (n *mockNode) getRules() map[routing.RouteID]routing.Rule {
	res := make(map[routing.RouteID]routing.Rule)
	n.Lock()
	for id, rule := range n.rules {
		res[id] = rule
	}
	n.Unlock()
	return res
}

func (n *mockNode) serveTransport(tr transport.Transport) error {
	proto := NewSetupProtocol(tr)
	sp, data, err := proto.ReadPacket()
	if err != nil {
		return err
	}

	n.Lock()
	var res interface{}
	switch sp {
	case PacketAddRules:
		rules := []routing.Rule{}
		json.Unmarshal(data, &rules) // nolint: errcheck
		for _, rule := range rules {
			for i := routing.RouteID(1); i < 255; i++ {
				if n.rules[i] == nil {
					n.rules[i] = rule
					res = []routing.RouteID{i}
					break
				}
			}
		}
	case PacketConfirmLoop:
		ld := LoopData{}
		json.Unmarshal(data, &ld) // nolint: errcheck
		for _, rule := range n.rules {
			if rule.Type() == routing.RuleApp && rule.RemotePK() == ld.RemotePK &&
				rule.RemotePort() == ld.RemotePort && rule.LocalPort() == ld.LocalPort {

				rule.SetRouteID(ld.RouteID)
				break
			}
		}
	case PacketLoopClosed:
		ld := &LoopData{}
		json.Unmarshal(data, ld) // nolint: errcheck
		for routeID, rule := range n.rules {
			if rule.Type() == routing.RuleApp && rule.RemotePK() == ld.RemotePK &&
				rule.RemotePort() == ld.RemotePort && rule.LocalPort() == ld.LocalPort {

				delete(n.rules, routeID)
				break
			}
		}
	default:
		err = errors.New("unknown foundation packet")
	}
	n.Unlock()

	if err != nil {
		return proto.WritePacket(RespFailure, err)
	}

	return proto.WritePacket(RespSuccess, res)
}
