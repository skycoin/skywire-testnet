package router

// func TestRouteManagerAddRemoveRule(t *testing.T) {
// 	done := make(chan struct{})
// 	expired := time.NewTimer(time.Second * 5)
// 	go func() {
// 		select {
// 		case <-done:
// 			return
// 		case <-expired.C:
// 		}
// 	}()
// 	defer func() {
// 		close(done)
// 	}()
// 	rt := NewRoutingTableManager(
// 		logging.MustGetLogger("rt_manager"),
// 		routing.InMemoryRoutingTable(),
// 		DefaultRouteKeepalive,
// 		DefaultRouteCleanupDuration)

// 	in, out := net.Pipe()
// 	errCh := make(chan error)
// 	go func() {
// 		errCh <- rt.handleSetupNode(out)
// 	}()

// 	proto := setup.NewSetupProtocol(in)

// 	rule := routing.ForwardRule(time.Now(), 3, uuid.New())
// 	id, err := setup.AddRule(proto, rule)
// 	require.NoError(t, err)
// 	assert.Equal(t, routing.RouteID(1), id)

// 	assert.Equal(t, 1, rt.Count())
// 	r, err := rt.Rule(id)
// 	require.NoError(t, err)
// 	assert.Equal(t, rule, r)

// 	require.NoError(t, in.Close())
// 	require.NoError(t, <-errCh)
// }

//
//func TestRouteManagerDeleteRules(t *testing.T) {
//	rt := NewRoutingTableManager(routing.InMemoryRoutingTable())
//	rm := &setupManager{logging.MustGetLogger("routesetup"), rt, nil}
//
//	in, out := net.Pipe()
//	errCh := make(chan error)
//	go func() {
//		errCh <- rm.handleSetupNode(out)
//	}()
//
//	proto := setup.NewSetupProtocol(in)
//
//	rule := routing.ForwardRule(time.Now(), 3, uuid.New())
//	id, err := rt.AddRule(rule)
//	require.NoError(t, err)
//	assert.Equal(t, 1, rt.Count())
//
//	require.NoError(t, setup.DeleteRule(proto, id))
//	assert.Equal(t, 0, rt.Count())
//
//	require.NoError(t, in.Close())
//	require.NoError(t, <-errCh)
//}

// TODO(evanlinjin): re-implement the tests below.
//func TestRouteManagerConfirmLoop(t *testing.T) {
//	rtm := NewRoutingTableManager(routing.InMemoryRoutingTable())
//	var inAddr *app.LoopMeta
//	var inRule routing.Rule
//	var noiseMsg []byte
//	callbacks := &setupCallbacks{
//		ConfirmLoop: func(addr *app.LoopMeta, rule routing.Rule, nMsg []byte) (noiseRes []byte, err error) {
//			inAddr = addr
//			inRule = rule
//			noiseMsg = nMsg
//			return []byte("foo"), nil
//		},
//	}
//	rm := &setupManager{logging.MustGetLogger("routesetup"), rtm, callbacks}
//
//	in, out := net.Pipe()
//	errCh := make(chan error)
//	go func() {
//		errCh <- rm.Serve(out)
//	}()
//
//	proto := setup.NewProtocol(in)
//	pk, _ := cipher.GenerateKeyPair()
//	rule := routing.AppRule(time.Now(), 3, pk, 3, 2)
//	require.NoError(t, rtm.SetRule(2, rule))
//
//	rule = routing.ForwardRule(time.Now(), 3, uuid.New())
//	require.NoError(t, rtm.SetRule(1, rule))
//
//	ld := &setup.LoopData{
//		RemotePK:     pk,
//		RemotePort:   3,
//		LocalPort:    2,
//		RouteID:      1,
//		NoiseMessage: []byte("bar"),
//	}
//	noiseRes, err := setup.ConfirmLoop(proto, ld)
//	require.NoError(t, err)
//	assert.Equal(t, []byte("foo"), noiseRes)
//	assert.Equal(t, []byte("bar"), noiseMsg)
//	assert.Equal(t, rule, inRule)
//	assert.Equal(t, uint16(2), inAddr.Local.Port)
//	assert.Equal(t, uint16(3), inAddr.Remote.Port)
//	assert.Equal(t, pk, inAddr.Remote.PubKey)
//
//	require.NoError(t, in.Close())
//	require.NoError(t, <-errCh)
//}

//func TestRouteManagerLoopClosed(t *testing.T) {
//	rtm := NewRoutingTableManager(routing.InMemoryRoutingTable())
//	var inAddr *app.LoopMeta
//	callbacks := &setupCallbacks{
//		LoopClosed: func(addr *app.LoopMeta) error {
//			inAddr = addr
//			return nil
//		},
//	}
//	rm := &setupManager{logging.MustGetLogger("routesetup"), rtm, callbacks}
//
//	in, out := net.Pipe()
//	errCh := make(chan error)
//	go func() {
//		errCh <- rm.Serve(out)
//	}()
//
//	proto := setup.NewProtocol(in)
//
//	pk, _ := cipher.GenerateKeyPair()
//
//	rule := routing.AppRule(time.Now(), 3, pk, 3, 2)
//	require.NoError(t, rtm.SetRule(2, rule))
//
//	rule = routing.ForwardRule(time.Now(), 3, uuid.New())
//	require.NoError(t, rtm.SetRule(1, rule))
//
//	ld := &setup.LoopData{
//		RemotePK:     pk,
//		RemotePort:   3,
//		LocalPort:    2,
//		RouteID:      1,
//		NoiseMessage: []byte("bar"),
//	}
//	require.NoError(t, setup.LoopClosed(proto, ld))
//	assert.Equal(t, uint16(2), inAddr.Local.Port)
//	assert.Equal(t, uint16(3), inAddr.Remote.Port)
//	assert.Equal(t, pk, inAddr.Remote.PubKey)
//
//	require.NoError(t, in.Close())
//	require.NoError(t, <-errCh)
//}
