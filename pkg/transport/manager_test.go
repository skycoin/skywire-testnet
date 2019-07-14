package transport

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/skycoin/src/util/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/internal/testhelpers"
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

func TestTransportManager(t *testing.T) {
	client := NewDiscoveryMock()
	logStore := InMemoryTransportLogStore()

	pk1, sk1 := cipher.GenerateKeyPair()
	pk2, sk2 := cipher.GenerateKeyPair()

	c1 := &ManagerConfig{pk1, sk1, client, logStore, nil}
	c2 := &ManagerConfig{pk2, sk2, client, logStore, nil}

	f1, f2 := NewMockFactoryPair(pk1, pk2)
	m1, err := NewManager(c1, f1)
	require.NoError(t, err)

	assert.Equal(t, []string{"mock"}, m1.Factories())

	errCh := make(chan error)
	go func() {
		errCh <- m1.Serve(context.TODO())
	}()

	m2, err := NewManager(c2, f2)
	require.NoError(t, err)

	var mu sync.Mutex
	m1Observed := uint32(0)

	acceptCh := m1.TrChan
	go func() {
		for tr := range acceptCh {
			mu.Lock()
			if tr.Accepted {
				m1Observed++
			}
			mu.Unlock()
		}
	}()

	m2Observed := uint32(0)
	dialCh := m2.TrChan
	go func() {
		for tr := range dialCh {
			mu.Lock()
			if !tr.Accepted {
				m2Observed++
			}
			mu.Unlock()
		}
	}()

	tr2, err := m2.CreateTransport(context.TODO(), pk1, "mock", true)
	require.NoError(t, err)

	time.Sleep(time.Second)

	tr1 := m1.Transport(tr2.Entry.ID)
	require.NotNil(t, tr1)

	dEntry, err := client.GetTransportByID(context.TODO(), tr2.Entry.ID)
	require.NoError(t, err)
	assert.Equal(t, SortPubKeys(pk2, pk1), dEntry.Entry.Edges())
	assert.True(t, dEntry.IsUp)

	require.NoError(t, m1.DeleteTransport(tr1.Entry.ID))
	dEntry, err = client.GetTransportByID(context.TODO(), tr1.Entry.ID)
	require.NoError(t, err)
	assert.False(t, dEntry.IsUp)

	buf := make([]byte, 3)
	_, err = tr2.Read(buf)
	require.Equal(t, io.EOF, err)

	time.Sleep(time.Second)

	dEntry, err = client.GetTransportByID(context.TODO(), tr1.Entry.ID)
	require.NoError(t, err)
	assert.True(t, dEntry.IsUp)

	require.NoError(t, m2.DeleteTransport(tr2.Entry.ID))
	dEntry, err = client.GetTransportByID(context.TODO(), tr2.Entry.ID)
	require.NoError(t, err)
	assert.False(t, dEntry.IsUp)

	require.NoError(t, m2.Close())
	require.NoError(t, m1.Close())
	require.NoError(t, <-errCh)

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	assert.Equal(t, uint32(2), m1Observed)
	assert.Equal(t, uint32(1), m2Observed)
	mu.Unlock()
}

func TestTransportManagerReEstablishTransports(t *testing.T) {
	client := NewDiscoveryMock()
	logStore := InMemoryTransportLogStore()

	pk1, sk1 := cipher.GenerateKeyPair()
	pk2, sk2 := cipher.GenerateKeyPair()

	c1 := &ManagerConfig{pk1, sk1, client, logStore, nil}
	c2 := &ManagerConfig{pk2, sk2, client, logStore, nil}

	f1, f2 := NewMockFactoryPair(pk1, pk2)
	m1, err := NewManager(c1, f1)
	require.NoError(t, err)
	assert.Equal(t, []string{"mock"}, m1.Factories())

	m1.reconnectTransports(context.TODO())

	m1errCh := make(chan error, 1)
	go func() { m1errCh <- m1.Serve(context.TODO()) }()

	m2, err := NewManager(c2, f2)
	require.NoError(t, err)

	tr2, err := m2.CreateTransport(context.TODO(), pk1, "mock", true)
	require.NoError(t, err)

	tr1 := m1.Transport(tr2.Entry.ID)
	require.NotNil(t, tr1)

	dEntry, err := client.GetTransportByID(context.TODO(), tr2.Entry.ID)
	require.NoError(t, err)
	assert.Equal(t, SortPubKeys(pk2, pk1), dEntry.Entry.Edges())
	assert.True(t, dEntry.IsUp)

	require.NoError(t, m2.Close())

	dEntry2, err := client.GetTransportByID(context.TODO(), tr2.Entry.ID)
	require.NoError(t, err)
	assert.False(t, dEntry2.IsUp)

	m2, err = NewManager(c2, f2)
	require.NoError(t, err)

	m2.reconnectTransports(context.TODO())

	m2errCh := make(chan error, 1)
	go func() { m2errCh <- m2.Serve(context.TODO()) }()

	// time.Sleep(time.Second * 1) // TODO: this time.Sleep looks fishy - figure out later
	dEntry3, err := client.GetTransportByID(context.TODO(), tr2.Entry.ID)
	require.NoError(t, err)

	assert.True(t, dEntry3.IsUp)

	require.NoError(t, m2.Close())
	require.NoError(t, m1.Close())

	require.NoError(t, <-m1errCh)
	require.NoError(t, <-m2errCh)
}

func TestTransportManagerLogs(t *testing.T) {
	client := NewDiscoveryMock()
	logStore1 := InMemoryTransportLogStore()
	logStore2 := InMemoryTransportLogStore()

	pk1, sk1 := cipher.GenerateKeyPair()
	pk2, sk2 := cipher.GenerateKeyPair()

	c1 := &ManagerConfig{pk1, sk1, client, logStore1, nil}
	c2 := &ManagerConfig{pk2, sk2, client, logStore2, nil}

	f1, f2 := NewMockFactoryPair(pk1, pk2)
	m1, err := NewManager(c1, f1)
	require.NoError(t, err)

	assert.Equal(t, []string{"mock"}, m1.Factories())

	errCh := make(chan error)
	go func() {
		errCh <- m1.Serve(context.TODO())
	}()

	m2, err := NewManager(c2, f2)
	require.NoError(t, err)

	tr2, err := m2.CreateTransport(context.TODO(), pk1, "mock", true)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	tr1 := m1.Transport(tr2.Entry.ID)
	require.NotNil(t, tr1)

	writeErrCh := make(chan error, 1)
	go func() {
		_, writeErr := tr1.Write([]byte("foo"))
		writeErrCh <- writeErr
	}()
	buf := make([]byte, 3)
	_, err = tr2.Read(buf)
	require.NoError(t, err)

	// 2x log write interval just to be safe.
	time.Sleep(logWriteInterval * 2)

	entry1, err := logStore1.Entry(tr1.Entry.ID)
	require.NoError(t, err)
	assert.Equal(t, uint64(3), entry1.SentBytes)
	assert.Equal(t, uint64(0), entry1.RecvBytes)

	entry2, err := logStore2.Entry(tr1.Entry.ID)
	require.NoError(t, err)
	assert.Equal(t, uint64(0), entry2.SentBytes)
	assert.Equal(t, uint64(3), entry2.RecvBytes)

	require.NoError(t, m2.Close())
	require.NoError(t, m1.Close())
	require.NoError(t, <-errCh)
	require.NoError(t, testhelpers.NoErrorWithinTimeout(writeErrCh))
}

func ExampleSortPubKeys() {
	keyA, _ := cipher.GenerateKeyPair()
	keyB, _ := cipher.GenerateKeyPair()

	sortedKeysAB := SortPubKeys(keyA, keyB)
	sortedKeysBA := SortPubKeys(keyB, keyA)
	_ = SortPubKeys(keyA, keyA)
	fmt.Println("SortPubKeys(keyA, keyA) is successful")

	if sortedKeysAB == sortedKeysBA {
		fmt.Println("SortPubKeys(keyA, keyB) == SortPubKeys(keyB, keyA)")
	}

	// Output: SortPubKeys(keyA, keyA) is successful
	// SortPubKeys(keyA, keyB) == SortPubKeys(keyB, keyA)
}

func ExampleMakeTransportID() {
	keyA, _ := cipher.GenerateKeyPair()
	keyB, _ := cipher.GenerateKeyPair()

	uuidAB := MakeTransportID(keyA, keyB, "type", true)

	for i := 0; i < 256; i++ {
		if MakeTransportID(keyA, keyB, "type", true) != uuidAB {
			fmt.Println("uuid is unstable")
			break
		}
	}
	fmt.Printf("uuid is stable\n")

	uuidBA := MakeTransportID(keyB, keyA, "type", true)
	if uuidAB == uuidBA {
		fmt.Println("uuid is bidirectional")
	} else {
		fmt.Printf("keyA = %v\n keyB=%v\n uuidAB=%v\n uuidBA=%v\n", keyA, keyB, uuidAB, uuidBA)
	}

	_ = MakeTransportID(keyA, keyA, "type", true) // works for equal keys
	fmt.Println("works for equal keys")

	if MakeTransportID(keyA, keyB, "type", true) != MakeTransportID(keyA, keyB, "another_type", true) {
		fmt.Println("uuid is different for different types")
	}

	if MakeTransportID(keyA, keyB, "type", true) != MakeTransportID(keyA, keyB, "type", false) {
		fmt.Println("uuid is different for public and private transports")
	}

	// Output: uuid is stable
	// uuid is bidirectional
	// works for equal keys
	// uuid is different for different types
	// uuid is different for public and private transports
}

func ExampleManager_CreateTransport() {
	// Repetition is required here to guarantee that correctness does not depends on order of edges
	for i := 0; i < 4; i++ {
		pkB, mgrA, err := MockTransportManager()
		if err != nil {
			fmt.Printf("MockTransportManager failed on iteration %v with: %v\n", i, err)
			return
		}

		mtrAB, err := mgrA.CreateTransport(context.TODO(), pkB, "mock", true)
		if err != nil {
			fmt.Printf("Manager.CreateTransport failed on iteration %v with: %v\n", i, err)
			return
		}

		if (mtrAB.Entry.ID == uuid.UUID{}) {
			fmt.Printf("Manager.CreateTransport failed on iteration %v", i)
			return
		}
	}

	fmt.Println("Manager.CreateTransport success")

	// Output: Manager.CreateTransport success
}
