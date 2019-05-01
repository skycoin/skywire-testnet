package node

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/internal/httpauth"
	"github.com/skycoin/skywire/pkg/cipher"
)

func TestMessagingDiscovery(t *testing.T) {
	pk, sk := cipher.GenerateKeyPair()
	conf := Config{}
	conf.Node.PubKey = pk
	conf.Node.SecKey = sk
	conf.Messaging.Discovery = "skywire.skycoin.net:8001"
	conf.Messaging.ServerCount = 10

	c, err := conf.MessagingConfig()
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
		json.NewEncoder(w).Encode(&httpauth.NextNonceResponse{Edge: pk, NextNonce: 1}) // nolint: errcheck
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
	defer os.RemoveAll(dir)

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
	defer os.Remove(tmpfile.Name())

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

func TestAppsDir(t *testing.T) {
	conf := Config{AppsPath: "apps"}
	dir, err := conf.AppsDir()
	require.NoError(t, err)

	defer os.Remove(dir)

	_, err = os.Stat(dir)
	assert.NoError(t, err)
}

func TestLocalDir(t *testing.T) {
	conf := Config{LocalPath: "local"}
	dir, err := conf.LocalDir()
	require.NoError(t, err)

	defer os.Remove(dir)

	_, err = os.Stat(dir)
	assert.NoError(t, err)
}
