// +build !no_ci

package messaging

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/pkg/cipher"
	"github.com/skycoin/skywire/pkg/messaging-discovery/client"
)

func TestClientConnectInitialServers(t *testing.T) {
	pk, sk := cipher.GenerateKeyPair()
	discovery := client.NewMock()
	c := NewMsgFactory(&Config{pk, sk, discovery, 1, 100 * time.Millisecond})

	srv, err := newMockServer(discovery)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	require.NoError(t, c.ConnectToInitialServers(context.TODO(), 1))
	c.mu.RLock()
	require.Len(t, c.links, 1)
	c.mu.RUnlock()

	entry, err := discovery.Entry(context.TODO(), pk)
	require.NoError(t, err)
	assert.Len(t, entry.Client.DelegatedServers, 1)
	assert.Equal(t, srv.config.Public, entry.Client.DelegatedServers[0])

	c.mu.RLock()
	l := c.links[srv.config.Public]
	c.mu.RUnlock()
	require.NotNil(t, l)
	require.NoError(t, l.link.Close())

	time.Sleep(200 * time.Millisecond)

	c.mu.RLock()
	require.Len(t, c.links, 1)
	c.mu.RUnlock()

	require.NoError(t, c.Close())

	time.Sleep(100 * time.Millisecond)

	c.mu.RLock()
	require.Len(t, c.links, 0)
	c.mu.RUnlock()

	entry, err = discovery.Entry(context.TODO(), pk)
	require.NoError(t, err)
	require.Len(t, entry.Client.DelegatedServers, 0)
}
