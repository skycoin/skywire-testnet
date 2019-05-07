package node

import (
	"os"
	"testing"

	"github.com/skycoin/skycoin/src/util/logging"
)

func TestMain(m *testing.M) {
	lvl, _ := logging.LevelFromString("error") // nolint: errcheck
	logging.SetLevel(lvl)
	os.Exit(m.Run())
}

//func TestNewNode(t *testing.T) {
//	pk, sk := cipher.GenerateKeyPair()
//	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		json.NewEncoder(w).Encode(&httpauth.NextNonceResponse{Edge: pk, NextNonce: 1}) // nolint: errcheck
//	}))
//	defer srv.Close()
//
//	conf := Config{Version: "1.0", LocalPath: "local", AppsPath: "apps"}
//	conf.Node.PubKey = pk
//	conf.Node.SecKey = sk
//	conf.Messaging.Discovery = "http://skywire.skycoin.net:8001"
//	conf.Messaging.ServerCount = 10
//	conf.Transport.Discovery = srv.URL
//	conf.Apps = []AppConfig{
//		{App: "foo", Port: 1},
//		{App: "bar", AutoStart: true, Port: 2},
//	}
//
//	defer os.RemoveAll("local")
//
//	node, err := NewNode(&conf)
//	require.NoError(t, err)
//
//	assert.NotNil(t, node.r)
//	assert.NotNil(t, node.appsPath)
//	assert.NotNil(t, node.localPath)
//	assert.NotNil(t, node.startedApps)
//}

// TODO(evanlinjin): fix.
//func TestNodeStartClose(t *testing.T) {
//	r := new(mockRouter)
//	executer := &MockExecuter{}
//	conf := []AppConfig{
//		{App: "chat", Version: "1.0", AutoStart: true, Port: 1},
//		{App: "foo", Version: "1.0", AutoStart: false},
//	}
//	defer os.RemoveAll("chat")
//	node := &Node{config: &Config{}, router: r, executer: executer, appsConf: conf,
//		startedApps: map[string]*appBind{}, logger: logging.MustGetLogger("test")}
//	mConf := &messaging.Config{PubKey: cipher.PubKey{}, SecKey: cipher.SecKey{}, Discovery: client.NewMock()}
//	node.messenger = messaging.NewClient(mConf)
//	var err error
//
//	tmConf := &transport.ManagerConfig{PubKey: cipher.PubKey{}, DiscoveryClient: transport.NewDiscoveryMock()}
//	node.tm, err = transport.NewManager(tmConf, node.messenger)
//	require.NoError(t, err)
//
//	errCh := make(chan error)
//	go func() {
//		errCh <- node.Start()
//	}()
//
//	time.Sleep(100 * time.Millisecond)
//	require.NoError(t, node.Close())
//	require.True(t, r.didClose)
//	require.NoError(t, <-errCh)
//
//	require.Len(t, executer.cmds, 1)
//	assert.Equal(t, "chat.v1.0", executer.cmds[0].Path)
//	assert.Equal(t, "chat/v1.0", executer.cmds[0].Dir)
//}

// TODO(evanlinjin): fix.
//func TestNodeSpawnApp(t *testing.T) {
//	pk, sk := cipher.GenerateKeyPair()
//	r := new(mockRouter)
//	executer := &MockExecuter{}
//	defer os.RemoveAll("chat")
//	apps := []AppConfig{{App: "chat", Version: "1.0", AutoStart: false, Port: 10, Args: []string{"foo"}}}
//	node := &Node{
//		config: &Config{Node: KeyFields{PubKey: pk, SecKey: sk}},
//		router: r,
//		executer: executer,
//		appsConf: apps,
//		startedApps: map[string]*appBind{},
//		logger: logging.MustGetLogger("test"),
//	}
//
//	require.NoError(t, node.StartApp("chat"))
//	time.Sleep(100 * time.Millisecond)
//
//	require.NotNil(t, node.startedApps["chat"])
//
//	executer.Lock()
//	require.Len(t, executer.cmds, 1)
//	assert.Equal(t, "chat.v1.0", executer.cmds[0].Path)
//	assert.Equal(t, "chat/v1.0", executer.cmds[0].Dir)
//	assert.Equal(t, []string{"chat.v1.0", "foo"}, executer.cmds[0].Args)
//	executer.Unlock()
//
//	ports := r.Ports()
//	require.Len(t, ports, 1)
//	assert.Equal(t, uint16(10), ports[0])
//
//	require.NoError(t, node.StopApp("chat"))
//}

// TODO(evanlinjin): fix.
//func TestNodeSpawnAppValidations(t *testing.T) {
//	pk, sk := cipher.GenerateKeyPair()
//	conn, _ := net.Pipe()
//	r := new(mockRouter)
//	executer := &MockExecuter{err: errors.New("foo")}
//	defer os.RemoveAll("chat")
//	node := &Node{router: r, executer: executer,
//		config: &Config{Node: KeyFields{PubKey: pk, SecKey: sk}},
//		startedApps: map[string]*appBind{"chat": {conn, 10}},
//		logger:      logging.MustGetLogger("test")}
//
//	cases := []struct {
//		conf *AppConfig
//		err  string
//	}{
//		{&AppConfig{App: "chat", Version: "1.0", Port: 2}, "can't bind to reserved Port 2"},
//		{&AppConfig{App: "chat", Version: "1.0", Port: 10}, "app chat is already started"},
//		{&AppConfig{App: "foo", Version: "1.0", Port: 11}, "failed to run app executable: foo"},
//	}
//
//	for _, c := range cases {
//		t.Run(c.err, func(t *testing.T) {
//			errCh := make(chan error)
//			go func() {
//				errCh <- node.SpawnApp(c.conf, nil)
//			}()
//
//			time.Sleep(100 * time.Millisecond)
//			require.NoError(t, node.Close())
//			err := <-errCh
//			require.Error(t, err)
//			assert.Equal(t, c.err, err.Error())
//		})
//	}
//}

//type mockRouter struct {
//	sync.Mutex
//
//	ports []uint16
//
//	didStart bool
//	didClose bool
//
//	errChan chan error
//}
//
//func (r *mockRouter) Ports() []uint16 {
//	r.Lock()
//	p := r.ports
//	r.Unlock()
//	return p
//}
//
//func (r *mockRouter) Serve(_ context.Context) error {
//	r.didStart = true
//	return nil
//}
//
//func (r *mockRouter) ServeApp(conn net.Conn, Port uint16) error {
//	r.Lock()
//	if r.ports == nil {
//		r.ports = []uint16{}
//	}
//
//	r.ports = append(r.ports, Port)
//	r.Unlock()
//
//	if r.errChan == nil {
//		r.Lock()
//		r.errChan = make(chan error)
//		r.Unlock()
//	}
//
//	return <-r.errChan
//}
//
//func (r *mockRouter) Close() error {
//	r.didClose = true
//	r.Lock()
//	if r.errChan != nil {
//		close(r.errChan)
//	}
//	r.Unlock()
//	return nil
//}
