// +build integration

package client_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"

	"github.com/skycoin/dmsg/cipher"
	"github.com/stretchr/testify/assert"

	"github.com/skycoin/skywire/pkg/messaging-discovery/api"
	"github.com/skycoin/skywire/pkg/messaging-discovery/client"
	"github.com/skycoin/skywire/pkg/messaging-discovery/store"
)

func TestEntriesEndpoint(t *testing.T) {
	var apiServerAddress string
	var mockStore store.Storer

	mockStore = store.NewStore("mock", "")
	apiServer := api.New(mockStore, 5)

	// get a free port
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err)
	}
	apiServerAddress = fmt.Sprintf("http://localhost:%d", listener.Addr().(*net.TCPAddr).Port)

	go func() {
		apiServer.Start(listener)
	}()

	pk, sk := cipher.GenerateKeyPair()
	serverStaticPk, _ := cipher.GenerateKeyPair()
	ephemeralPk1, _ := cipher.GenerateKeyPair()
	ephemeralPk2, _ := cipher.GenerateKeyPair()
	baseEntry := newTestEntry(pk, serverStaticPk, ephemeralPk1, ephemeralPk2)

	cases := []struct {
		name            string
		httpResponse    client.HTTPMessage
		publicKey       cipher.PubKey
		responseIsEntry bool
		entry           client.Entry
		entryPreHook    func(*client.Entry)
		storerPreHook   func(store.Storer, *client.Entry)
	}{
		{
			name:            "get entry",
			publicKey:       pk,
			responseIsEntry: true,
			entry:           baseEntry,
			entryPreHook: func(e *client.Entry) {
				e.Sign(sk)
			},
			storerPreHook: func(s store.Storer, e *client.Entry) {
				s.SetEntry(context.Background(), e)
			},
		},
		{
			name:            "get not valid entry",
			publicKey:       pk,
			responseIsEntry: false,
			httpResponse:    client.HTTPMessage{Code: http.StatusNotFound, Message: client.ErrKeyNotFound.Error()},
			entry:           baseEntry,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clientServer := client.New(apiServerAddress)

			if tc.entryPreHook != nil {
				tc.entryPreHook(&tc.entry)
			}

			mockStore.(*store.MockStore).Clear()

			if tc.storerPreHook != nil {
				tc.storerPreHook(mockStore, &tc.entry)
			}

			entry, err := clientServer.Entry(context.TODO(), tc.publicKey)
			if tc.responseIsEntry {
				assert.NoError(t, err)
				assert.Equal(t, &tc.entry, entry)
			} else {
				assert.Equal(t, tc.httpResponse.String(), err.Error())
			}

		})
	}
}
