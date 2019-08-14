package visor

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/skycoin/dmsg/cipher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/internal/httpauth"
	"github.com/skycoin/skywire/pkg/routing"
)

func TestMessagingDiscovery(t *testing.T) {
	pk, sk := cipher.GenerateKeyPair()
	conf := Config{}
	conf.Node.StaticPubKey = pk
	conf.Node.StaticSecKey = sk
	conf.Dmsg.Discovery = "skywire.skycoin.net:8001"
	conf.Dmsg.ServerCount = 10

	c, err := conf.DmsgConfig()
	require.NoError(t, err)

	assert.NotNil(t, c.Discovery)
	assert.False(t, c.PubKey.Null())
	assert.False(t, c.SecKey.Null())
	assert.Equal(t, 5, c.Retries)
	assert.Equal(t, time.Second, c.RetryDelay)
}

func TestTransportDiscovery(t *testing.T) {
	pk, _ := cipher.GenerateKeyPair()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewEncoder(w).Encode(&httpauth.NextNonceResponse{Edge: pk, NextNonce: 1}))
	}))
	defer srv.Close()

	conf := Config{}
	conf.Transport.Discovery = srv.URL

	discovery, err := conf.TransportDiscovery()
	require.NoError(t, err)

	assert.NotNil(t, discovery)
}

func TestTransportLogStore(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "foo")
	defer func() {
		require.NoError(t, os.RemoveAll(dir))
	}()

	conf := Config{}
	conf.Transport.LogStore.Type = "file"
	conf.Transport.LogStore.Location = dir
	ls, err := conf.TransportLogStore()
	require.NoError(t, err)
	require.NotNil(t, ls)

	conf.Transport.LogStore.Type = "memory"
	conf.Transport.LogStore.Location = ""
	ls, err = conf.TransportLogStore()
	require.NoError(t, err)
	require.NotNil(t, ls)
}

func TestRoutingTable(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "routing")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.Remove(tmpfile.Name()))
	}()

	conf := Config{}
	conf.Routing.Table.Type = "boltdb"
	conf.Routing.Table.Location = tmpfile.Name()
	_, err = conf.RoutingTable()
	require.NoError(t, err)

	conf.Routing.Table.Type = "memory"
	conf.Routing.Table.Location = ""
	_, err = conf.RoutingTable()
	require.NoError(t, err)
}

func TestAppsConfig(t *testing.T) {
	conf := Config{Version: "1.0"}
	conf.Apps = []AppConfig{
		{App: "foo", Version: "1.1", Port: 1},
		{App: "bar", AutoStart: true, Port: 2},
	}

	appsConf, err := conf.AppsConfig()
	require.NoError(t, err)

	app1 := appsConf[0]
	assert.Equal(t, "foo", app1.App)
	assert.Equal(t, "1.1", app1.Version)
	assert.Equal(t, routing.Port(1), app1.Port)
	assert.False(t, app1.AutoStart)

	app2 := appsConf[1]
	assert.Equal(t, "bar", app2.App)
	assert.Equal(t, "1.0", app2.Version)
	assert.Equal(t, routing.Port(2), app2.Port)
	assert.True(t, app2.AutoStart)
}

func TestAppsDir(t *testing.T) {
	conf := Config{AppsPath: "apps"}
	dir, err := conf.AppsDir()
	require.NoError(t, err)

	defer func() {
		require.NoError(t, os.Remove(dir))
	}()

	_, err = os.Stat(dir)
	assert.NoError(t, err)
}

func TestLocalDir(t *testing.T) {
	conf := Config{LocalPath: "local"}
	dir, err := conf.LocalDir()
	require.NoError(t, err)

	defer func() {
		require.NoError(t, os.Remove(dir))
	}()

	_, err = os.Stat(dir)
	assert.NoError(t, err)
}
