package transport

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
)

// LogEntry represents a logging entry for a given Transport.
// The entry is updated every time a packet is received or sent.
type LogEntry struct {
	ReceivedBytes *big.Int `json:"received"` // Total received bytes.
	SentBytes     *big.Int `json:"sent"`     // Total sent bytes.
}

// LogStore stores transport log entries.
type LogStore interface {
	Entry(id uuid.UUID) (*LogEntry, error)
	Record(id uuid.UUID, entry *LogEntry) error
}

type inMemoryTransportLogStore struct {
	entries map[uuid.UUID]*LogEntry
	mu      sync.Mutex
}

// InMemoryTransportLogStore implements in-memory TransportLogStore.
func InMemoryTransportLogStore() LogStore {
	return &inMemoryTransportLogStore{
		entries: map[uuid.UUID]*LogEntry{},
	}
}

func (tls *inMemoryTransportLogStore) Entry(id uuid.UUID) (*LogEntry, error) {
	tls.mu.Lock()
	entry := tls.entries[id]
	tls.mu.Unlock()

	return entry, nil
}

func (tls *inMemoryTransportLogStore) Record(id uuid.UUID, entry *LogEntry) error {
	tls.mu.Lock()
	if tls.entries == nil {
		tls.entries = make(map[uuid.UUID]*LogEntry)
	}
	tls.entries[id] = entry
	tls.mu.Unlock()
	return nil
}

type fileTransportLogStore struct {
	dir string
}

// FileTransportLogStore implements file TransportLogStore.
func FileTransportLogStore(dir string) LogStore {
	return &fileTransportLogStore{dir}
}

func (tls *fileTransportLogStore) Entry(id uuid.UUID) (*LogEntry, error) {
	f, err := os.Open(filepath.Join(tls.dir, fmt.Sprintf("%s.log", id)))
	if err != nil {
		return nil, fmt.Errorf("open: %s", err)
	}

	entry := &LogEntry{}
	if err := json.NewDecoder(f).Decode(entry); err != nil {
		return nil, fmt.Errorf("json: %s", err)
	}

	return entry, nil
}

func (tls *fileTransportLogStore) Record(id uuid.UUID, entry *LogEntry) error {
	f, err := os.OpenFile(filepath.Join(tls.dir, fmt.Sprintf("%s.log", id)), os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return fmt.Errorf("open: %s", err)
	}

	if err := json.NewEncoder(f).Encode(entry); err != nil {
		return fmt.Errorf("json: %s", err)
	}

	return nil
}
