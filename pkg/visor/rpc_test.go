package visor

import (
	"context"
	"encoding/json"
	"net"
	"net/rpc"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/skycoin/src/util/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/transport"
	"github.com/skycoin/skywire/pkg/util/pathutil"
)

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
	assert.Equal(t, uint16(10), app1.Port)
	assert.Equal(t, AppStatusStopped, app1.Status)

	app2 := reply[1]
	assert.Equal(t, "bar", app2.Name)
	assert.True(t, app2.AutoStart)
	assert.Equal(t, uint16(11), app2.Port)
	assert.Equal(t, AppStatusRunning, app2.Status)
}

func TestStartStopApp(t *testing.T) {
	pk, _ := cipher.GenerateKeyPair()
	router := new(mockRouter)
	executer := new(MockExecuter)
	defer os.RemoveAll("skychat")

	apps := []AppConfig{{App: "foo", Version: "1.0", AutoStart: false, Port: 10}}
	node := &Node{router: router, executer: executer, appsConf: apps, startedApps: map[string]*appBind{}, logger: logging.MustGetLogger("test"), config: &Config{}}
	node.config.Node.StaticPubKey = pk
	pathutil.EnsureDir(node.dir())
	defer os.RemoveAll(node.dir())

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

func TestRPC(t *testing.T) {
	r := new(mockRouter)
	executer := new(MockExecuter)
	defer os.RemoveAll("skychat")

	pk1, _, tm1, tm2, errCh, err := transport.MockTransportManagersPair()
	require.NoError(t, err)
	defer func() {
		require.NoError(t, tm1.Close())
		require.NoError(t, tm2.Close())
		require.NoError(t, <-errCh)
		require.NoError(t, <-errCh)
	}()

	_, err = tm2.CreateTransport(context.TODO(), pk1, "mock", true)
	require.NoError(t, err)

	apps := []AppConfig{
		{App: "foo", Version: "1.0", AutoStart: false, Port: 10},
		{App: "bar", Version: "2.0", AutoStart: false, Port: 20},
	}
	conf := &Config{}
	conf.Node.StaticPubKey = pk1
	node := &Node{
		config:      conf,
		router:      r,
		tm:          tm1,
		rt:          routing.InMemoryRoutingTable(),
		executer:    executer,
		appsConf:    apps,
		startedApps: map[string]*appBind{},
		logger:      logging.MustGetLogger("test"),
	}
	pathutil.EnsureDir(node.dir())
	defer os.RemoveAll(node.dir())

	require.NoError(t, node.StartApp("foo"))
	require.NoError(t, node.StartApp("bar"))

	time.Sleep(time.Second)
	gateway := &RPC{node: node}

	sConn, cConn := net.Pipe()
	defer func() {
		require.NoError(t, sConn.Close())
		require.NoError(t, cConn.Close())
	}()

	svr := rpc.NewServer()
	require.NoError(t, svr.RegisterName(RPCPrefix, gateway))
	go svr.ServeConn(sConn)

	//client := RPCClient{Client: rpc.NewClient(cConn)}

	print := func(t *testing.T, name string, v interface{}) {
		j, err := json.MarshalIndent(v, name+": ", "  ")
		require.NoError(t, err)
		t.Log(string(j))
	}

	t.Run("Summary", func(t *testing.T) {
		test := func(t *testing.T, summary *Summary) {
			assert.Equal(t, pk1, summary.PubKey)
			assert.Len(t, summary.Apps, 2)
			assert.Len(t, summary.Transports, 1)
			print(t, "Summary", summary)
		}
		t.Run("RPCServer", func(t *testing.T) {
			var summary Summary
			require.NoError(t, gateway.Summary(&struct{}{}, &summary))
			test(t, &summary)
		})
		//t.Run("RPCClient", func(t *testing.T) {
		//	summary, err := client.Summary()
		//	require.NoError(t, err)
		//	test(t, summary)
		//})
	})

	t.Run("Apps", func(t *testing.T) {
		test := func(t *testing.T, apps []*AppState) {
			assert.Len(t, apps, 2)
			print(t, "Apps", apps)
		}
		t.Run("RPCServer", func(t *testing.T) {
			var apps []*AppState
			require.NoError(t, gateway.Apps(&struct{}{}, &apps))
			test(t, apps)
		})
		//t.Run("RPCClient", func(t *testing.T) {
		//	apps, err := client.Apps()
		//	require.NoError(t, err)
		//	test(t, apps)
		//})
	})

	// TODO(evanlinjin): For some reason, this freezes.
	//t.Run("StopStartApp", func(t *testing.T) {
	//	appName := "foo"
	//	require.NoError(t, gateway.StopApp(&appName, &struct{}{}))
	//	require.NoError(t, gateway.StartApp(&appName, &struct{}{}))
	//	require.NoError(t, client.StopApp(appName))
	//	require.NoError(t, client.StartApp(appName))
	//})

	t.Run("SetAutoStart", func(t *testing.T) {
		unknownAppName := "whoAmI"
		appName := "foo"

		in1 := SetAutoStartIn{AppName: unknownAppName, AutoStart: true}
		in2 := SetAutoStartIn{AppName: appName, AutoStart: true}
		in3 := SetAutoStartIn{AppName: appName, AutoStart: false}

		// Test with RPC Server

		err := gateway.SetAutoStart(&in1, &struct{}{})
		require.Error(t, err)
		assert.Equal(t, ErrUnknownApp, err)

		require.NoError(t, gateway.SetAutoStart(&in2, &struct{}{}))
		assert.True(t, node.appsConf[0].AutoStart)

		require.NoError(t, gateway.SetAutoStart(&in3, &struct{}{}))
		assert.False(t, node.appsConf[0].AutoStart)

		// Test with RPC Client

		//err = client.SetAutoStart(in1.AppName, in1.AutoStart)
		//require.Error(t, err)
		//assert.Equal(t, ErrUnknownApp.Error(), err.Error())
		//
		//require.NoError(t, client.SetAutoStart(in2.AppName, in2.AutoStart))
		//assert.True(t, node.appsConf[0].AutoStart)
		//
		//require.NoError(t, client.SetAutoStart(in3.AppName, in3.AutoStart))
		//assert.False(t, node.appsConf[0].AutoStart)
	})

	t.Run("TransportTypes", func(t *testing.T) {
		in := TransportsIn{ShowLogs: true}

		var out []*TransportSummary
		require.NoError(t, gateway.Transports(&in, &out))
		assert.Len(t, out, 1)
		assert.Equal(t, "mock", out[0].Type)

		//out2, err := client.Transports(in.FilterTypes, in.FilterPubKeys, in.ShowLogs)
		//require.NoError(t, err)
		//assert.Equal(t, out, out2)
	})

	t.Run("Transport", func(t *testing.T) {
		var ids []uuid.UUID
		node.tm.WalkTransports(func(tp *transport.ManagedTransport) bool {
			ids = append(ids, tp.Entry.ID)
			return true
		})

		for _, id := range ids {
			var summary TransportSummary
			require.NoError(t, gateway.Transport(&id, &summary))

			//summary2, err := client.Transport(id)
			//require.NoError(t, err)
			//require.Equal(t, summary, *summary2)
		}
	})

	// TODO: Test add/remove transports
}
