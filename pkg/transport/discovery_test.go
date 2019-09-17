package transport_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/SkycoinProject/dmsg/cipher"

	"github.com/SkycoinProject/skywire/pkg/transport"
)

func TestNewDiscoveryMock(t *testing.T) {
	dc := transport.NewDiscoveryMock()
	pk1, _ := cipher.GenerateKeyPair() // local
	pk2, _ := cipher.GenerateKeyPair()
	entry := &transport.Entry{Type: "mock", Edges: transport.SortEdges(pk1, pk2)}

	sEntry := &transport.SignedEntry{Entry: entry}

	require.NoError(t, dc.RegisterTransports(context.TODO(), sEntry))

	entryWS, err := dc.GetTransportByID(context.TODO(), sEntry.Entry.ID)
	require.NoError(t, err)
	require.True(t, entryWS.Entry.ID == sEntry.Entry.ID)

	entriesWS, err := dc.GetTransportsByEdge(context.TODO(), pk1)
	require.NoError(t, err)
	require.Equal(t, entry.Edges, entriesWS[0].Entry.Edges)

	_, err = dc.UpdateStatuses(context.TODO(), &transport.Status{})
	require.NoError(t, err)
}
