package transport_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/SkycoinProject/SkycoinProject/src/util/logging"

	"github.com/SkycoinProject/skywire/pkg/routing"
	"github.com/SkycoinProject/skywire/pkg/transport"
	"github.com/SkycoinProject/skywire/pkg/transport/dmsg"

	"github.com/google/uuid"
	"github.com/SkycoinProject/dmsg/cipher"
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

	keys := dmsg.GenKeyPairs(2)
	dmsgEnv := dmsg.SetupTestEnv(t, keys)
	defer dmsgEnv.TearDown()

	// Prepare tp manager 1.
	pk1, sk1 := keys[0].PK, keys[0].SK
	ls1 := transport.InMemoryTransportLogStore()
	c1 := &transport.ManagerConfig{pk1, sk1, tpDisc, ls1, nil}
	f1 := dmsgEnv.Clients[0]
	m1, err := transport.NewManager(c1, nil, f1)
	require.NoError(t, err)
	m1Err := make(chan error, 1)
	go func() {
		m1Err <- m1.Serve(context.TODO())
		close(m1Err)
	}()
	defer func() {
		require.NoError(t, m1.Close())
		require.NoError(t, <-m1Err)
	}()
	fmt.Println("tp manager 1 prepared")

	// Prepare tp manager 2.
	pk2, sk2 := keys[1].PK, keys[1].SK
	ls2 := transport.InMemoryTransportLogStore()
	c2 := &transport.ManagerConfig{pk2, sk2, tpDisc, ls2, nil}
	f2 := dmsgEnv.Clients[1]
	m2, err := transport.NewManager(c2, nil, f2)
	require.NoError(t, err)
	m2Err := make(chan error, 1)
	go func() {
		m2Err <- m2.Serve(context.TODO())
		close(m2Err)
	}()
	defer func() {
		require.NoError(t, m2.Close())
		require.NoError(t, <-m2Err)
	}()
	fmt.Println("tp manager 2 prepared")

	// Create data transport between manager 1 & manager 2.
	tp2, err := m2.SaveTransport(context.TODO(), pk1, "dmsg")
	require.NoError(t, err)
	tp1 := m1.Transport(transport.MakeTransportID(pk1, pk2, "dmsg"))
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

			recv, err := m1.ReadPacket()
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

		entry1, err := ls1.Entry(tp1.Entry.ID)
		require.NoError(t, err)
		assert.Equal(t, uint64(totalSent1), entry1.SentBytes)
		assert.Equal(t, uint64(totalSent2), entry1.RecvBytes)

		entry2, err := ls2.Entry(tp2.Entry.ID)
		require.NoError(t, err)
		assert.Equal(t, uint64(totalSent2), entry2.SentBytes)
		assert.Equal(t, uint64(totalSent1), entry2.RecvBytes)
	})

	// Ensure deleting a transport works as expected.
	t.Run("check_delete_tp", func(t *testing.T) {

		// Make transport ID.
		tpID := transport.MakeTransportID(pk1, pk2, "dmsg")

		// Ensure transports are registered properly in tp discovery.
		entry, err := tpDisc.GetTransportByID(context.TODO(), tpID)
		require.NoError(t, err)
		assert.Equal(t, transport.SortEdges(pk1, pk2), entry.Entry.Edges)
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

func ExampleManager_SaveTransport() {
	// Repetition is required here to guarantee that correctness does not depends on order of edges
	for i := 0; i < 4; i++ {
		pkB, mgrA, err := transport.MockTransportManager()
		if err != nil {
			fmt.Printf("MockTransportManager failed on iteration %v with: %v\n", i, err)
			return
		}

		mtrAB, err := mgrA.SaveTransport(context.TODO(), pkB, "mock")
		if err != nil {
			fmt.Printf("Manager.SaveTransport failed on iteration %v with: %v\n", i, err)
			return
		}

		if (mtrAB.Entry.ID == uuid.UUID{}) {
			fmt.Printf("Manager.SaveTransport failed on iteration %v", i)
			return
		}
	}

	fmt.Println("Manager.SaveTransport success")

	// Output: Manager.SaveTransport success
}
