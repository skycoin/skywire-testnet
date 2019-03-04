// +build integration

package client_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/pkg/cipher"
	"github.com/skycoin/skywire/pkg/messaging-discovery/api"
	"github.com/skycoin/skywire/pkg/messaging-discovery/client"
	"github.com/skycoin/skywire/pkg/messaging-discovery/store"
)

func TestUpdateEntriesEndpoint(t *testing.T) {
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
	ephemeralPk1, ephemeralSk1 := cipher.GenerateKeyPair()
	ephemeralPk2, _ := cipher.GenerateKeyPair()
	baseEntry := newTestEntry(pk, serverStaticPk, ephemeralPk1, ephemeralPk2)
	baseEntry.Sign(sk)

	mockStore.SetEntry(context.TODO(), &baseEntry)

	cases := []struct {
		name                string
		httpResponse        client.HTTPMessage
		secretKey           cipher.SecKey
		responseShouldError bool
		entryPreHook        func(entry *client.Entry)
		storerPreHook       func(*testing.T, store.Storer, *client.Entry)
	}{

		{
			name:                "update entry iteration",
			responseShouldError: false,
			secretKey:           sk,
			storerPreHook: func(t *testing.T, s store.Storer, e *client.Entry) {
				err := s.SetEntry(context.TODO(), e)
				require.NoError(t, err)
				e.Server.Address = "different one"
			},
		},
		{
			name:                "update a client entry with no ephemerals",
			responseShouldError: false,
			secretKey:           sk,
			storerPreHook: func(t *testing.T, s store.Storer, e *client.Entry) {
				err := s.SetEntry(context.TODO(), e)
				require.NoError(t, err)
				e.Server = nil
				e.Keys.Ephemerals = nil
			},
		},
		{
			name:                "update entry unauthorized",
			responseShouldError: true,
			secretKey:           ephemeralSk1,
			httpResponse:        client.HTTPMessage{Code: http.StatusUnauthorized, Message: client.ErrUnauthorized.Error()},
			storerPreHook: func(t *testing.T, s store.Storer, e *client.Entry) {
				e.Server.Address = "different one"
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clientServer := client.New(apiServerAddress)
			entry := baseEntry

			if tc.entryPreHook != nil {
				tc.entryPreHook(&entry)
			}

			mockStore.(*store.MockStore).Clear()

			if tc.storerPreHook != nil {
				tc.storerPreHook(t, mockStore, &entry)
			}

			err := clientServer.UpdateEntry(context.TODO(), tc.secretKey, &entry)

			if tc.responseShouldError {
				assert.Error(t, err)
				assert.Equal(t, tc.httpResponse.String(), err.Error())
			} else {
				assert.NoError(t, err)
			}

		})
	}
}
