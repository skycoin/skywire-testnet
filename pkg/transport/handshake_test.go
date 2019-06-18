package transport

import (
	"context"
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/pkg/cipher"
)

type hsMockEnv struct {
	client DiscoveryClient
	pk1    cipher.PubKey
	sk1    cipher.SecKey
	pk2    cipher.PubKey
	sk2    cipher.SecKey
	tr1    *MockTransport
	tr2    *MockTransport
	m1     *Manager
	err1   error
	m2     *Manager
	err2   error
}

func newHsMockEnv() *hsMockEnv {
	client := NewDiscoveryMock()

	pk1, sk1 := cipher.GenerateKeyPair()
	pk2, sk2 := cipher.GenerateKeyPair()

	in, out := net.Pipe()
	tr1 := NewMockTransport(in, pk1, pk2)
	tr2 := NewMockTransport(out, pk2, pk1)

	m1, err1 := NewManager(&ManagerConfig{PubKey: pk1, SecKey: sk1, DiscoveryClient: client})
	m2, err2 := NewManager(&ManagerConfig{PubKey: pk2, SecKey: sk2, DiscoveryClient: client})

	return &hsMockEnv{
		client: client,
		pk1:    pk1,
		sk1:    sk1,
		pk2:    pk2,
		sk2:    sk2,
		tr1:    tr1,
		tr2:    tr2,
		m1:     m1,
		err1:   err1,
		m2:     m2,
		err2:   err2,
	}
}

func TestHsMock(t *testing.T) {
	mockEnv := newHsMockEnv()
	require.NoError(t, mockEnv.err1)
	require.NoError(t, mockEnv.err2)
}

func Example_newHsMock() {
	mockEnv := newHsMockEnv()

	fmt.Printf("client is set: %v\n", mockEnv.client != nil)
	fmt.Printf("pk1 is set: %v\n", mockEnv.pk1 != cipher.PubKey{})
	fmt.Printf("sk1 is set: %v\n", mockEnv.sk1 != cipher.SecKey{})
	fmt.Printf("pk2 is set: %v\n", mockEnv.pk2 != cipher.PubKey{})
	fmt.Printf("sk2 is set: %v\n", mockEnv.sk2 != cipher.SecKey{})
	fmt.Printf("tr1 is set: %v\n", mockEnv.tr1 != nil)
	fmt.Printf("tr2 is set: %v\n", mockEnv.tr2 != nil)
	fmt.Printf("m1 is set: %v\n", mockEnv.m1 != nil)
	fmt.Printf("err1 is nil: %v\n", mockEnv.err1 == nil)
	fmt.Printf("m2 is set: %v\n", mockEnv.m2 != nil)
	fmt.Printf("err2 is nil: %v\n", mockEnv.err2 == nil)

	// Output: client is set: true
	// pk1 is set: true
	// sk1 is set: true
	// pk2 is set: true
	// sk2 is set: true
	// tr1 is set: true
	// tr2 is set: true
	// m1 is set: true
	// err1 is nil: true
	// m2 is set: true
	// err2 is nil: true
}

func Example_validateEntry() {
	pk1, sk1 := cipher.GenerateKeyPair()
	pk2, _ := cipher.GenerateKeyPair()
	pk3, _ := cipher.GenerateKeyPair()
	tr := NewMockTransport(nil, pk1, pk2)

	entryInvalidEdges := &SignedEntry{
		Entry: &Entry{Type: "mock",
			EdgeKeys: SortPubKeys(pk2, pk3),
		}}
	if err := validateSignedEntry(entryInvalidEdges, tr, pk1); err != nil {
		fmt.Println(err.Error())
	}

	entry := NewEntry(pk1, pk2, "mock", true)
	sEntry, ok := NewSignedEntry(entry, pk1, sk1)
	if !ok {
		fmt.Println("error creating signed entry")
	}
	if err := validateSignedEntry(sEntry, tr, pk1); err != nil {
		fmt.Println(err.Error())
	}

	// Output: invalid entry edges
}

func TestValidateEntry(t *testing.T) {
	pk1, sk1 := cipher.GenerateKeyPair()
	pk2, sk2 := cipher.GenerateKeyPair()
	pk3, _ := cipher.GenerateKeyPair()
	tr := NewMockTransport(nil, pk1, pk2)

	entry := &Entry{Type: "mock", EdgeKeys: SortPubKeys(pk2, pk1)}
	tcs := []struct {
		sEntry *SignedEntry
		err    string
	}{
		{
			&SignedEntry{Entry: &Entry{Type: "foo"}},
			"invalid entry type",
		},
		{
			&SignedEntry{Entry: &Entry{Type: "mock", EdgeKeys: SortPubKeys(pk1, pk3)}},
			"invalid entry edges",
		},
		{
			&SignedEntry{Entry: &Entry{Type: "mock", EdgeKeys: SortPubKeys(pk2, pk1)}},
			"invalid entry signature",
		},
		{
			&SignedEntry{Entry: entry, Signatures: [2]cipher.Sig{}},
			"invalid entry signature",
		},
		{
			func() *SignedEntry {
				sEntry := &SignedEntry{Entry: entry, Signatures: [2]cipher.Sig{}}
				_ = sEntry.Sign(pk1, sk2) // nolint
				_ = sEntry.Sign(pk2, sk1) // nolint
				return sEntry
			}(),
			"Recovered pubkey does not match pubkey",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.err, func(t *testing.T) {
			err := validateSignedEntry(tc.sEntry, tr, pk2)
			require.Error(t, err)
			assert.Equal(t, tc.err, err.Error())
		})
	}

	sEntry := &SignedEntry{Entry: entry, Signatures: [2]cipher.Sig{}}
	require.True(t, sEntry.Sign(pk1, sk1))
	require.True(t, sEntry.Sign(pk2, sk2))

	require.NoError(t, validateSignedEntry(sEntry, tr, pk1))
}

func TestSettlementHandshake(t *testing.T) {

	mockEnv := newHsMockEnv()
	t.Run("Create Mock Env", func(t *testing.T) {
		require.NoError(t, mockEnv.err1)
		require.NoError(t, mockEnv.err2)
	})

	errCh := make(chan error)
	var resEntry *Entry
	go func() {
		e, err := settlementResponderHandshake(mockEnv.m2, mockEnv.tr2)
		resEntry = e
		errCh <- err
	}()

	entry, err := settlementInitiatorHandshake(true)(mockEnv.m1, mockEnv.tr1)
	require.NoError(t, <-errCh)
	require.NoError(t, err)

	require.NotNil(t, resEntry)
	require.NotNil(t, entry)

	assert.Equal(t, entry.ID, resEntry.ID)

	dEntry, err := mockEnv.client.GetTransportByID(context.TODO(), entry.ID)
	require.NoError(t, err)

	assert.Equal(t, entry, dEntry.Entry)

}

func TestSettlementHandshakePrivate(t *testing.T) {
	mockEnv := newHsMockEnv()

	require.NoError(t, mockEnv.err1)
	require.NoError(t, mockEnv.err2)

	errCh := make(chan error)
	var resEntry *Entry
	go func() {
		e, err := settlementResponderHandshake(mockEnv.m2, mockEnv.tr2)
		resEntry = e
		errCh <- err
	}()

	entry, err := settlementInitiatorHandshake(false)(mockEnv.m1, mockEnv.tr1)
	require.NoError(t, <-errCh)
	require.NoError(t, err)

	require.NotNil(t, resEntry)
	require.NotNil(t, entry)

	assert.Equal(t, entry.ID, resEntry.ID)
	_, err = mockEnv.client.GetTransportByID(context.TODO(), entry.ID)
	require.Error(t, err)

}

func TestSettlementHandshakeExistingTransport(t *testing.T) {
	mockEnv := newHsMockEnv()

	require.NoError(t, mockEnv.err1)
	require.NoError(t, mockEnv.err2)

	tpType := "mock"
	entry := &Entry{
		ID:       MakeTransportID(mockEnv.pk1, mockEnv.pk2, tpType, true),
		EdgeKeys: SortPubKeys(mockEnv.pk1, mockEnv.pk2),
		Type:     tpType,
		Public:   true,
	}

	mockEnv.m1.entries[*entry] = struct{}{}
	mockEnv.m2.entries[*entry] = struct{}{}

	t.Run("RegisterTransports", func(t *testing.T) {
		require.NoError(t, mockEnv.client.RegisterTransports(context.TODO(), &SignedEntry{Entry: entry}))
	})

	t.Run("UpdateStatuses", func(t *testing.T) {
		_, err := mockEnv.client.UpdateStatuses(context.Background(), &Status{ID: entry.ID, IsUp: false})
		require.NoError(t, err)
	})

	errCh := make(chan error)
	var resEntry *Entry
	go func() {
		e, err := settlementResponderHandshake(mockEnv.m2, mockEnv.tr2)
		resEntry = e
		errCh <- err
	}()

	entry, err := settlementInitiatorHandshake(true)(mockEnv.m1, mockEnv.tr1)
	require.NoError(t, <-errCh)
	require.NoError(t, err)

	require.NotNil(t, resEntry)
	require.NotNil(t, entry)

	assert.Equal(t, entry.ID, resEntry.ID)
	dEntry, err := mockEnv.client.GetTransportByID(context.TODO(), entry.ID)
	require.NoError(t, err)

	assert.True(t, dEntry.IsUp)

}

func Example_validateSignedEntry() {
	mockEnv := newHsMockEnv()

	tm, tr := mockEnv.m1, mockEnv.tr1
	entry := NewEntry(mockEnv.pk1, mockEnv.pk2, "mock", true)
	sEntry, ok := NewSignedEntry(entry, tm.config.PubKey, tm.config.SecKey)
	if !ok {
		fmt.Println("error creating signed entry")
	}
	if err := validateSignedEntry(sEntry, tr, tm.config.PubKey); err != nil {
		fmt.Printf("NewSignedEntry: %v", err.Error())
	}

	fmt.Printf("System is working")
	// Output: System is working
}

func Example_settlementInitiatorHandshake() {
	mockEnv := newHsMockEnv()

	initHandshake := settlementInitiatorHandshake(true)
	respondHandshake := settlementResponderHandshake

	errCh := make(chan error)
	go func() {
		entry, err := initHandshake(mockEnv.m1, mockEnv.tr1)
		if err != nil {
			fmt.Printf("initHandshake error: %v\n entry:\n%v\n", err.Error(), entry)
			errCh <- err
		}
		errCh <- nil
	}()

	go func() {
		if _, err := respondHandshake(mockEnv.m2, mockEnv.tr2); err != nil {
			fmt.Printf("respondHandshake error: %v\n", err.Error())
			errCh <- err
		}
		errCh <- nil
	}()

	<-errCh
	<-errCh

	_ = mockEnv
	_ = initHandshake
	_ = respondHandshake
	fmt.Println("System is working")
	// Output: System is working
}
