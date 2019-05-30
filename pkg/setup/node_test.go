package setup

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/skycoin/skywire/pkg/dms"
	"sync"
	"testing"
	"time"

	"github.com/skycoin/skywire/pkg/metrics"

	"github.com/skycoin/skycoin/src/util/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/pkg/cipher"
	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/transport"
)

func TestCreateLoop(t *testing.T) {
	client := transport.NewDiscoveryMock()
	logStore := transport.InMemoryTransportLogStore()

	pk1, sk1 := cipher.GenerateKeyPair()
	pk2, sk2 := cipher.GenerateKeyPair()
	pk3, sk3 := cipher.GenerateKeyPair()
	pk4, sk4 := cipher.GenerateKeyPair()
	pkS, skS := cipher.GenerateKeyPair()

	c1 := &transport.ManagerConfig{PubKey: pk1, SecKey: sk1, DiscoveryClient: client, LogStore: logStore}
	c2 := &transport.ManagerConfig{PubKey: pk2, SecKey: sk2, DiscoveryClient: client, LogStore: logStore}
	c3 := &transport.ManagerConfig{PubKey: pk3, SecKey: sk3, DiscoveryClient: client, LogStore: logStore}
	c4 := &transport.ManagerConfig{PubKey: pk4, SecKey: sk4, DiscoveryClient: client, LogStore: logStore}
	cS := &transport.ManagerConfig{PubKey: pkS, SecKey: skS, DiscoveryClient: client, LogStore: logStore}

	f1, f2 := transport.NewMockFactoryPair(pk1, pk2)
	f3, f4 := transport.NewMockFactoryPair(pk2, pk3)
	f3.SetType("mock2")
	f4.SetType("mock2")

	fs1, fs2 := transport.NewMockFactoryPair(pk1, pkS)
	fs1.SetType(dms.TpType)
	fs2.SetType(dms.TpType)
	fs3, fs4 := transport.NewMockFactoryPair(pk2, pkS)
	fs3.SetType(dms.TpType)
	fs5, fs6 := transport.NewMockFactoryPair(pk3, pkS)
	fs5.SetType(dms.TpType)
	fs7, fs8 := transport.NewMockFactoryPair(pk4, pkS)
	fs7.SetType(dms.TpType)

	fS := newMuxFactory(pkS, dms.TpType, map[cipher.PubKey]transport.Factory{pk1: fs2, pk2: fs4, pk3: fs6, pk4: fs8})

	m1, err := transport.NewManager(c1, f1, fs1)
	require.NoError(t, err)

	m2, err := transport.NewManager(c2, f2, f3, fs3)
	require.NoError(t, err)

	m3, err := transport.NewManager(c3, f4, fs5)
	require.NoError(t, err)

	m4, err := transport.NewManager(c4, fs7)
	require.NoError(t, err)

	mS, err := transport.NewManager(cS, fS)
	require.NoError(t, err)

	n1 := newMockNode(m1)
	go n1.serve() // nolint: errcheck
	n2 := newMockNode(m2)
	go n2.serve() // nolint: errcheck
	n3 := newMockNode(m3)
	go n3.serve() // nolint: errcheck

	tr1, err := m1.CreateTransport(context.TODO(), pk2, "mock", true)
	require.NoError(t, err)

	tr3, err := m3.CreateTransport(context.TODO(), pk2, "mock2", true)
	require.NoError(t, err)

	l := &routing.Loop{LocalPort: 1, RemotePort: 2, Expiry: time.Now().Add(time.Hour),
		Forward: routing.Route{
			&routing.Hop{From: pk1, To: pk2, Transport: tr1.ID},
			&routing.Hop{From: pk2, To: pk3, Transport: tr3.ID},
		},
		Reverse: routing.Route{
			&routing.Hop{From: pk3, To: pk2, Transport: tr3.ID},
			&routing.Hop{From: pk2, To: pk1, Transport: tr1.ID},
		},
	}

	time.Sleep(100 * time.Millisecond)

	sn := &Node{logging.MustGetLogger("routesetup"), mS, nil, 0, metrics.NewDummy()}
	errChan := make(chan error)
	go func() {
		errChan <- sn.Serve(context.TODO())
	}()

	tr, err := m4.CreateTransport(context.TODO(), pkS, dms.TpType, false)
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
	assert.Equal(t, tr1.ID, rule.TransportID())
	assert.Equal(t, routing.RouteID(2), rule.RouteID())

	rules = n2.getRules()
	require.Len(t, rules, 2)
	rule = rules[1]
	assert.Equal(t, routing.RuleForward, rule.Type())
	assert.Equal(t, tr1.ID, rule.TransportID())
	assert.Equal(t, routing.RouteID(1), rule.RouteID())
	rule = rules[2]
	assert.Equal(t, routing.RuleForward, rule.Type())
	assert.Equal(t, tr3.ID, rule.TransportID())
	assert.Equal(t, routing.RouteID(2), rule.RouteID())

	rules = n3.getRules()
	require.Len(t, rules, 2)
	rule = rules[1]
	assert.Equal(t, routing.RuleForward, rule.Type())
	assert.Equal(t, tr3.ID, rule.TransportID())
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
	client := transport.NewDiscoveryMock()
	logStore := transport.InMemoryTransportLogStore()

	pk1, sk1 := cipher.GenerateKeyPair()
	pk3, sk3 := cipher.GenerateKeyPair()
	pkS, skS := cipher.GenerateKeyPair()

	c1 := &transport.ManagerConfig{PubKey: pk1, SecKey: sk1, DiscoveryClient: client, LogStore: logStore}
	c3 := &transport.ManagerConfig{PubKey: pk3, SecKey: sk3, DiscoveryClient: client, LogStore: logStore}
	cS := &transport.ManagerConfig{PubKey: pkS, SecKey: skS, DiscoveryClient: client, LogStore: logStore}

	fs1, fs2 := transport.NewMockFactoryPair(pk1, pkS)
	fs1.SetType(dms.TpType)
	fs2.SetType(dms.TpType)
	fs5, fs6 := transport.NewMockFactoryPair(pk3, pkS)
	fs5.SetType(dms.TpType)

	fS := newMuxFactory(pkS, dms.TpType, map[cipher.PubKey]transport.Factory{pk1: fs2, pk3: fs6})

	m1, err := transport.NewManager(c1, fs1)
	require.NoError(t, err)

	m3, err := transport.NewManager(c3, fs5)
	require.NoError(t, err)

	mS, err := transport.NewManager(cS, fS)
	require.NoError(t, err)

	n3 := newMockNode(m3)
	go n3.serve() // nolint: errcheck

	time.Sleep(100 * time.Millisecond)

	sn := &Node{logging.MustGetLogger("routesetup"), mS, nil, 0, metrics.NewDummy()}
	errChan := make(chan error)
	go func() {
		errChan <- sn.Serve(context.TODO())
	}()

	n3.setRule(1, routing.AppRule(time.Now(), 2, pk1, 1, 2))
	rules := n3.getRules()
	require.Len(t, rules, 1)

	tr, err := m1.CreateTransport(context.TODO(), pkS, dms.TpType, false)
	require.NoError(t, err)

	proto := NewSetupProtocol(tr)
	require.NoError(t, CloseLoop(proto, &LoopData{RemotePK: pk3, RemotePort: 2, LocalPort: 1}))

	rules = n3.getRules()
	require.Len(t, rules, 0)
	require.Nil(t, rules[1])

	require.NoError(t, sn.Close())
	require.NoError(t, <-errChan)
}

type muxFactory struct {
	pk        cipher.PubKey
	fType     string
	factories map[cipher.PubKey]transport.Factory
}

func newMuxFactory(pk cipher.PubKey, fType string, factories map[cipher.PubKey]transport.Factory) *muxFactory {
	return &muxFactory{pk, fType, factories}
}

func (f *muxFactory) Accept(ctx context.Context) (transport.Transport, error) {
	trChan := make(chan transport.Transport)
	defer close(trChan)

	errChan := make(chan error)

	for _, factory := range f.factories {
		go func(ff transport.Factory) {
			tr, err := ff.Accept(ctx)
			if err != nil {
				errChan <- err
			} else {
				trChan <- tr
			}
		}(factory)
	}

	select {
	case tr := <-trChan:
		return tr, nil
	case err := <-errChan:
		return nil, err
	}
}

func (f *muxFactory) Dial(ctx context.Context, remote cipher.PubKey) (transport.Transport, error) {
	return f.factories[remote].Dial(ctx, remote)
}

func (f *muxFactory) Close() error {
	var err error
	for _, factory := range f.factories {
		if fErr := factory.Close(); err == nil && fErr != nil {
			err = fErr
		}
	}

	return err
}

func (f *muxFactory) Local() cipher.PubKey {
	return f.pk
}

func (f *muxFactory) Type() string {
	return f.fType
}

type mockNode struct {
	sync.Mutex
	rules map[routing.RouteID]routing.Rule
	tm    *transport.Manager
}

func newMockNode(tm *transport.Manager) *mockNode {
	return &mockNode{tm: tm, rules: make(map[routing.RouteID]routing.Rule)}
}

func (n *mockNode) serve() error {
	go func() {
		for tr := range n.tm.TrChan {
			go func(t transport.Transport) { n.serveTransport(t) }(tr) // nolint: errcheck
		}
	}()

	return n.tm.Serve(context.Background())
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
