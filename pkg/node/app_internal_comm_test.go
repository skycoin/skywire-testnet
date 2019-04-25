package node

import (
	"encoding/json"
	"github.com/skycoin/skywire/internal/httpauth"
	"github.com/skycoin/skywire/pkg/cipher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestNewNode(t *testing.T) {
		pk, sk := cipher.GenerateKeyPair()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(&httpauth.NextNonceResponse{Edge: pk, NextNonce: 1}) // nolint: errcheck
		}))
		defer srv.Close()

		conf := Config{Version: "1.0", LocalPath: "local", AppsPath: "apps"}
		conf.Node.PubKey = pk
		conf.Node.SecKey = sk
		conf.Messaging.Discovery = "http://skywire.skycoin.net:8001"
		conf.Messaging.ServerCount = 10
		conf.Transport.Discovery = srv.URL
		conf.
		conf.Apps = []AppConfig{
			{App: "foo", Port: 1},
			{App: "bar", AutoStart: true, Port: 2},
		}

		defer os.RemoveAll("local")

		node, err := NewNode(&conf)
		require.NoError(t, err)

		assert.NotNil(t, node.r)
		assert.NotNil(t, node.appsPath)
		assert.NotNil(t, node.localPath)
		assert.NotNil(t, node.startedApps)
}
