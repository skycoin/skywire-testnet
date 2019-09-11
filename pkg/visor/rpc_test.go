package visor

import (
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/skycoin/src/util/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/util/pathutil"
)

func TestHealth(t *testing.T) {
	sPK, sSK := cipher.GenerateKeyPair()

	c := &Config{}
	c.Node.StaticPubKey = sPK
	c.Node.StaticSecKey = sSK
	c.Transport.Discovery = "foo"
	c.Routing.SetupNodes = []cipher.PubKey{sPK}
	c.Routing.RouteFinder = "foo"

	t.Run("Report all the services as available", func(t *testing.T) {
		rpc := &RPC{&Node{config: c}}
		h := &HealthInfo{}
		err := rpc.Health(nil, h)
		require.NoError(t, err)

		// Transport discovery needs to be mocked or will always fail
		assert.Equal(t, http.StatusOK, h.SetupNode)
		assert.Equal(t, http.StatusOK, h.RouteFinder)
	})

	t.Run("Report as unavailable", func(t *testing.T) {
		rpc := &RPC{&Node{config: &Config{}}}
		h := &HealthInfo{}
		err := rpc.Health(nil, h)
		require.NoError(t, err)

		assert.Equal(t, http.StatusNotFound, h.SetupNode)
		assert.Equal(t, http.StatusNotFound, h.RouteFinder)
	})
}

func TestUptime(t *testing.T) {
	rpc := &RPC{&Node{startedAt: time.Now()}}
	time.Sleep(time.Second)
	var res float64
	err := rpc.Uptime(nil, &res)
	require.NoError(t, err)

	assert.Contains(t, fmt.Sprintf("%f", res), "1.0")
}

func TestListApps(t *testing.T) {
	apps := []AppConfig{
		{App: "foo", AutoStart: false, Port: 10},
		{App: "bar", AutoStart: true, Port: 11},
	}

	sApps := map[string]*appBind{
		"bar": {},
	}
	rpc := &RPC{&Node{appsConf: apps, startedApps: sApps}}

	var reply []*AppState
	require.NoError(t, rpc.Apps(nil, &reply))
	require.Len(t, reply, 2)

	app1 := reply[0]
	assert.Equal(t, "foo", app1.Name)
	assert.False(t, app1.AutoStart)
	assert.Equal(t, routing.Port(10), app1.Port)
	assert.Equal(t, AppStatusStopped, app1.Status)

	app2 := reply[1]
	assert.Equal(t, "bar", app2.Name)
	assert.True(t, app2.AutoStart)
	assert.Equal(t, routing.Port(11), app2.Port)
	assert.Equal(t, AppStatusRunning, app2.Status)
}

func TestStartStopApp(t *testing.T) {
	pk, _ := cipher.GenerateKeyPair()
	router := new(mockRouter)
	executer := new(MockExecuter)
	defer func() {
		require.NoError(t, os.RemoveAll("skychat"))
	}()

	apps := []AppConfig{{App: "foo", Version: "1.0", AutoStart: false, Port: 10}}
	node := &Node{router: router, executer: executer, appsConf: apps, startedApps: map[string]*appBind{}, logger: logging.MustGetLogger("test"), config: &Config{}}
	node.config.Node.StaticPubKey = pk
	pathutil.EnsureDir(node.dir())
	defer func() {
		require.NoError(t, os.RemoveAll(node.dir()))
	}()

	rpc := &RPC{node: node}
	unknownApp := "bar"
	app := "foo"

	err := rpc.StartApp(&unknownApp, nil)
	require.Error(t, err)
	assert.Equal(t, ErrUnknownApp, err)

	require.NoError(t, rpc.StartApp(&app, nil))
	time.Sleep(100 * time.Millisecond)

	executer.Lock()
	require.Len(t, executer.cmds, 1)
	assert.Equal(t, "foo.v1.0", executer.cmds[0].Path)
	assert.Equal(t, "foo/v1.0", executer.cmds[0].Dir)
	executer.Unlock()
	node.startedMu.Lock()
	assert.NotNil(t, node.startedApps["foo"])
	node.startedMu.Unlock()

	err = rpc.StopApp(&unknownApp, nil)
	require.Error(t, err)
	assert.Equal(t, ErrUnknownApp, err)

	require.NoError(t, rpc.StopApp(&app, nil))
	time.Sleep(100 * time.Millisecond)

	node.startedMu.Lock()
	assert.Nil(t, node.startedApps["foo"])
	node.startedMu.Unlock()
}

type TestRPC struct{}

type AddIn struct{ A, B int }

func (r *TestRPC) Add(in *AddIn, out *int) error {
	*out = in.A + in.B
	return nil
}

// TODO: Implement correctly
/*
func TestRPCClientDialer(t *testing.T) {
	svr := rpc.NewServer()
	require.NoError(t, svr.Register(new(TestRPC)))

	lPK, lSK := cipher.GenerateKeyPair()
	var l *snet.Listener
	var port uint16
	var listenerN *snet.Network
	discMock := disc.NewMock()

	sPK, sSK := cipher.GenerateKeyPair()
	sl, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	log.Info("here: ", sl.Addr().String())
	server, err := dmsg.NewServer(sPK, sSK, fmt.Sprintf(":%d", sl.Addr().(*net.TCPAddr).Port), sl, discMock)
	require.NoError(t, err)
	go func() {
		log.Fatal("server error is !!!!!!!!!1 ---->>> ", server.Serve())
	}()

	setup := func() {
		var err error

		listenerN = snet.NewRaw(snet.Config{
			PubKey:      lPK,
			SecKey:      lSK,
			TpNetworks:  []string{snet.DmsgType},
			DmsgMinSrvs: 1,
		}, dmsg.NewClient(lPK, lSK, discMock), nil)

		require.NoError(t, listenerN.Init(context.Background()))
		l, err = listenerN.Listen(snet.DmsgType, 9999)
		require.NoError(t, err)

		lAddr := l.Addr().(dmsg.Addr).String()
		t.Logf("Listening on %s", lAddr)

		port = l.Addr().(dmsg.Addr).Port
	}

	teardown := func() {
		require.NoError(t, l.Close())
		require.NoError(t, listenerN.Close())
		l = nil
	}

	t.Run("RunRetry", func(t *testing.T) {
		setup()
		defer teardown() // Just in case of failure.

		const reconCount = 5
		const retry = time.Second / 4

		dPK, dSK := cipher.GenerateKeyPair()
		n := snet.NewRaw(snet.Config{
			PubKey:      dPK,
			SecKey:      dSK,
			TpNetworks:  []string{snet.DmsgType},
			DmsgMinSrvs: 1,
		}, dmsg.NewClient(dPK, dSK, discMock), nil)
		require.NoError(t, n.Init(context.Background()))

		d := NewRPCClientDialer(n, lPK, port)
		dDone := make(chan error, 1)

		go func() {
			err := d.Run(svr, retry)
			dDone <- err
			close(dDone)
		}()

		for i := 0; i < reconCount; i++ {
			teardown()
			time.Sleep(retry * 2) // Dialer shouldn't quit retrying in this time.
			setup()

			conn, err := l.Accept()
			require.NoError(t, err)

			in, out := &AddIn{A: i, B: i}, new(int)
			require.NoError(t, rpc.NewClient(conn).Call("TestRPC.Add", in, out))
			require.Equal(t, in.A+in.B, *out)
			require.NoError(t, conn.Close()) // NOTE: also closes d, as it's the same connection
		}

		// The same connection is closed above (conn.Close()), and hence, this may return an error.
		_ = d.Close() // nolint: errcheck
		require.NoError(t, <-dDone)
	})
}
/*

/*
TODO(evanlinjin): Fix these tests.
These tests have been commented out for the following reasons:
- We can't seem to get them to work.
- Mock transport causes too much issues so we deleted it.
*/

//func TestRPC(t *testing.T) {
//	r := new(mockRouter)
//	executer := new(MockExecuter)
//	defer func() {
//		require.NoError(t, os.RemoveAll("skychat"))
//	}()
//
//	pk1, _, tm1, tm2, errCh, err := transport.MockTransportManagersPair()
//
//	require.NoError(t, err)
//	defer func() {
//		require.NoError(t, tm1.Close())
//		require.NoError(t, tm2.Close())
//		require.NoError(t, <-errCh)
//		require.NoError(t, <-errCh)
//	}()
//
//	_, err = tm2.SaveTransport(context.TODO(), pk1, snet.DmsgType)
//	require.NoError(t, err)
//
//	apps := []AppConfig{
//		{App: "foo", Version: "1.0", AutoStart: false, Port: 10},
//		{App: "bar", Version: "2.0", AutoStart: false, Port: 20},
//	}
//	conf := &Config{}
//	conf.Node.StaticPubKey = pk1
//	node := &Node{
//		config:      conf,
//		router:      r,
//		tm:          tm1,
//		rt:          routing.InMemoryRoutingTable(),
//		executer:    executer,
//		appsConf:    apps,
//		startedApps: map[string]*appBind{},
//		logger:      logging.MustGetLogger("test"),
//	}
//	pathutil.EnsureDir(node.dir())
//	defer func() {
//		if err := os.RemoveAll(node.dir()); err != nil {
//			log.WithError(err).Warn(err)
//		}
//	}()
//
//	require.NoError(t, node.StartApp("foo"))
//
//	time.Sleep(time.Second)
//	gateway := &RPC{node: node}
//
//	sConn, cConn := net.Pipe()
//	defer func() {
//		require.NoError(t, sConn.Close())
//		require.NoError(t, cConn.Close())
//	}()
//
//	svr := rpc.NewServer()
//	require.NoError(t, svr.RegisterName(RPCPrefix, gateway))
//	go svr.ServeConn(sConn)
//
//	// client := RPCClient{Client: rpc.NewClient(cConn)}
//	client := NewRPCClient(rpc.NewClient(cConn), "")
//
//	printFunc := func(t *testing.T, name string, v interface{}) {
//		j, err := json.MarshalIndent(v, name+": ", "  ")
//		require.NoError(t, err)
//		t.Log(string(j))
//	}
//
//	t.Run("Summary", func(t *testing.T) {
//		test := func(t *testing.T, summary *Summary) {
//			assert.Equal(t, pk1, summary.PubKey)
//			assert.Len(t, summary.Apps, 2)
//			assert.Len(t, summary.Transports, 1)
//			printFunc(t, "Summary", summary)
//		}
//		t.Run("RPCServer", func(t *testing.T) {
//			var summary Summary
//			require.NoError(t, gateway.Summary(&struct{}{}, &summary))
//			test(t, &summary)
//		})
//		t.Run("RPCClient", func(t *testing.T) {
//			summary, err := client.Summary()
//			require.NoError(t, err)
//			test(t, summary)
//		})
//	})
//
//	t.Run("Exec", func(t *testing.T) {
//		command := "echo 1"
//
//		t.Run("RPCServer", func(t *testing.T) {
//			var out []byte
//			require.NoError(t, gateway.Exec(&command, &out))
//			assert.Equal(t, []byte("1\n"), out)
//		})
//
//		t.Run("RPCClient", func(t *testing.T) {
//			out, err := client.Exec(command)
//			require.NoError(t, err)
//			assert.Equal(t, []byte("1\n"), out)
//		})
//	})
//
//	t.Run("Apps", func(t *testing.T) {
//		test := func(t *testing.T, apps []*AppState) {
//			assert.Len(t, apps, 2)
//			printFunc(t, "Apps", apps)
//		}
//		t.Run("RPCServer", func(t *testing.T) {
//			var apps []*AppState
//			require.NoError(t, gateway.Apps(&struct{}{}, &apps))
//			test(t, apps)
//		})
//		t.Run("RPCClient", func(t *testing.T) {
//			apps, err := client.Apps()
//			require.NoError(t, err)
//			test(t, apps)
//		})
//	})
//
//	// TODO(evanlinjin): For some reason, this freezes.
//	t.Run("StopStartApp", func(t *testing.T) {
//		appName := "foo"
//		require.NoError(t, gateway.StopApp(&appName, &struct{}{}))
//		require.NoError(t, gateway.StartApp(&appName, &struct{}{}))
//		require.NoError(t, client.StopApp(appName))
//		require.NoError(t, client.StartApp(appName))
//	})
//
//	t.Run("SetAutoStart", func(t *testing.T) {
//		unknownAppName := "whoAmI"
//		appName := "foo"
//
//		in1 := SetAutoStartIn{AppName: unknownAppName, AutoStart: true}
//		in2 := SetAutoStartIn{AppName: appName, AutoStart: true}
//		in3 := SetAutoStartIn{AppName: appName, AutoStart: false}
//
//		// Test with RPC Server
//
//		err := gateway.SetAutoStart(&in1, &struct{}{})
//		require.Error(t, err)
//		assert.Equal(t, ErrUnknownApp, err)
//
//		require.NoError(t, gateway.SetAutoStart(&in2, &struct{}{}))
//		assert.True(t, node.appsConf[0].AutoStart)
//
//		require.NoError(t, gateway.SetAutoStart(&in3, &struct{}{}))
//		assert.False(t, node.appsConf[0].AutoStart)
//
//		// Test with RPC Client
//
//		err = client.SetAutoStart(in1.AppName, in1.AutoStart)
//		require.Error(t, err)
//		assert.Equal(t, ErrUnknownApp.Error(), err.Error())
//
//		require.NoError(t, client.SetAutoStart(in2.AppName, in2.AutoStart))
//		assert.True(t, node.appsConf[0].AutoStart)
//
//		require.NoError(t, client.SetAutoStart(in3.AppName, in3.AutoStart))
//		assert.False(t, node.appsConf[0].AutoStart)
//	})
//
//	t.Run("TransportTypes", func(t *testing.T) {
//		in := TransportsIn{ShowLogs: true}
//
//		var out []*TransportSummary
//		require.NoError(t, gateway.Transports(&in, &out))
//		require.Len(t, out, 1)
//		assert.Equal(t, "mock", out[0].Type)
//
//		out2, err := client.Transports(in.FilterTypes, in.FilterPubKeys, in.ShowLogs)
//		require.NoError(t, err)
//		assert.Equal(t, out, out2)
//	})
//
//	t.Run("Transport", func(t *testing.T) {
//		var ids []uuid.UUID
//		node.tm.WalkTransports(func(tp *transport.ManagedTransport) bool {
//			ids = append(ids, tp.Entry.ID)
//			return true
//		})
//
//		for _, id := range ids {
//			id := id
//			var summary TransportSummary
//			require.NoError(t, gateway.Transport(&id, &summary))
//
//			summary2, err := client.Transport(id)
//			require.NoError(t, err)
//			require.Equal(t, summary, *summary2)
//		}
//	})
//
//	// TODO: Test add/remove transports
//
//}
