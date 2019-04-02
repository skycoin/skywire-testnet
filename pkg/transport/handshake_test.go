package transport

import (
	"context"
	"net"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/pkg/cipher"
)

func TestSettlementHandshake(t *testing.T) {
	client := NewDiscoveryMock()

	pk1, sk1 := cipher.GenerateKeyPair()
	pk2, sk2 := cipher.GenerateKeyPair()

	in, out := net.Pipe()
	tr1 := NewMockTransport(in, pk1, pk2)
	tr2 := NewMockTransport(out, pk2, pk1)

	m1, err := NewManager(&ManagerConfig{SecKey: sk1, DiscoveryClient: client})
	require.NoError(t, err)
	m2, err := NewManager(&ManagerConfig{SecKey: sk2, DiscoveryClient: client})
	require.NoError(t, err)

	errCh := make(chan error)
	var resEntry *Entry
	go func() {
		e, err := settlementResponderHandshake(m2, tr2)
		resEntry = e
		errCh <- err
	}()

	entry, err := settlementInitiatorHandshake(uuid.UUID{}, true)(m1, tr1)
	require.NoError(t, <-errCh)
	require.NoError(t, err)

	require.NotNil(t, resEntry)
	require.NotNil(t, entry)

	assert.Equal(t, entry.ID, resEntry.ID)
	dEntry, err := client.GetTransportByID(context.TODO(), entry.ID)
	require.NoError(t, err)

	assert.Equal(t, entry, dEntry.Entry)
}

func TestSettlementHandshakeInvalidSig(t *testing.T) {
	client := NewDiscoveryMock()

	pk1, sk1 := cipher.GenerateKeyPair()
	pk2, sk2 := cipher.GenerateKeyPair()

	in, out := net.Pipe()
	tr1 := NewMockTransport(in, pk1, pk2)
	tr2 := NewMockTransport(out, pk2, pk1)

	m1, err := NewManager(&ManagerConfig{SecKey: sk1, DiscoveryClient: client})
	require.NoError(t, err)
	m2, err := NewManager(&ManagerConfig{SecKey: sk2, DiscoveryClient: client})
	require.NoError(t, err)

	go settlementInitiatorHandshake(uuid.UUID{}, true)(m2, tr1) // nolint: errcheck
	_, err = settlementResponderHandshake(m2, tr2)
	require.Error(t, err)
	assert.Equal(t, "Recovered pubkey does not match pubkey", err.Error())

	in, out = net.Pipe()
	tr1 = NewMockTransport(in, pk1, pk2)
	tr2 = NewMockTransport(out, pk2, pk1)

	go settlementResponderHandshake(m1, tr2) // nolint: errcheck
	_, err = settlementInitiatorHandshake(uuid.UUID{}, true)(m1, tr1)
	require.Error(t, err)
	assert.Equal(t, "Recovered pubkey does not match pubkey", err.Error())
}

func TestSettlementHandshakePrivate(t *testing.T) {
	client := NewDiscoveryMock()

	pk1, sk1 := cipher.GenerateKeyPair()
	pk2, sk2 := cipher.GenerateKeyPair()

	in, out := net.Pipe()
	tr1 := NewMockTransport(in, pk1, pk2)
	tr2 := NewMockTransport(out, pk2, pk1)

	m1, err := NewManager(&ManagerConfig{SecKey: sk1, DiscoveryClient: client})
	require.NoError(t, err)
	m2, err := NewManager(&ManagerConfig{SecKey: sk2, DiscoveryClient: client})
	require.NoError(t, err)

	errCh := make(chan error)
	var resEntry *Entry
	go func() {
		e, err := settlementResponderHandshake(m2, tr2)
		resEntry = e
		errCh <- err
	}()

	entry, err := settlementInitiatorHandshake(uuid.UUID{}, false)(m1, tr1)
	require.NoError(t, <-errCh)
	require.NoError(t, err)

	require.NotNil(t, resEntry)
	require.NotNil(t, entry)

	assert.Equal(t, entry.ID, resEntry.ID)
	_, err = client.GetTransportByID(context.TODO(), entry.ID)
	require.Error(t, err)
}

func TestSettlementHandshakeExistingTransport(t *testing.T) {
	client := NewDiscoveryMock()

	pk1, sk1 := cipher.GenerateKeyPair()
	pk2, sk2 := cipher.GenerateKeyPair()

	in, out := net.Pipe()
	tr1 := NewMockTransport(in, pk1, pk2)
	tr2 := NewMockTransport(out, pk2, pk1)

	m1, err := NewManager(&ManagerConfig{SecKey: sk1, DiscoveryClient: client})
	require.NoError(t, err)
	m2, err := NewManager(&ManagerConfig{SecKey: sk2, DiscoveryClient: client})
	require.NoError(t, err)

	entry := &Entry{
		ID:     GetTransportUUID(pk1, pk2, ""),
		edges:  SortPubKeys(pk1, pk2),
		Type:   "mock",
		Public: true,
	}

	m1.entries = append(m1.entries, entry)
	m2.entries = append(m2.entries, entry)
	require.NoError(t, client.RegisterTransports(context.TODO(), &SignedEntry{Entry: entry}))
	_, err = client.UpdateStatuses(context.Background(), &Status{ID: entry.ID, IsUp: false})
	require.NoError(t, err)

	errCh := make(chan error)
	var resEntry *Entry
	go func() {
		e, err := settlementResponderHandshake(m2, tr2)
		resEntry = e
		errCh <- err
	}()

	entry, err = settlementInitiatorHandshake(entry.ID, true)(m1, tr1)
	require.NoError(t, <-errCh)
	require.NoError(t, err)

	require.NotNil(t, resEntry)
	require.NotNil(t, entry)

	assert.Equal(t, entry.ID, resEntry.ID)
	dEntry, err := client.GetTransportByID(context.TODO(), entry.ID)
	require.NoError(t, err)

	assert.True(t, dEntry.IsUp)
}

func TestValidateEntry(t *testing.T) {
	pk1, sk1 := cipher.GenerateKeyPair()
	pk2, sk2 := cipher.GenerateKeyPair()
	tr := NewMockTransport(nil, pk1, pk2)

	entry := &Entry{Type: "mock", edges: SortPubKeys(pk2, pk1)}
	tcs := []struct {
		sEntry *SignedEntry
		err    string
	}{
		{
			&SignedEntry{Entry: &Entry{Type: "foo"}},
			"invalid entry type",
		},
		{
			&SignedEntry{Entry: &Entry{Type: "mock", edges: SortPubKeys(pk2, pk1)}},
			"invalid entry edges",
		},
		{
			&SignedEntry{Entry: &Entry{Type: "mock", edges: SortPubKeys(pk2, pk1)}},
			"invalid entry signature",
		},
		{
			&SignedEntry{Entry: entry, Signatures: [2]cipher.Sig{}},
			"invalid entry signature",
		},
		{
			&SignedEntry{Entry: entry, Signatures: [2]cipher.Sig{entry.Signature(sk1)}},
			"Recovered pubkey does not match pubkey",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.err, func(t *testing.T) {
			err := validateEntry(tc.sEntry, tr, pk2)
			require.Error(t, err)
			assert.Equal(t, tc.err, err.Error())
		})
	}

	sEntry := &SignedEntry{Entry: entry, Signatures: [2]cipher.Sig{entry.Signature(sk2)}}
	require.NoError(t, validateEntry(sEntry, tr, pk2))
}
