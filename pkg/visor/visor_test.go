package visor

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"

	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/skycoin/src/util/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/internal/httpauth"
	"github.com/skycoin/skywire/pkg/app"
	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/transport"
	"github.com/skycoin/skywire/pkg/util/pathutil"
)

var masterLogger *logging.MasterLogger

func TestMain(m *testing.M) {
	masterLogger = logging.NewMasterLogger()
	loggingLevel, ok := os.LookupEnv("TEST_LOGGING_LEVEL")
	if ok {
		lvl, err := logging.LevelFromString(loggingLevel)
		if err != nil {
			log.Fatal(err)
		}
		masterLogger.SetLevel(lvl)
	} else {
		masterLogger.Out = ioutil.Discard
	}

	os.Exit(m.Run())
}

func TestNewNode(t *testing.T) {
	pk, sk := cipher.GenerateKeyPair()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewEncoder(w).Encode(&httpauth.NextNonceResponse{Edge: pk, NextNonce: 1}))
	}))
	defer srv.Close()

	conf := Config{Version: "1.0", LocalPath: "local", AppsPath: "apps"}
	conf.Node.StaticPubKey = pk
	conf.Node.StaticSecKey = sk
	conf.Messaging.Discovery = "http://skywire.skycoin.net:8001"
	conf.Messaging.ServerCount = 10
	conf.Transport.Discovery = srv.URL
	conf.Apps = []AppConfig{
		{App: "foo", Version: "1.1", Port: 1},
		{App: "bar", AutoStart: true, Port: 2},
	}

	defer func() {
		require.NoError(t, os.RemoveAll("local"))
	}()

	node, err := NewNode(&conf, masterLogger)
	require.NoError(t, err)

	assert.NotNil(t, node.router)
	assert.NotNil(t, node.appsConf)
	assert.NotNil(t, node.appsPath)
	assert.NotNil(t, node.localPath)
	assert.NotNil(t, node.startedApps)
}

// TODO(Darkren): fix test
/*func TestNodeStartClose(t *testing.T) {
	r := new(mockRouter)
	executer := &MockExecuter{}
	conf := []AppConfig{
		{App: "skychat", Version: "1.0", AutoStart: true, Port: 1},
		{App: "foo", Version: "1.0", AutoStart: false},
	}

	defer func() {
		require.NoError(t, os.RemoveAll("skychat"))
	}()

	node := &Node{config: &Config{}, router: r, executer: executer, appsConf: conf,
		startedApps: map[string]*appBind{}, logger: logging.MustGetLogger("test")}
	mConf := &dmsg.Config{PubKey: cipher.PubKey{}, SecKey: cipher.SecKey{}, Discovery: disc.NewMock()}
	node.messenger = dmsg.NewClient(mConf.PubKey, mConf.SecKey, mConf.Discovery)

	var err error

	tmConf := &transport.ManagerConfig{PubKey: cipher.PubKey{}, DiscoveryClient: transport.NewDiscoveryMock()}
	node.tm, err = transport.NewManager(tmConf, nil, node.messenger)
	require.NoError(t, err)

	errCh := make(chan error)
	go func() {
		errCh <- node.Start()
	}()

	time.Sleep(100 * time.Millisecond)
	require.NoError(t, node.Close())
	require.True(t, r.didClose)
	require.NoError(t, <-errCh)

	require.Len(t, executer.cmds, 1)
	assert.Equal(t, "skychat.v1.0", executer.cmds[0].Path)
	assert.Equal(t, "skychat/v1.0", executer.cmds[0].Dir)
}*/

func TestNodeSpawnApp(t *testing.T) {
	pk, _ := cipher.GenerateKeyPair()
	r := new(mockRouter)
	executer := &MockExecuter{}
	defer func() {
		require.NoError(t, os.RemoveAll("skychat"))
	}()
	apps := []AppConfig{{App: "skychat", Version: "1.0", AutoStart: false, Port: 10, Args: []string{"foo"}}}
	node := &Node{router: r, executer: executer, appsConf: apps, startedApps: map[string]*appBind{}, logger: logging.MustGetLogger("test"),
		config: &Config{}}
	node.config.Node.StaticPubKey = pk
	pathutil.EnsureDir(node.dir())
	defer func() {
		require.NoError(t, os.RemoveAll(node.dir()))
	}()

	require.NoError(t, node.StartApp("skychat"))
	time.Sleep(100 * time.Millisecond)

	require.NotNil(t, node.startedApps["skychat"])

	executer.Lock()
	require.Len(t, executer.cmds, 1)
	assert.Equal(t, "skychat.v1.0", executer.cmds[0].Path)
	assert.Equal(t, "skychat/v1.0", executer.cmds[0].Dir)
	assert.Equal(t, []string{"skychat.v1.0", "foo"}, executer.cmds[0].Args)
	executer.Unlock()

	ports := r.Ports()
	require.Len(t, ports, 1)
	assert.Equal(t, routing.Port(10), ports[0])

	require.NoError(t, node.StopApp("skychat"))
}

func TestNodeSpawnAppValidations(t *testing.T) {
	conn, _ := net.Pipe()
	r := new(mockRouter)
	executer := &MockExecuter{err: errors.New("foo")}
	defer func() {
		require.NoError(t, os.RemoveAll("skychat"))
	}()
	node := &Node{router: r, executer: executer,
		startedApps: map[string]*appBind{"skychat": {conn, 10}},
		logger:      logging.MustGetLogger("test")}

	cases := []struct {
		conf *AppConfig
		err  string
	}{
		{&AppConfig{App: "skychat", Version: "1.0", Port: 2}, "can't bind to reserved port 2"},
		{&AppConfig{App: "skychat", Version: "1.0", Port: 10}, "app skychat is already started"},
		{&AppConfig{App: "foo", Version: "1.0", Port: 11}, "failed to run app executable: foo"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.err, func(t *testing.T) {
			errCh := make(chan error)
			go func() {
				errCh <- node.SpawnApp(tc.conf, nil)
			}()

			time.Sleep(100 * time.Millisecond)
			require.NoError(t, node.Close())
			err := <-errCh
			require.Error(t, err)
			assert.Equal(t, tc.err, err.Error())
		})
	}
}

type MockExecuter struct {
	sync.Mutex
	err    error
	cmds   []*exec.Cmd
	stopCh chan struct{}
}

func (exc *MockExecuter) Start(cmd *exec.Cmd) (int, error) {
	exc.Lock()
	defer exc.Unlock()
	if exc.stopCh != nil {
		return -1, errors.New("already executing")
	}

	exc.stopCh = make(chan struct{})

	if exc.err != nil {
		return -1, exc.err
	}

	if exc.cmds == nil {
		exc.cmds = make([]*exec.Cmd, 0)
	}

	exc.cmds = append(exc.cmds, cmd)

	return 10, nil
}

func (exc *MockExecuter) Stop(pid int) error {
	exc.Lock()
	if exc.stopCh != nil {
		select {
		case <-exc.stopCh:
		default:
			close(exc.stopCh)
		}
	}
	exc.Unlock()
	return nil
}

func (exc *MockExecuter) Wait(cmd *exec.Cmd) error {
	<-exc.stopCh
	return nil
}

type mockRouter struct {
	sync.Mutex

	ports []routing.Port

	didStart bool
	didClose bool

	errChan chan error
}

func (r *mockRouter) Ports() []routing.Port {
	r.Lock()
	p := r.ports
	r.Unlock()
	return p
}

func (r *mockRouter) Serve(_ context.Context) error {
	r.didStart = true
	return nil
}

func (r *mockRouter) ServeApp(conn net.Conn, port routing.Port, appConf *app.Config) error {
	r.Lock()
	if r.ports == nil {
		r.ports = []routing.Port{}
	}

	r.ports = append(r.ports, port)
	r.Unlock()

	if r.errChan == nil {
		r.Lock()
		r.errChan = make(chan error)
		r.Unlock()
	}

	return <-r.errChan
}

func (r *mockRouter) Close() error {
	if r == nil {
		return nil
	}
	r.didClose = true
	r.Lock()
	if r.errChan != nil {
		close(r.errChan)
	}
	r.Unlock()
	return nil
}

func (r *mockRouter) IsSetupTransport(tr *transport.ManagedTransport) bool {
	return false
}

func (r *mockRouter) SetupIsTrusted(sPK cipher.PubKey) bool {
	return true
}
