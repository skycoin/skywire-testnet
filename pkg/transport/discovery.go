package transport

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/skycoin/dmsg/cipher"
)

// DiscoveryClient performs Transport discovery operations.
type DiscoveryClient interface {
	RegisterTransports(ctx context.Context, entries ...*SignedEntry) error
	GetTransportByID(ctx context.Context, id uuid.UUID) (*EntryWithStatus, error)
	GetTransportsByEdge(ctx context.Context, pk cipher.PubKey) ([]*EntryWithStatus, error)
	UpdateStatuses(ctx context.Context, statuses ...*Status) ([]*EntryWithStatus, error)
}

type mockDiscoveryClient struct {
	sync.Mutex
	entries map[uuid.UUID]EntryWithStatus
}

// NewDiscoveryMock construct a new mock transport discovery client.
func NewDiscoveryMock() DiscoveryClient {
	return &mockDiscoveryClient{entries: map[uuid.UUID]EntryWithStatus{}}
}

func (td *mockDiscoveryClient) RegisterTransports(ctx context.Context, entries ...*SignedEntry) error {
	td.Lock()
	for _, entry := range entries {
		entryWithStatus := &EntryWithStatus{
			Entry:      entry.Entry,
			IsUp:       true,
			Registered: time.Now().Unix(),
			Statuses:   [2]bool{true, true},
		}
		td.entries[entry.Entry.ID] = *entryWithStatus
		entry.Registered = entryWithStatus.Registered
	}
	td.Unlock()

	return nil
}

func (td *mockDiscoveryClient) GetTransportByID(ctx context.Context, id uuid.UUID) (*EntryWithStatus, error) {
	td.Lock()
	entry, ok := td.entries[id]
	td.Unlock()

	if !ok {
		return nil, errors.New("transport not found")
	}

	return &EntryWithStatus{
		Entry:      entry.Entry,
		IsUp:       entry.IsUp,
		Registered: entry.Registered,
		Statuses:   entry.Statuses,
	}, nil
}

func (td *mockDiscoveryClient) GetTransportsByEdge(ctx context.Context, pk cipher.PubKey) ([]*EntryWithStatus, error) {
	td.Lock()
	res := make([]*EntryWithStatus, 0)
	for _, entry := range td.entries {
		if entry.Entry.LocalPK() == pk || entry.Entry.RemotePK() == pk {
			e := &EntryWithStatus{}
			*e = entry
			res = append(res, e)
		}
	}
	td.Unlock()

	if len(res) == 0 {
		return nil, nil
	}

	return res, nil
}

func (td *mockDiscoveryClient) UpdateStatuses(ctx context.Context, statuses ...*Status) ([]*EntryWithStatus, error) {
	res := make([]*EntryWithStatus, 0)
	for _, status := range statuses {
		entry, err := td.GetTransportByID(ctx, status.ID)
		if err != nil {
			return nil, err
		}

		td.Lock()
		entry.IsUp = status.IsUp
		td.entries[status.ID] = *entry
		td.Unlock()
	}

	return res, nil
}
