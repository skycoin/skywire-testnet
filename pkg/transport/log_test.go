package transport_test

import (
	"io/ioutil"
	"math/big"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/pkg/transport"
)

func testTransportLogStore(t *testing.T, logStore transport.LogStore) {
	t.Helper()

	id1 := uuid.New()
	entry1 := &transport.LogEntry{big.NewInt(100), big.NewInt(200)}
	id2 := uuid.New()
	entry2 := &transport.LogEntry{big.NewInt(300), big.NewInt(400)}

	require.NoError(t, logStore.Record(id1, entry1))
	require.NoError(t, logStore.Record(id2, entry2))

	entry, err := logStore.Entry(id2)
	require.NoError(t, err)
	assert.Equal(t, int64(300), entry.ReceivedBytes.Int64())
	assert.Equal(t, int64(400), entry.SentBytes.Int64())
}

func TestInMemoryTransportLogStore(t *testing.T) {
	testTransportLogStore(t, transport.InMemoryTransportLogStore())
}

func TestFileTransportLogStore(t *testing.T) {
	dir, err := ioutil.TempDir("", "log_store")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	ls, err := transport.FileTransportLogStore(dir)
	require.NoError(t, err)
	testTransportLogStore(t, ls)
}
