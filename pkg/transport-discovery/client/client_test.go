package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"github.com/SkycoinProject/dmsg/cipher"
	"github.com/SkycoinProject/skycoin/src/util/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SkycoinProject/skywire-mainnet/internal/httpauth"
	"github.com/SkycoinProject/skywire-mainnet/pkg/transport"
)

func TestMain(m *testing.M) {
	loggingLevel, ok := os.LookupEnv("TEST_LOGGING_LEVEL")
	if ok {
		lvl, err := logging.LevelFromString(loggingLevel)
		if err != nil {
			log.Fatal(err)
		}
		logging.SetLevel(lvl)
	} else {
		logging.Disable()
	}

	os.Exit(m.Run())
}

var testPubKey, testSecKey = cipher.GenerateKeyPair()

func newTestEntry() *transport.Entry {
	pk1, _ := cipher.GenerateKeyPair()
	entry := &transport.Entry{
		ID:     transport.MakeTransportID(pk1, testPubKey, "messaging"),
		Type:   "messaging",
		Public: true,
	}
	entry.SetEdges(pk1, testPubKey)

	return entry
}

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
				_, err := fmt.Fprintf(w, `{"edge": "%s", "next_nonce": 1}`, testPubKey)
				require.NoError(t, err)

			default:
				t.Errorf("Don't know how to handle URL = '%s'", url)
			}
		},
	))
	defer srv.Close()

	client, err := NewHTTP(srv.URL, testPubKey, testSecKey)
	require.NoError(t, err)
	c := client.(*apiClient)

	wg.Add(1)
	_, err = c.Post(context.Background(), "/", bytes.NewBufferString("test payload"))
	require.NoError(t, err)

	wg.Wait()
}

func TestRegisterTransportResponses(t *testing.T) {
	wg := sync.WaitGroup{}

	tests := []struct {
		name    string
		handler func(w http.ResponseWriter, r *http.Request)
		assert  func(err error)
	}{
		{
			"StatusCreated",
			func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusCreated) },
			func(err error) { require.NoError(t, err) },
		},
		{
			"StatusOK",
			func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) },
			func(err error) { require.Error(t, err) },
		},
		{
			"StatusInternalServerError",
			func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusInternalServerError) },
			func(err error) { require.Error(t, err) },
		},
		{
			"JSONError",
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				require.NoError(t, json.NewEncoder(w).Encode(Error{Error: "boom"}))
			},
			func(err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "status: 500")
				assert.Contains(t, err.Error(), "error: boom")
			},
		},
		{
			"NonJSONError",
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_, err := fmt.Fprintf(w, "boom")
				require.NoError(t, err)
			},
			func(err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "status: 500")
				assert.Contains(t, err.Error(), "error: boom")
			},
		},
		{
			"Request",
			func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/transports/", r.URL.String())
			},
			nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(_ *testing.T) {
			wg.Add(1)

			srv := httptest.NewServer(authHandler(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				defer wg.Done()
				tc.handler(w, r)
			})))
			defer srv.Close()

			c, err := NewHTTP(srv.URL, testPubKey, testSecKey)
			require.NoError(t, err)
			err = c.RegisterTransports(context.Background(), &transport.SignedEntry{})
			if tc.assert != nil {
				tc.assert(err)
			}

			wg.Wait()
		})
	}
}

func TestRegisterTransports(t *testing.T) {
	// Signatures does not matter in this test
	sEntry := &transport.SignedEntry{Entry: newTestEntry()}

	srv := httptest.NewServer(authHandler(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/transports/", r.URL.String())
		var entries []*transport.SignedEntry
		require.NoError(t, json.NewDecoder(r.Body).Decode(&entries))
		require.Len(t, entries, 1)
		assert.Equal(t, sEntry.Entry, entries[0].Entry)
		w.WriteHeader(http.StatusCreated)
	})))
	defer srv.Close()

	c, err := NewHTTP(srv.URL, testPubKey, testSecKey)
	require.NoError(t, err)
	require.NoError(t, c.RegisterTransports(context.Background(), sEntry))
}

func TestGetTransportByID(t *testing.T) {
	entry := &transport.EntryWithStatus{Entry: newTestEntry(), IsUp: true}
	srv := httptest.NewServer(authHandler(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, fmt.Sprintf("/transports/id:%s", entry.Entry.ID), r.URL.String())
		require.NoError(t, json.NewEncoder(w).Encode(entry))
	})))
	defer srv.Close()

	c, err := NewHTTP(srv.URL, testPubKey, testSecKey)
	require.NoError(t, err)
	resEntry, err := c.GetTransportByID(context.Background(), entry.Entry.ID)
	require.NoError(t, err)

	assert.Equal(t, entry.Entry, resEntry.Entry)
	assert.True(t, entry.IsUp)
}

func TestGetTransportsByEdge(t *testing.T) {
	entry := &transport.EntryWithStatus{Entry: newTestEntry(), IsUp: true}
	srv := httptest.NewServer(authHandler(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, fmt.Sprintf("/transports/edge:%s", entry.Entry.Edges[0]), r.URL.String())
		require.NoError(t, json.NewEncoder(w).Encode([]*transport.EntryWithStatus{entry}))
	})))
	defer srv.Close()

	c, err := NewHTTP(srv.URL, testPubKey, testSecKey)
	require.NoError(t, err)
	entries, err := c.GetTransportsByEdge(context.Background(), entry.Entry.Edges[0])
	require.NoError(t, err)

	require.Len(t, entries, 1)
	assert.Equal(t, entry.Entry, entries[0].Entry)
	assert.True(t, entries[0].IsUp)
}

func TestUpdateStatuses(t *testing.T) {
	entry := &transport.EntryWithStatus{Entry: newTestEntry(), IsUp: true}
	srv := httptest.NewServer(authHandler(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/statuses", r.URL.String())
		statuses := make([]*transport.Status, 0)
		require.NoError(t, json.NewDecoder(r.Body).Decode(&statuses))
		require.Len(t, statuses, 1)
		assert.Equal(t, entry.Entry.ID, statuses[0].ID)
		require.NoError(t, json.NewEncoder(w).Encode([]*transport.EntryWithStatus{entry}))
	})))
	defer srv.Close()

	c, err := NewHTTP(srv.URL, testPubKey, testSecKey)
	require.NoError(t, err)
	entries, err := c.UpdateStatuses(context.Background(), &transport.Status{ID: entry.Entry.ID, IsUp: false})
	require.NoError(t, err)

	require.Len(t, entries, 1)
	assert.Equal(t, entry.Entry, entries[0].Entry)
}

func authHandler(t *testing.T, next http.Handler) http.Handler {
	m := http.NewServeMux()
	m.Handle("/security/nonces/", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			require.NoError(t, json.NewEncoder(w).Encode(&httpauth.NextNonceResponse{Edge: testPubKey, NextNonce: 1}))
		},
	))
	m.Handle("/", next)
	return m
}
