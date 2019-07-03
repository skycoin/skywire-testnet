package router

import (
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/skycoin/src/util/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/pkg/app"
	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/setup"
)

func TestRouteManagerGetRule(t *testing.T) {
	rt := manageRoutingTable(routing.InMemoryRoutingTable())
	rm := &routeManager{logging.MustGetLogger("routesetup"), rt, nil}

	expiredRule := routing.ForwardRule(time.Now().Add(-10*time.Minute), 3, uuid.New())
	expiredID, err := rt.AddRule(expiredRule)
	require.NoError(t, err)

	rule := routing.ForwardRule(time.Now().Add(10*time.Minute), 3, uuid.New())
	id, err := rt.AddRule(rule)
	require.NoError(t, err)

	_, err = rm.GetRule(expiredID)
	require.Error(t, err)

	_, err = rm.GetRule(123)
	require.Error(t, err)

	r, err := rm.GetRule(id)
	require.NoError(t, err)
	assert.Equal(t, rule, r)
}

func TestRouteManagerRemoveLoopRule(t *testing.T) {
	rt := manageRoutingTable(routing.InMemoryRoutingTable())
	rm := &routeManager{logging.MustGetLogger("routesetup"), rt, nil}

	pk, _ := cipher.GenerateKeyPair()
	rule := routing.AppRule(time.Now(), 3, pk, 3, 2)
	_, err := rt.AddRule(rule)
	require.NoError(t, err)

	addr := &app.LoopAddr{Port: 3, Remote: app.Addr{PubKey: pk, Port: 3}}
	require.NoError(t, rm.RemoveLoopRule(addr))
	assert.Equal(t, 1, rt.Count())

	addr = &app.LoopAddr{Port: 2, Remote: app.Addr{PubKey: pk, Port: 3}}
	require.NoError(t, rm.RemoveLoopRule(addr))
	assert.Equal(t, 0, rt.Count())
}

func TestRouteManagerAddRemoveRule(t *testing.T) {
	done := make(chan struct{})
	expired := time.NewTimer(time.Second * 5)
	go func() {
		select {
		case <-done:
			return
		case <-expired.C:
		}
	}()
	defer func() {
		close(done)
	}()
	rt := manageRoutingTable(routing.InMemoryRoutingTable())
	rm := &routeManager{logging.MustGetLogger("routesetup"), rt, nil}

	in, out := net.Pipe()
	errCh := make(chan error)
	go func() {
		errCh <- rm.Serve(out)
	}()

	proto := setup.NewSetupProtocol(in)

	rule := routing.ForwardRule(time.Now(), 3, uuid.New())
	id, err := setup.AddRule(proto, rule)
	require.NoError(t, err)
	assert.Equal(t, routing.RouteID(1), id)

	assert.Equal(t, 1, rt.Count())
	r, err := rt.Rule(id)
	require.NoError(t, err)
	assert.Equal(t, rule, r)

	require.NoError(t, in.Close())
	require.NoError(t, <-errCh)
}

func TestRouteManagerDeleteRules(t *testing.T) {
	rt := manageRoutingTable(routing.InMemoryRoutingTable())
	rm := &routeManager{logging.MustGetLogger("routesetup"), rt, nil}

	in, out := net.Pipe()
	errCh := make(chan error)
	go func() {
		errCh <- rm.Serve(out)
	}()

	proto := setup.NewSetupProtocol(in)

	rule := routing.ForwardRule(time.Now(), 3, uuid.New())
	id, err := rt.AddRule(rule)
	require.NoError(t, err)
	assert.Equal(t, 1, rt.Count())

	require.NoError(t, setup.DeleteRule(proto, id))
	assert.Equal(t, 0, rt.Count())

	require.NoError(t, in.Close())
	require.NoError(t, <-errCh)
}

func TestRouteManagerConfirmLoop(t *testing.T) {
	rt := manageRoutingTable(routing.InMemoryRoutingTable())
	var inAddr *app.LoopAddr
	var inRule routing.Rule
	callbacks := &setupCallbacks{
		ConfirmLoop: func(addr *app.LoopAddr, rule routing.Rule) (err error) {
			inAddr = addr
			inRule = rule
			return nil
		},
	}
	rm := &routeManager{logging.MustGetLogger("routesetup"), rt, callbacks}

	in, out := net.Pipe()
	errCh := make(chan error)
	go func() {
		errCh <- rm.Serve(out)
	}()

	proto := setup.NewSetupProtocol(in)
	pk, _ := cipher.GenerateKeyPair()
	rule := routing.AppRule(time.Now(), 3, pk, 3, 2)
	require.NoError(t, rt.SetRule(2, rule))

	rule = routing.ForwardRule(time.Now(), 3, uuid.New())
	require.NoError(t, rt.SetRule(1, rule))

	ld := &setup.LoopData{
		RemotePK:     pk,
		RemotePort:   3,
		LocalPort:    2,
		RouteID:      1,
		NoiseMessage: []byte("bar"),
	}
	noiseRes, err := setup.ConfirmLoop(proto, ld)
	require.NoError(t, err)
	assert.Equal(t, []byte("foo"), noiseRes)
	assert.Equal(t, rule, inRule)
	assert.Equal(t, uint16(2), inAddr.Port)
	assert.Equal(t, uint16(3), inAddr.Remote.Port)
	assert.Equal(t, pk, inAddr.Remote.PubKey)

	require.NoError(t, in.Close())
	require.NoError(t, <-errCh)
}

func TestRouteManagerLoopClosed(t *testing.T) {
	rt := manageRoutingTable(routing.InMemoryRoutingTable())
	var inAddr *app.LoopAddr
	callbacks := &setupCallbacks{
		LoopClosed: func(addr *app.LoopAddr) error {
			inAddr = addr
			return nil
		},
	}
	rm := &routeManager{logging.MustGetLogger("routesetup"), rt, callbacks}

	in, out := net.Pipe()
	errCh := make(chan error)
	go func() {
		errCh <- rm.Serve(out)
	}()

	proto := setup.NewSetupProtocol(in)

	pk, _ := cipher.GenerateKeyPair()

	rule := routing.AppRule(time.Now(), 3, pk, 3, 2)
	require.NoError(t, rt.SetRule(2, rule))

	rule = routing.ForwardRule(time.Now(), 3, uuid.New())
	require.NoError(t, rt.SetRule(1, rule))

	ld := &setup.LoopData{
		RemotePK:     pk,
		RemotePort:   3,
		LocalPort:    2,
		RouteID:      1,
		NoiseMessage: []byte("bar"),
	}
	require.NoError(t, setup.LoopClosed(proto, ld))
	assert.Equal(t, uint16(2), inAddr.Port)
	assert.Equal(t, uint16(3), inAddr.Remote.Port)
	assert.Equal(t, pk, inAddr.Remote.PubKey)

	require.NoError(t, in.Close())
	require.NoError(t, <-errCh)
}
