package transport_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SkycoinProject/skywire-mainnet/pkg/transport"
)

func testTransportLogStore(t *testing.T, logStore transport.LogStore) {
	t.Helper()

	id1 := uuid.New()
	entry1 := new(transport.LogEntry)
	entry1.AddRecv(100)
	entry1.AddSent(200)

	id2 := uuid.New()
	entry2 := new(transport.LogEntry)
	entry2.AddRecv(300)
	entry2.AddSent(400)

	require.NoError(t, logStore.Record(id1, entry1))
	require.NoError(t, logStore.Record(id2, entry2))

	entry, err := logStore.Entry(id2)
	require.NoError(t, err)
	assert.Equal(t, uint64(300), entry.RecvBytes)
	assert.Equal(t, uint64(400), entry.SentBytes)
}

func TestInMemoryTransportLogStore(t *testing.T) {
	testTransportLogStore(t, transport.InMemoryTransportLogStore())
}

func TestFileTransportLogStore(t *testing.T) {
	dir, err := ioutil.TempDir("", "log_store")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.RemoveAll(dir))
	}()

	ls, err := transport.FileTransportLogStore(dir)
	require.NoError(t, err)
	testTransportLogStore(t, ls)
}

func TestLogEntry_MarshalJSON(t *testing.T) {
	entry := new(transport.LogEntry)
	entry.AddSent(10)
	entry.AddRecv(100)
	b, err := json.Marshal(entry)
	require.NoError(t, err)
	fmt.Println(string(b))
	b, err = json.MarshalIndent(entry, "", "\t")
	require.NoError(t, err)
	fmt.Println(string(b))
}

func TestLogEntry_GobEncode(t *testing.T) {
	var entry transport.LogEntry

	enc, err := entry.GobEncode()
	require.NoError(t, err)

	require.NoError(t, entry.GobDecode(enc))
}
