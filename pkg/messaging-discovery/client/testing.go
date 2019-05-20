package client

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/skycoin/skywire/pkg/cipher"
)

// MockClient is an APIClient mock. The mock doesn't reply with the same errors as the
// real client, and it mimics it's functionality not being 100% accurate.
type mockClient struct {
	entriesLock sync.RWMutex
	listLock    sync.RWMutex
	entries     map[string]Entry
	list        []*Entry
}

// NewMock constructs  a new mock APIClient.
func NewMock() APIClient {
	return &mockClient{
		entries: make(map[string]Entry),
		list:    []*Entry{},
	}
}

func (m *mockClient) entry(key string) (Entry, bool) {
	m.entriesLock.RLock()
	defer m.entriesLock.RUnlock()

	e, ok := m.entries[key]
	return e, ok
}

func (m *mockClient) setEntry(key string, entry Entry) {
	m.entriesLock.Lock()
	defer m.entriesLock.Unlock()

	m.entries[key] = entry
}

func (m *mockClient) setServer(entry *Entry) {
	m.listLock.Lock()
	defer m.listLock.Unlock()

	m.list = append(m.list, entry)
}

// Entry returns the mock client static public key associated entry
func (m *mockClient) Entry(ctx context.Context, publicKey cipher.PubKey) (*Entry, error) {
	entry, ok := m.entry(publicKey.Hex())
	if !ok {
		return nil, errors.New(HTTPMessage{ErrKeyNotFound.Error(), http.StatusNotFound}.String())
	}
	res := &Entry{}
	Copy(res, &entry)
	return res, nil
}

// SetEntry sets an entry on the APIClient mock
func (m *mockClient) SetEntry(ctx context.Context, e *Entry) error {
	previousEntry, ok := m.entry(e.Static.Hex())
	if ok {
		err := previousEntry.ValidateIteration(e)
		if err != nil {
			return err
		}
		err = e.VerifySignature()
		if err != nil {
			return err
		}
	}

	m.setEntry(e.Static.Hex(), *e)
	if e.Server != nil {
		m.setServer(e)
	}

	return nil
}

// UpdateEntry updates a previously set entry
func (m *mockClient) UpdateEntry(ctx context.Context, sk cipher.SecKey, e *Entry) error {
	e.Sequence++
	e.Timestamp = time.Now().UnixNano()
	err := e.Sign(sk)
	if err != nil {
		return err
	}

	for {
		err = m.SetEntry(ctx, e)
		if err == nil {
			return nil
		}
		if err != ErrValidationWrongSequence {
			e.Sequence--
			return err
		}
		rE, entryErr := m.Entry(ctx, e.Static)
		if entryErr != nil {
			return err
		}
		if rE.Timestamp > e.Timestamp { // If there is a more up to date entry drop update
			e.Sequence = rE.Sequence
			return nil
		}
		e.Sequence = rE.Sequence + 1
		err := e.Sign(sk)
		if err != nil {
			return err
		}
	}
}

// AvailableServers returns all the servers that the APIClient mock has
func (m *mockClient) AvailableServers(ctx context.Context) ([]*Entry, error) {
	m.listLock.RLock()
	defer m.listLock.RUnlock()
	return m.list, nil
}
