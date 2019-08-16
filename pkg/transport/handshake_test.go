package transport

import (
	"context"
	"fmt"
	"net"
	"testing"

	"github.com/skycoin/dmsg/cipher"
	"github.com/stretchr/testify/require"
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

	m1, err1 := NewManager(&ManagerConfig{PubKey: pk1, SecKey: sk1, DiscoveryClient: client}, nil)
	m2, err2 := NewManager(&ManagerConfig{PubKey: pk2, SecKey: sk2, DiscoveryClient: client}, nil)

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

func TestSettlementHS_Do(t *testing.T) {
	t.Run("no_error", func(t *testing.T) {
		mockEnv := newHsMockEnv()
		require.NoError(t, mockEnv.err1)
		require.NoError(t, mockEnv.err2)

		errCh := make(chan error, 1)
		go func() {
			err := MakeSettlementHS(false).Do(context.TODO(), mockEnv.client, mockEnv.tr2, mockEnv.sk2)
			errCh <- err
			close(errCh)
		}()
		require.NoError(t, MakeSettlementHS(true)(context.TODO(), mockEnv.client, mockEnv.tr1, mockEnv.sk1))
		require.NoError(t, <-errCh)

		expEntry := makeEntry(mockEnv.pk1, mockEnv.pk2, "mock")
		entry, err := mockEnv.client.GetTransportByID(context.TODO(), expEntry.ID)
		require.NoError(t, err)
		require.Equal(t, expEntry, *entry.Entry)
	})
}
