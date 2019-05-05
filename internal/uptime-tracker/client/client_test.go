package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/internal/httpauth"
	"github.com/skycoin/skywire/pkg/cipher"
)

var testPubKey, testSecKey = cipher.GenerateKeyPair()

func TestClientAuth(t *testing.T) {
	wg := sync.WaitGroup{}

	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			switch url := r.URL.String(); url {
			case "/":
				defer wg.Done()
				assert.Equal(t, testPubKey.Hex(), r.Header.Get("SW-Public"))
				assert.Equal(t, "1", r.Header.Get("SW-Nonce"))
				assert.NotEmpty(t, r.Header.Get("SW-Sig")) // TODO: check for the right key

			case fmt.Sprintf("/security/nonces/%s", testPubKey):
				fmt.Fprintf(w, `{"edge": "%s", "next_nonce": 1}`, testPubKey)

			default:
				t.Errorf("Don't know how to handle URL = '%s'", url)
			}
		},
	))
	defer srv.Close()

	client, err := NewHTTP(srv.URL, testPubKey, testSecKey)
	require.NoError(t, err)
	c := client.(*httpClient)

	wg.Add(1)
	_, err = c.Get(context.Background(), "/")
	require.NoError(t, err)

	wg.Wait()
}

func TestUpdateNodeUptime(t *testing.T) {
	srv := httptest.NewServer(authHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/update", r.URL.String())
	})))
	defer srv.Close()

	c, err := NewHTTP(srv.URL, testPubKey, testSecKey)
	require.NoError(t, err)
	err = c.UpdateNodeUptime(context.Background())
	require.NoError(t, err)
}

func authHandler(next http.Handler) http.Handler {
	m := http.NewServeMux()
	m.Handle("/security/nonces/", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(&httpauth.NextNonceResponse{Edge: testPubKey, NextNonce: 1}) // nolint: errcheck
		},
	))
	m.Handle("/", next)
	return m
}
