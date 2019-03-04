// +build integration

package client_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/skycoin/skywire/pkg/cipher"
	"github.com/skycoin/skywire/pkg/messaging-discovery/api"
	"github.com/skycoin/skywire/pkg/messaging-discovery/client"
	"github.com/skycoin/skywire/pkg/messaging-discovery/store"
)

func TestSetEntriesEndpoint(t *testing.T) {
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

	cases := []struct {
		name                string
		httpResponse        client.HTTPMessage
		responseShouldError bool
		entryPreHook        func(entry *client.Entry)
		storerPreHook       func(store.Storer, *client.Entry)
	}{

		{
			name:                "set entry right",
			responseShouldError: false,
			entryPreHook: func(e *client.Entry) {
				e.Sign(sk)
			},
		},
		{
			name:                "set entry iteration",
			responseShouldError: false,
			entryPreHook: func(e *client.Entry) {
				e.Sign(sk)
			},
			storerPreHook: func(s store.Storer, e *client.Entry) {
				e.Sequence = 0
				s.SetEntry(context.Background(), e)
				e.Sequence = 1
				e.Timestamp += 3
			},
		},
		{
			name:                "set entry iteration wrong sequence",
			responseShouldError: true,
			httpResponse:        client.HTTPMessage{Code: http.StatusUnprocessableEntity, Message: client.ErrValidationWrongSequence.Error()},
			entryPreHook: func(e *client.Entry) {
				e.Sign(sk)
			},
			storerPreHook: func(s store.Storer, e *client.Entry) {
				e.Sequence = 2
				s.SetEntry(context.Background(), e)
			},
		},
		{
			name:                "set entry iteration unauthorized",
			responseShouldError: true,
			httpResponse:        client.HTTPMessage{Code: http.StatusUnauthorized, Message: client.ErrUnauthorized.Error()},
			entryPreHook: func(e *client.Entry) {
				e.Sign(sk)
			},
			storerPreHook: func(s store.Storer, e *client.Entry) {
				e.Sequence = 0
				s.SetEntry(context.Background(), e)
				e.Signature = ""
				e.Sequence = 1
				e.Timestamp += 3
				e.Sign(ephemeralSk1)
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
				tc.storerPreHook(mockStore, &entry)
			}

			err := clientServer.SetEntry(context.TODO(), &entry)

			if tc.responseShouldError {
				assert.Error(t, err)
				assert.Equal(t, tc.httpResponse.String(), err.Error())
			} else {
				assert.NoError(t, err)
			}

		})
	}
}
