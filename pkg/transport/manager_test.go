package transport_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/skycoin/skywire/pkg/snet/snettest"

	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/transport"

	"github.com/skycoin/dmsg/cipher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestNewManager(t *testing.T) {
	tpDisc := transport.NewDiscoveryMock()

	keys := snettest.GenKeyPairs(2)
	nEnv := snettest.NewEnv(t, keys)
	defer nEnv.Teardown()

	// Prepare tp manager 0.
	pk0, sk0 := keys[0].PK, keys[0].SK
	ls0 := transport.InMemoryTransportLogStore()
	m0, err := transport.NewManager(nEnv.Nets[0], &transport.ManagerConfig{
		PubKey:          pk0,
		SecKey:          sk0,
		DiscoveryClient: tpDisc,
		LogStore:        ls0,
	})
	require.NoError(t, err)
	go m0.Serve(context.TODO())
	defer func() { require.NoError(t, m0.Close()) }()

	// Prepare tp manager 1.
	pk1, sk1 := keys[1].PK, keys[1].SK
	ls1 := transport.InMemoryTransportLogStore()
	m2, err := transport.NewManager(nEnv.Nets[1], &transport.ManagerConfig{
		PubKey:          pk1,
		SecKey:          sk1,
		DiscoveryClient: tpDisc,
		LogStore:        ls1,
	})
	require.NoError(t, err)
	go m2.Serve(context.TODO())
	defer func() { require.NoError(t, m2.Close()) }()

	// Create data transport between manager 1 & manager 2.
	tp2, err := m2.SaveTransport(context.TODO(), pk0, "dmsg")
	require.NoError(t, err)
	tp1 := m0.Transport(transport.MakeTransportID(pk0, pk1, "dmsg"))
	require.NotNil(t, tp1)

	fmt.Println("transports created")

	totalSent2 := 0
	totalSent1 := 0

	// Check read/writes are of expected.
	t.Run("check_read_write", func(t *testing.T) {

		for i := 0; i < 10; i++ {
			totalSent2 += i
			rID := routing.RouteID(i)
			payload := cipher.RandByte(i)
			require.NoError(t, tp2.WritePacket(context.TODO(), rID, payload))

			recv, err := m0.ReadPacket()
			require.NoError(t, err)
			require.Equal(t, rID, recv.RouteID())
			require.Equal(t, uint16(i), recv.Size())
			require.Equal(t, payload, recv.Payload())
		}

		for i := 0; i < 20; i++ {
			totalSent1 += i
			rID := routing.RouteID(i)
			payload := cipher.RandByte(i)
			require.NoError(t, tp1.WritePacket(context.TODO(), rID, payload))

			recv, err := m2.ReadPacket()
			require.NoError(t, err)
			require.Equal(t, rID, recv.RouteID())
			require.Equal(t, uint16(i), recv.Size())
			require.Equal(t, payload, recv.Payload())
		}
	})

	// Ensure tp log entries are of expected.
	t.Run("check_tp_logs", func(t *testing.T) {

		// 1.5x log write interval just to be safe.
		time.Sleep(time.Second * 9 / 2)

		entry1, err := ls0.Entry(tp1.Entry.ID)
		require.NoError(t, err)
		assert.Equal(t, uint64(totalSent1), entry1.SentBytes)
		assert.Equal(t, uint64(totalSent2), entry1.RecvBytes)

		entry2, err := ls1.Entry(tp2.Entry.ID)
		require.NoError(t, err)
		assert.Equal(t, uint64(totalSent2), entry2.SentBytes)
		assert.Equal(t, uint64(totalSent1), entry2.RecvBytes)
	})

	// Ensure deleting a transport works as expected.
	t.Run("check_delete_tp", func(t *testing.T) {

		// Make transport ID.
		tpID := transport.MakeTransportID(pk0, pk1, "dmsg")

		// Ensure transports are registered properly in tp discovery.
		entry, err := tpDisc.GetTransportByID(context.TODO(), tpID)
		require.NoError(t, err)
		assert.Equal(t, transport.SortEdges(pk0, pk1), entry.Entry.Edges)
		assert.True(t, entry.IsUp)

		m2.DeleteTransport(tp2.Entry.ID)
		entry, err = tpDisc.GetTransportByID(context.TODO(), tpID)
		require.NoError(t, err)
		assert.False(t, entry.IsUp)
	})
}

func TestSortEdges(t *testing.T) {
	for i := 0; i < 100; i++ {
		keyA, _ := cipher.GenerateKeyPair()
		keyB, _ := cipher.GenerateKeyPair()
		require.Equal(t, transport.SortEdges(keyA, keyB), transport.SortEdges(keyB, keyA))
	}
}

func TestMakeTransportID(t *testing.T) {
	t.Run("id_is_stable", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			keyA, _ := cipher.GenerateKeyPair()
			keyB, _ := cipher.GenerateKeyPair()
			idAB := transport.MakeTransportID(keyA, keyB, "type")
			idBA := transport.MakeTransportID(keyB, keyA, "type")
			require.Equal(t, idAB, idBA)
		}
	})
	t.Run("tpType_changes_id", func(t *testing.T) {
		keyA, _ := cipher.GenerateKeyPair()
		require.NotEqual(t, transport.MakeTransportID(keyA, keyA, "a"), transport.MakeTransportID(keyA, keyA, "b"))
	})
}
