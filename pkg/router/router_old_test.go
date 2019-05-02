package router

// func TestRouterForwarding(t *testing.T) {

// 	env, err := makeMockRouterEnv()
// 	require.NoError(t, err)

// 	errCh := make(chan error)
// 	go func() {
// 		errCh <- env.r.Serve(context.TODO(), env.pm)
// 	}()

// 	trLocal, err := env.tpmLocal.CreateTransport(context.TODO(), env.pkSetup, "messaging", true)
// 	// require.NoError(t, err)

// 	trRemote, err := env.tpmSetup.CreateTransport(context.TODO(), env.pkSetup, "messaging", true)
// 	// require.NoError(t, err)

// 	rule := routing.ForwardRule(time.Now().Add(time.Hour), 4, trRemote.ID)
// 	routeID, err := env.rt.AddRule(rule)
// 	require.NoError(t, err)

// 	time.Sleep(100 * time.Millisecond)

// 	_, err = trLocal.Write(routing.MakePacket(routeID, []byte("foo")))
// 	require.NoError(t, err)

// 	packet := make(routing.Packet, 9)
// 	_, err = trRemote.Read(packet)
// 	require.NoError(t, err)
// 	assert.Equal(t, uint16(3), packet.Size())
// 	assert.Equal(t, routing.RouteID(4), packet.RouteID())
// 	assert.Equal(t, []byte("foo"), packet.Payload())

// 	require.NoError(t, env.TearDown())
// 	// require.NoError(t, m3.Close())

// 	time.Sleep(100 * time.Millisecond)

// 	// require.NoError(t, r.Close())
// 	require.NoError(t, <-errCh)
// }

// TODO(evanlinjin): re-implement.
// func TestRouterAppInit(t *testing.T) {
// 	client := transport.NewDiscoveryMock()
// 	logStore := transport.InMemoryTransportLogStore()

// 	pk1, sk1 := cipher.GenerateKeyPair()
// 	c1 := &transport.ManagerConfig{PubKey: pk1, SecKey: sk1, DiscoveryClient: client, LogStore: logStore}

// 	m1, err := transport.NewManager(c1)
// 	require.NoError(t, err)

// 	c := &Config{
// 		log:           logging.MustGetLogger("routesetup"),
// 		PubKey:           pk1,
// 		SecKey:           sk1,
// 		TransportManager: m1,
// 	}
// 	r := New(c)
// 	rw, rwIn := net.Pipe()
// 	errCh := make(chan error)
// 	go func() {
// 		errCh <- r.ServeApp(rwIn, 10)
// 	}()

// 	proto := appnet.NewProtocol(rw)
// 	go proto.Serve(nil) // nolint: errcheck

// 	require.NoError(t, proto.Call(appnet.FrameInit, &app.Config{AppName: "foo", AppVersion: "0.0.1", ProtocolVersion: "0.0.1"}, nil))
// 	require.Error(t, proto.CallJSON(appnet.FrameInit, &app.Config{AppName: "foo1", AppVersion: "0.0.1", ProtocolVersion: "0.0.1"}, nil))

// 	require.NoError(t, proto.Close())
// 	require.NoError(t, r.Close())
// 	require.NoError(t, <-errCh)
// }

// func TestRouterApp(t *testing.T) {
// 	client := transport.NewDiscoveryMock()
// 	logStore := transport.InMemoryTransportLogStore()

// 	pk1, sk1 := cipher.GenerateKeyPair()
// 	pk2, sk2 := cipher.GenerateKeyPair()

// 	c1 := &transport.ManagerConfig{PubKey: pk1, SecKey: sk1, DiscoveryClient: client, LogStore: logStore}
// 	c2 := &transport.ManagerConfig{PubKey: pk2, SecKey: sk2, DiscoveryClient: client, LogStore: logStore}

// 	f1, f2 := transport.NewMockFactoryPair(pk1, pk2)
// 	m1, err := transport.NewManager(c1, f1)
// 	require.NoError(t, err)

// 	m2, err := transport.NewManager(c2, f2)
// 	require.NoError(t, err)

// 	go m2.Serve(context.TODO()) // nolint

// 	rt := routing.InMemoryRoutingTable()
// 	conf := &Config{
// 		Logger:           logging.MustGetLogger("routesetup"),
// 		PubKey:           pk1,
// 		SecKey:           sk1,
// 		TransportManager: m1,
// 		RoutingTable:     rt,
// 	}
// 	r := New(conf)
// 	errCh := make(chan error)
// 	go func() {
// 		errCh <- r.Serve(context.TODO())
// 	}()

// 	rw, rwIn := net.Pipe()
// 	go r.ServeApp(rwIn, 6) // nolint: errcheck
// 	proto := appnet.NewProtocol(rw)
// 	dataCh := make(chan []byte)
// 	go func() {
// 		// TODO(evanlinjin): Check of we need to get data of all frame types.
// 		_ = proto.Serve(appnet.HandlerMap{
// 			appnet.FrameData: func(_ *appnet.Protocol, b []byte) ([]byte, error) {
// 				go func() { dataCh <- b }()
// 				return nil, nil
// 			},
// 		})
// 	}()

// 	time.Sleep(100 * time.Millisecond)

// 	tr, err := m1.CreateTransport(context.TODO(), pk2, "mock", true)
// 	require.NoError(t, err)

// 	rule := routing.AppRule(time.Now().Add(time.Hour), 4, pk2, 5, 6)
// 	routeID, err := rt.AddRule(rule)
// 	require.NoError(t, err)

// 	ni1, ni2 := noiseInstances(t, pk1, pk2, sk1, sk2)
// 	raddr := &app.LoopAddr{PubKey: pk2, Port: 5}
// 	require.NoError(t, r.pm.SetLoop(6, raddr, &loop{tr.ID, 4, ni1}))

// 	tr2 := m2.Transport(tr.ID)
// 	df := app.DataFrame{
// 		Meta: app.LoopMeta{Local: app.LoopAddr{Port: 6}, Remote: *raddr},
// 		Data: []byte("bar"),
// 	}
// 	go proto.Call(appnet.FrameData, df.Encode()) // nolint: errcheck

// 	packet := make(routing.Packet, 25)
// 	_, err = tr2.Read(packet)
// 	require.NoError(t, err)
// 	assert.Equal(t, uint16(19), packet.Size())
// 	assert.Equal(t, routing.RouteID(4), packet.RouteID())
// 	decrypted, err := ni2.Decrypt(packet.Payload())
// 	require.NoError(t, err)
// 	assert.Equal(t, []byte("bar"), decrypted)

// 	_, err = tr2.Write(routing.MakePacket(routeID, ni2.Encrypt([]byte("foo"))))
// 	require.NoError(t, err)

// 	time.Sleep(100 * time.Millisecond)

// 	var aPacket app.DataFrame
// 	require.NoError(t, aPacket.Decode(<-dataCh))
// 	assert.Equal(t, pk2, aPacket.Meta.Remote.PubKey)
// 	assert.Equal(t, uint16(5), aPacket.Meta.Remote.Port)
// 	assert.Equal(t, uint16(6), aPacket.Meta.Local.Port)
// 	assert.Equal(t, []byte("foo"), aPacket.Data)

// 	require.NoError(t, r.Close())
// 	require.NoError(t, <-errCh)
// }

// func TestRouterLocalApp(t *testing.T) {
// 	client := transport.NewDiscoveryMock()
// 	logStore := transport.InMemoryTransportLogStore()

// 	pk, sk := cipher.GenerateKeyPair()
// 	m, err := transport.NewManager(&transport.ManagerConfig{PubKey: pk, SecKey: sk, DiscoveryClient: client, LogStore: logStore})
// 	require.NoError(t, err)

// 	conf := &Config{
// 		Logger:           logging.MustGetLogger("routesetup"),
// 		PubKey:           pk,
// 		SecKey:           sk,
// 		TransportManager: m,
// 		RoutingTable:     routing.InMemoryRoutingTable(),
// 	}
// 	r := New(conf)
// 	errCh := make(chan error)
// 	go func() {
// 		errCh <- r.Serve(context.TODO())
// 	}()

// 	rw1, rw1In := net.Pipe()
// 	go r.ServeApp(rw1In, 5) // nolint: errcheck
// 	proto1 := appnet.NewProtocol(rw1)
// 	go proto1.Serve(nil) // nolint: errcheck

// 	rw2, rw2In := net.Pipe()
// 	go r.ServeApp(rw2In, 6) // nolint: errcheck
// 	proto2 := appnet.NewProtocol(rw2)
// 	dataCh := make(chan []byte)
// 	go proto2.Serve(appnet.HandlerMap{
// 		appnet.FrameData: func(p *appnet.Protocol, b []byte) ([]byte, error) {
// 			go func() { dataCh <- b }()
// 			return nil, nil
// 		},
// 	})

// 	df := app.DataFrame{
// 		Meta: app.LoopMeta{Local: app.LoopAddr{Port: 5}, Remote: app.LoopAddr{PubKey: pk, Port: 6}},
// 		Data: []byte("foo"),
// 	}
// 	go proto1.Call(appnet.FrameData, df.Encode())

// 	time.Sleep(100 * time.Millisecond)

// 	var packet app.DataFrame
// 	require.NoError(t, packet.Decode(<-dataCh))
// 	require.NoError(t, err)
// 	assert.Equal(t, pk, packet.Meta.Remote.PubKey)
// 	assert.Equal(t, uint16(5), packet.Meta.Remote.Port)
// 	assert.Equal(t, uint16(6), packet.Meta.Local.Port)
// 	assert.Equal(t, []byte("foo"), packet.Data)

// 	require.NoError(t, r.Close())
// 	require.NoError(t, <-errCh)
// }

// TODO(evanlinjin): This test does not make sense. Fix it.
// func TestRouterSetup(t *testing.T) {
// 	client := transport.NewDiscoveryMock()
// 	logStore := transport.InMemoryTransportLogStore()

// 	pk1, sk1 := cipher.GenerateKeyPair()
// 	pk2, sk2 := cipher.GenerateKeyPair()

// 	c1 := &transport.ManagerConfig{PubKey: pk1, SecKey: sk1, DiscoveryClient: client, LogStore: logStore}
// 	c2 := &transport.ManagerConfig{PubKey: pk2, SecKey: sk2, DiscoveryClient: client, LogStore: logStore}

// 	f1, f2 := transport.NewMockFactory(pk1, pk2)
// 	m1, err := transport.NewManager(c1, f1)
// 	require.NoError(t, err)

// 	m2, err := transport.NewManager(c2, f2)
// 	require.NoError(t, err)

// 	rtm := routing.InMemoryRoutingTable()
// 	c := &Config{
// 		log:           logging.MustGetLogger("routesetup"),
// 		PubKey:           pk1,
// 		SecKey:           sk1,
// 		TransportManager: m1,
// 		RoutingTable:     rtm,
// 		SetupNodes:       []cipher.PubKey{pk2},
// 	}
// 	r := New(c)
// 	errCh := make(chan error)
// 	go func() {
// 		errCh <- r.Serve(context.TODO())
// 	}()

// 	tr, err := m2.CreateTransport(context.TODO(), pk1, "mock", false)
// 	require.NoError(t, err)
// 	sProto := setup.NewProtocol(tr)

// 	rw1, rwIn1 := net.Pipe()
// 	go r.ServeApp(rwIn1, 2) // nolint: errcheck
// 	proto1 := appnet.NewProtocol(rw1)
// 	dataCh := make(chan []byte)
// 	go proto1.Serve(appnet.HandlerMap{
// 		appnet.FrameData: func(p *appnet.Protocol, b []byte) ([]byte, error) {
// 			go func() { dataCh <- b }()
// 			return nil, nil
// 		},
// 	})

// 	rw2, rwIn2 := net.Pipe()
// 	go r.ServeApp(rwIn2, 4) // nolint: errcheck
// 	proto2 := appnet.NewProtocol(rw2)
// 	go proto2.Serve(appnet.HandlerMap{
// 		appnet.FrameData: func(p *appnet.Protocol, b []byte) ([]byte, error) {
// 			go func() { dataCh <- b }()
// 			return nil, nil
// 		},
// 	})

// 	var routeID routing.RouteID
// 	t.Run("add route", func(t *testing.T) {
// 		routeID, err = setup.AddRule(sProto, routing.ForwardRule(time.Now().Add(time.Hour), 2, tr.ID))
// 		require.NoError(t, err)

// 		rule, err := rtm.Rule(routeID)
// 		require.NoError(t, err)
// 		assert.Equal(t, routing.RouteID(2), rule.RouteID())
// 		assert.Equal(t, tr.ID, rule.TransportID())
// 	})

// 	t.Run("confirm loop - responder", func(t *testing.T) {
// 		confI := noise.Config{
// 			LocalSK:   sk2,
// 			LocalPK:   pk2,
// 			RemotePK:  pk1,
// 			Initiator: true,
// 		}

// 		ni, err := noise.KKAndSecp256k1(confI)
// 		require.NoError(t, err)
// 		msg, err := ni.HandshakeMessage()
// 		require.NoError(t, err)

// 		time.Sleep(100 * time.Millisecond)

// 		appRouteID, err := setup.AddRule(sProto, routing.AppRule(time.Now().Add(time.Hour), 0, pk2, 1, 2))
// 		require.NoError(t, err)

// 		noiseRes, err := setup.ConfirmLoop(sProto, &setup.LoopData{RemotePK: pk2, RemotePort: 1, LocalPort: 2, RouteID: routeID, NoiseMessage: msg})
// 		require.NoError(t, err)

// 		rule, err := rtm.Rule(appRouteID)
// 		require.NoError(t, err)
// 		assert.Equal(t, routeID, rule.RouteID())
// 		_, err = r.pm.Get(2)
// 		require.NoError(t, err)
// 		loop, err := r.pm.GetLoop(2, &app.LoopAddr{PubKey: pk2, Port: 1})
// 		require.NoError(t, err)
// 		require.NotNil(t, loop)
// 		assert.Equal(t, tr.ID, loop.trID)
// 		assert.Equal(t, routing.RouteID(2), loop.routeID)

// 		addrs := [2]*app.LoopAddr{}
// 		require.NoError(t, json.Unmarshal(<-dataCh, &addrs))
// 		require.NoError(t, err)
// 		assert.Equal(t, pk1, addrs[0].PubKey)
// 		assert.Equal(t, uint16(2), addrs[0].Port)
// 		assert.Equal(t, pk2, addrs[1].PubKey)
// 		assert.Equal(t, uint16(1), addrs[1].Port)

// 		require.NoError(t, ni.ProcessMessage(noiseRes))
// 	})

// 	t.Run("confirm loop - initiator", func(t *testing.T) {
// 		confI := noise.Config{
// 			LocalSK:   sk1,
// 			LocalPK:   pk1,
// 			RemotePK:  pk2,
// 			Initiator: true,
// 		}

// 		ni, err := noise.KKAndSecp256k1(confI)
// 		require.NoError(t, err)
// 		msg, err := ni.HandshakeMessage()
// 		require.NoError(t, err)

// 		confR := noise.Config{
// 			LocalSK:   sk2,
// 			LocalPK:   pk2,
// 			RemotePK:  pk1,
// 			Initiator: false,
// 		}

// 		nr, err := noise.KKAndSecp256k1(confR)
// 		require.NoError(t, err)
// 		require.NoError(t, nr.ProcessMessage(msg))
// 		noiseRes, err := nr.HandshakeMessage()
// 		require.NoError(t, err)

// 		time.Sleep(100 * time.Millisecond)

// 		require.NoError(t, r.pm.SetLoop(4, &app.LoopAddr{PubKey: pk2, Port: 3}, &loop{noise: ni}))

// 		appRouteID, err := setup.AddRule(sProto, routing.AppRule(time.Now().Add(time.Hour), 0, pk2, 3, 4))
// 		require.NoError(t, err)

// 		_, err = setup.ConfirmLoop(sProto, &setup.LoopData{RemotePK: pk2, RemotePort: 3, LocalPort: 4, RouteID: routeID, NoiseMessage: noiseRes})
// 		require.NoError(t, err)

// 		rule, err := rtm.Rule(appRouteID)
// 		require.NoError(t, err)
// 		assert.Equal(t, routeID, rule.RouteID())
// 		l, err := r.pm.GetLoop(2, &app.LoopAddr{PubKey: pk2, Port: 1})
// 		require.NoError(t, err)
// 		require.NotNil(t, l)
// 		assert.Equal(t, tr.ID, l.trID)
// 		assert.Equal(t, routing.RouteID(2), l.routeID)

// 		addrs := [2]*app.LoopAddr{}
// 		require.NoError(t, json.Unmarshal(<-dataCh, &addrs))
// 		require.NoError(t, err)
// 		assert.Equal(t, pk1, addrs[0].PubKey)
// 		assert.Equal(t, uint16(4), addrs[0].Port)
// 		assert.Equal(t, pk2, addrs[1].PubKey)
// 		assert.Equal(t, uint16(3), addrs[1].Port)
// 	})

// 	t.Run("loop closed", func(t *testing.T) {
// 		rule, err := rtm.Rule(3)
// 		require.NoError(t, err)
// 		require.NotNil(t, rule)
// 		assert.Equal(t, routing.RuleApp, rule.Type())

// 		require.NoError(t, setup.LoopClosed(sProto, &setup.LoopData{RemotePK: pk2, RemotePort: 3, LocalPort: 4}))
// 		time.Sleep(100 * time.Millisecond)

// 		_, err = r.pm.GetLoop(4, &app.LoopAddr{PubKey: pk2, Port: 3})
// 		require.Error(t, err)
// 		_, err = r.pm.Get(4)
// 		require.NoError(t, err)

// 		rule, err = rtm.Rule(3)
// 		require.NoError(t, err)
// 		require.Nil(t, rule)
// 	})

// 	t.Run("delete rule", func(t *testing.T) {
// 		require.NoError(t, setup.DeleteRule(sProto, routeID))

// 		rule, err := rtm.Rule(routeID)
// 		require.NoError(t, err)
// 		assert.Nil(t, rule)
// 	})
// }

// func TestRouterSetupLoop(t *testing.T) {
// 	client := transport.NewDiscoveryMock()
// 	logStore := transport.InMemoryTransportLogStore()

// 	pk1, sk1 := cipher.GenerateKeyPair()
// 	pk2, sk2 := cipher.GenerateKeyPair()

// 	f1, f2 := transport.NewMockFactoryPair(pk1, pk2)
// 	f1.SetType("messaging")
// 	f2.SetType("messaging")

// 	m1, err := transport.NewManager(&transport.ManagerConfig{PubKey: pk1, SecKey: sk1, DiscoveryClient: client, LogStore: logStore}, f1)
// 	require.NoError(t, err)

// 	m2, err := transport.NewManager(&transport.ManagerConfig{PubKey: pk2, SecKey: sk2, DiscoveryClient: client, LogStore: logStore}, f2)
// 	require.NoError(t, err)
// 	go m2.Serve(context.TODO()) // nolint: errcheck

// 	conf := &Config{
// 		Logger:           logging.MustGetLogger("routesetup"),
// 		PubKey:           pk1,
// 		SecKey:           sk1,
// 		TransportManager: m1,
// 		RoutingTable:     routing.InMemoryRoutingTable(),
// 		RouteFinder:      routeFinder.NewMock(),
// 		SetupNodes:       []cipher.PubKey{pk2},
// 	}
// 	r := New(conf)
// 	errCh := make(chan error)
// 	go func() {
// 		acceptCh, _ := m2.Observe()
// 		tr := <-acceptCh

// 		proto := setup.NewSetupProtocol(tr)
// 		p, data, err := proto.ReadPacket()
// 		if err != nil {
// 			errCh <- err
// 			return
// 		}

// 		if p != setup.PacketCreateLoop {
// 			errCh <- errors.New("unknown command")
// 			return
// 		}

// 		l := &routing.Loop{}
// 		if err := json.Unmarshal(data, l); err != nil {
// 			errCh <- err
// 			return
// 		}

// 		if l.LocalPort != 10 || l.RemotePort != 6 {
// 			errCh <- errors.New("invalid payload")
// 			return
// 		}

// 		errCh <- proto.WritePacket(setup.RespSuccess, []byte{})
// 	}()

// 	rw, rwIn := net.Pipe()
// 	go r.ServeApp(rwIn, 5) // nolint: errcheck
// 	proto := appnet.NewProtocol(rw)
// 	go proto.Serve(nil) // nolint: errcheck

// 	addrRaw, err := proto.Call(appnet.FrameCreateLoop, (&app.LoopAddr{PubKey: pk2, Port: 6}).Encode())
// 	require.NoError(t, err)
// 	var lm app.LoopMeta
// 	require.NoError(t, lm.Decode(addrRaw))

// 	require.NoError(t, <-errCh)
// 	ll, err := r.pm.GetLoop(10, &app.LoopAddr{PubKey: pk2, Port: 6})
// 	require.NoError(t, err)
// 	require.NotNil(t, ll)
// 	require.NotNil(t, ll.noise)

// 	assert.Equal(t, pk1, lm.Local.PubKey)
// 	assert.Equal(t, uint16(10), lm.Local.Port)
// }

// func noiseInstances(t *testing.T, pkI, pkR cipher.PubKey, skI, skR cipher.SecKey) (ni, nr *noise.Noise) {
// 	t.Helper()

// 	var err error
// 	confI := noise.Config{
// 		LocalSK:   skI,
// 		LocalPK:   pkI,
// 		RemotePK:  pkR,
// 		Initiator: true,
// 	}

// 	confR := noise.Config{
// 		LocalSK:   skR,
// 		LocalPK:   pkR,
// 		RemotePK:  pkI,
// 		Initiator: false,
// 	}

// 	ni, err = noise.KKAndSecp256k1(confI)
// 	require.NoError(t, err)

// 	nr, err = noise.KKAndSecp256k1(confR)
// 	require.NoError(t, err)

// 	msg, err := ni.HandshakeMessage()
// 	require.NoError(t, err)
// 	require.NoError(t, nr.ProcessMessage(msg))

// 	res, err := nr.HandshakeMessage()
// 	require.NoError(t, err)
// 	require.NoError(t, ni.ProcessMessage(res))
// 	return ni, nr
// }
