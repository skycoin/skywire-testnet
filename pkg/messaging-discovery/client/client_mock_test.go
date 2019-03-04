package client_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/pkg/cipher"

	"github.com/skycoin/skywire/pkg/messaging-discovery/client"
)

func TestNewMockGetAvailableServers(t *testing.T) {
	pk, sk := cipher.GenerateKeyPair()
	baseEntry := client.Entry{
		Static:    pk,
		Timestamp: time.Now().Unix(),
		Client:    &client.Client{},
		Server: &client.Server{
			Address:              "localhost:8080",
			AvailableConnections: 3,
		},
		Version:  "0",
		Sequence: 1,
	}

	cases := []struct {
		name                      string
		databaseAndEntriesPrehook func(*testing.T, client.APIClient, *[]*client.Entry)
		responseIsError           bool
		errorMessage              client.HTTPMessage
	}{
		{
			name:            "get 3 server entries",
			responseIsError: false,
			databaseAndEntriesPrehook: func(t *testing.T, mockClient client.APIClient, entries *[]*client.Entry) {
				entry1 := baseEntry
				entry2 := baseEntry
				entry3 := baseEntry

				err := entry1.Sign(sk)
				require.NoError(t, err)
				err = mockClient.SetEntry(context.TODO(), &entry1)
				require.NoError(t, err)

				pk1, sk1 := cipher.GenerateKeyPair()
				entry2.Static = pk1
				err = entry2.Sign(sk1)
				require.NoError(t, err)
				err = mockClient.SetEntry(context.TODO(), &entry2)
				require.NoError(t, err)

				pk2, sk2 := cipher.GenerateKeyPair()
				entry3.Static = pk2
				err = entry3.Sign(sk2)
				require.NoError(t, err)
				err = mockClient.SetEntry(context.TODO(), &entry3)
				require.NoError(t, err)

				*entries = append(*entries, &entry1, &entry2, &entry3)
			},
		},
		{
			name:            "get no entries",
			responseIsError: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clientServer := client.NewMock()
			expectedEntries := []*client.Entry{}

			if tc.databaseAndEntriesPrehook != nil {
				tc.databaseAndEntriesPrehook(t, clientServer, &expectedEntries)
			}

			entries, err := clientServer.AvailableServers(context.TODO())

			if !tc.responseIsError {
				assert.NoError(t, err)
				assert.Equal(t, expectedEntries, entries)
			} else {
				assert.Equal(t, tc.errorMessage.String(), err.Error())
			}
		})
	}
}

func TestNewMockEntriesEndpoint(t *testing.T) {
	pk, sk := cipher.GenerateKeyPair()
	baseEntry := newTestEntry(pk)

	cases := []struct {
		name            string
		httpResponse    client.HTTPMessage
		publicKey       cipher.PubKey
		responseIsEntry bool
		entry           client.Entry
		entryPreHook    func(*testing.T, *client.Entry)
		storerPreHook   func(*testing.T, client.APIClient, *client.Entry)
	}{
		{
			name:            "get entry",
			publicKey:       pk,
			responseIsEntry: true,
			entry:           baseEntry,
			entryPreHook: func(t *testing.T, e *client.Entry) {
				err := e.Sign(sk)
				require.NoError(t, err)
			},
			storerPreHook: func(t *testing.T, apiClient client.APIClient, e *client.Entry) {
				err := apiClient.SetEntry(context.TODO(), e)
				require.NoError(t, err)
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
			clientServer := client.NewMock()

			if tc.entryPreHook != nil {
				tc.entryPreHook(t, &tc.entry)
			}

			if tc.storerPreHook != nil {
				tc.storerPreHook(t, clientServer, &tc.entry)
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

func TestNewMockSetEntriesEndpoint(t *testing.T) {
	pk, sk := cipher.GenerateKeyPair()
	_, ephemeralSk1 := cipher.GenerateKeyPair()
	baseEntry := newTestEntry(pk)

	cases := []struct {
		name                string
		httpResponse        client.HTTPMessage
		responseShouldError bool
		entryPreHook        func(t *testing.T, entry *client.Entry)
		storerPreHook       func(*testing.T, client.APIClient, *client.Entry)
	}{

		{
			name:                "set entry right",
			responseShouldError: false,
			entryPreHook: func(t *testing.T, e *client.Entry) {
				err := e.Sign(sk)
				require.NoError(t, err)
			},
		},
		{
			name:                "set entry iteration",
			responseShouldError: false,
			entryPreHook: func(t *testing.T, e *client.Entry) {
				err := e.Sign(sk)
				require.NoError(t, err)
			},
			storerPreHook: func(t *testing.T, s client.APIClient, e *client.Entry) {
				var oldEntry client.Entry
				client.Copy(&oldEntry, e)
				fmt.Println(oldEntry.Static)
				oldEntry.Sequence = 0
				err := oldEntry.Sign(sk)
				require.NoError(t, err)
				err = s.SetEntry(context.TODO(), &oldEntry)
				require.NoError(t, err)
				e.Sequence = 1
				e.Timestamp += 3
				err = e.Sign(sk)
				require.NoError(t, err)
			},
		},
		{
			name:                "set entry iteration wrong sequence",
			responseShouldError: true,
			httpResponse:        client.HTTPMessage{Code: http.StatusUnprocessableEntity, Message: client.ErrValidationWrongSequence.Error()},
			entryPreHook: func(t *testing.T, e *client.Entry) {
				err := e.Sign(sk)
				require.NoError(t, err)
			},
			storerPreHook: func(t *testing.T, s client.APIClient, e *client.Entry) {
				e.Sequence = 2
				err := s.SetEntry(context.TODO(), e)
				require.NoError(t, err)
			},
		},
		{
			name:                "set entry iteration unauthorized",
			responseShouldError: true,
			httpResponse:        client.HTTPMessage{Code: http.StatusUnauthorized, Message: client.ErrUnauthorized.Error()},
			entryPreHook: func(t *testing.T, e *client.Entry) {
				err := e.Sign(sk)
				require.NoError(t, err)
			},
			storerPreHook: func(t *testing.T, s client.APIClient, e *client.Entry) {
				e.Sequence = 0
				err := s.SetEntry(context.TODO(), e)
				require.NoError(t, err)
				e.Signature = ""
				e.Sequence = 1
				e.Timestamp += 3
				err = e.Sign(ephemeralSk1)
				require.NoError(t, err)
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clientServer := client.NewMock()
			var entry client.Entry
			client.Copy(&entry, &baseEntry)

			if tc.entryPreHook != nil {
				tc.entryPreHook(t, &entry)
			}

			if tc.storerPreHook != nil {
				tc.storerPreHook(t, clientServer, &entry)
			}

			fmt.Println("key in: ", entry.Static)
			err := clientServer.SetEntry(context.TODO(), &entry)

			if tc.responseShouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

		})
	}
}

func TestNewMockUpdateEntriesEndpoint(t *testing.T) {
	pk, sk := cipher.GenerateKeyPair()
	_, ephemeralSk1 := cipher.GenerateKeyPair()
	baseEntry := newTestEntry(pk)
	err := baseEntry.Sign(sk)
	require.NoError(t, err)

	cases := []struct {
		name                string
		secretKey           cipher.SecKey
		responseShouldError bool
		entryPreHook        func(entry *client.Entry)
		storerPreHook       func(client.APIClient, *client.Entry)
	}{

		{
			name:                "update entry iteration",
			responseShouldError: false,
			secretKey:           sk,
			storerPreHook: func(apiClient client.APIClient, e *client.Entry) {
				e.Server.Address = "different one"
			},
		},
		{
			name:                "update entry unauthorized",
			responseShouldError: true,
			secretKey:           ephemeralSk1,
			storerPreHook: func(apiClient client.APIClient, e *client.Entry) {
				e.Server.Address = "different one"
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clientServer := client.NewMock()
			err := clientServer.SetEntry(context.TODO(), &baseEntry)
			require.NoError(t, err)

			entry := baseEntry

			if tc.entryPreHook != nil {
				tc.entryPreHook(&entry)
			}

			if tc.storerPreHook != nil {
				tc.storerPreHook(clientServer, &entry)
			}

			err = clientServer.UpdateEntry(context.TODO(), tc.secretKey, &entry)

			if tc.responseShouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

		})
	}
}

func TestNewMockUpdateEntrySequence(t *testing.T) {
	clientServer := client.NewMock()
	pk, sk := cipher.GenerateKeyPair()
	entry := &client.Entry{
		Sequence: 0,
		Static:   pk,
	}

	err := clientServer.UpdateEntry(context.TODO(), sk, entry)
	require.NoError(t, err)

	v1Entry, err := clientServer.Entry(context.TODO(), pk)
	require.NoError(t, err)

	err = clientServer.UpdateEntry(context.TODO(), sk, entry)
	require.NoError(t, err)

	v2Entry, err := clientServer.Entry(context.TODO(), pk)
	require.NoError(t, err)

	assert.NotEqual(t, v1Entry.Sequence, v2Entry.Sequence)
}

func newTestEntry(pk cipher.PubKey) client.Entry {
	baseEntry := client.Entry{
		Static:    pk,
		Timestamp: time.Now().UnixNano(),
		Client:    &client.Client{},
		Server: &client.Server{
			Address:              "localhost:8080",
			AvailableConnections: 3,
		},
		Version:  "0",
		Sequence: 0,
	}
	return baseEntry
}
