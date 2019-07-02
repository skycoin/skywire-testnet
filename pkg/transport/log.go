package transport

import (
	"encoding/json"
	"errors"
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
	RecvBytes big.Int `json:"recv"` // Total received bytes.
	SentBytes big.Int `json:"sent"` // Total sent bytes.
	rMx       sync.Mutex
	sMx       sync.Mutex
}

// AddRecv records read.
func (le *LogEntry) AddRecv(n int64) {
	le.rMx.Lock()
	le.RecvBytes.Add(&le.RecvBytes, big.NewInt(n))
	le.rMx.Unlock()
}

// AddSent records write.
func (le *LogEntry) AddSent(n int64) {
	le.sMx.Lock()
	le.SentBytes.Add(&le.SentBytes, big.NewInt(n))
	le.sMx.Unlock()
}

// MarshalJSON implements json.Marshaller
func (le *LogEntry) MarshalJSON() ([]byte, error) {
	le.rMx.Lock()
	recv := le.RecvBytes.String()
	le.rMx.Unlock()

	le.sMx.Lock()
	sent := le.SentBytes.String()
	le.sMx.Unlock()

	data := `{"recv":` + recv + `,"sent":` + sent + `}`
	return []byte(data), nil
}

// GobEncode implements gob.GobEncoder
func (le *LogEntry) GobEncode() ([]byte, error) {
	le.rMx.Lock()
	rb, err := le.RecvBytes.GobEncode()
	le.rMx.Unlock()
	if err != nil {
		return nil, err
	}
	le.sMx.Lock()
	sb, err := le.SentBytes.GobEncode()
	le.sMx.Unlock()
	if err != nil {
		return nil, err
	}
	return append(rb, sb...), err
}

// GobDecode implements gob.GobDecoder
func (le *LogEntry) GobDecode(b []byte) error {
	le.rMx.Lock()
	err := le.RecvBytes.GobDecode(b)
	le.rMx.Unlock()
	if err != nil {
		return err
	}
	le.sMx.Lock()
	err = le.SentBytes.GobDecode(b)
	le.sMx.Unlock()
	return err
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
		entries: make(map[uuid.UUID]*LogEntry),
	}
}

func (tls *inMemoryTransportLogStore) Entry(id uuid.UUID) (*LogEntry, error) {
	tls.mu.Lock()
	entry, ok := tls.entries[id]
	tls.mu.Unlock()
	if !ok {
		return entry, errors.New("not found")
	}

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
func FileTransportLogStore(dir string) (LogStore, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	return &fileTransportLogStore{dir}, nil
}

func (tls *fileTransportLogStore) Entry(id uuid.UUID) (*LogEntry, error) {
	f, err := os.Open(filepath.Join(tls.dir, fmt.Sprintf("%s.log", id)))
	if err != nil {
		return nil, fmt.Errorf("open: %s", err)
	}
	defer func() { _ = f.Close() }() //nolint:errcheck

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
	defer func() { _ = f.Close() }() //nolint:errcheck

	if err := json.NewEncoder(f).Encode(entry); err != nil {
		return fmt.Errorf("json: %s", err)
	}

	return nil
}
