// +build integration

package client_test

import (
	"context"
	"fmt"
	"net"
	"sort"
	"testing"

	"github.com/skycoin/dmsg/cipher"
	"github.com/stretchr/testify/assert"

	"github.com/skycoin/skywire/pkg/messaging-discovery/api"
	"github.com/skycoin/skywire/pkg/messaging-discovery/client"
	"github.com/skycoin/skywire/pkg/messaging-discovery/store"
)

func TestGetAvailableServers(t *testing.T) {
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
	ephemeralPk2, ephemeralSk2 := cipher.GenerateKeyPair()
	baseEntry := newTestEntry(pk, serverStaticPk, ephemeralPk1, ephemeralPk2)

	cases := []struct {
		name                      string
		databaseAndEntriesPrehook func(store.Storer, *[]*client.Entry)
		responseIsError           bool
		errorMessage              client.HTTPMessage
	}{
		{
			name:            "get 3 server entries",
			responseIsError: false,
			databaseAndEntriesPrehook: func(db store.Storer, entries *[]*client.Entry) {
				var entry1, entry2, entry3 client.Entry
				client.Copy(&entry1, &baseEntry)
				client.Copy(&entry2, &baseEntry)
				client.Copy(&entry3, &baseEntry)

				entry1.Sign(sk)
				entry2.Keys.Static = ephemeralPk1
				entry2.Sign(ephemeralSk1)
				entry3.Keys.Static = ephemeralPk2
				entry3.Sign(ephemeralSk2)

				db.SetEntry(context.Background(), &entry1)
				db.SetEntry(context.Background(), &entry2)
				db.SetEntry(context.Background(), &entry3)

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
			clientServer := client.New(apiServerAddress)
			mockStore.(*store.MockStore).Clear()
			expectedEntries := []*client.Entry{}

			if tc.databaseAndEntriesPrehook != nil {
				tc.databaseAndEntriesPrehook(mockStore, &expectedEntries)
			}

			entries, err := clientServer.AvailableServers(context.TODO())

			if !tc.responseIsError {
				assert.NoError(t, err)
				sort.Slice(expectedEntries, func(i, j int) bool { return expectedEntries[i].Keys.Static > expectedEntries[j].Keys.Static })
				sort.Slice(entries, func(i, j int) bool { return entries[i].Keys.Static > entries[j].Keys.Static })
				assert.EqualValues(t, expectedEntries, entries)
			} else {
				assert.Equal(t, tc.errorMessage.String(), err.Error())
			}
		})
	}
}
