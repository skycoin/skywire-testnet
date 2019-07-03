package transport

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
)

// LogEntry represents a logging entry for a given Transport.
// The entry is updated every time a packet is received or sent.
type LogEntry struct {
	RecvBytes uint64 `json:"recv"` // Total received bytes.
	SentBytes uint64 `json:"sent"` // Total sent bytes.
}

// AddRecv records read.
func (le *LogEntry) AddRecv(n uint64) {
	atomic.AddUint64(&le.RecvBytes, n)
}

// AddSent records write.
func (le *LogEntry) AddSent(n uint64) {
	atomic.AddUint64(&le.SentBytes, n)
}

// MarshalJSON implements json.Marshaller
func (le *LogEntry) MarshalJSON() ([]byte, error) {
	rb := strconv.FormatUint(atomic.LoadUint64(&le.RecvBytes), 10)
	sb := strconv.FormatUint(atomic.LoadUint64(&le.SentBytes), 10)
	return []byte(`{"recv":` + rb + `,"sent":` + sb + `}`), nil
}

// GobEncode implements gob.GobEncoder
func (le *LogEntry) GobEncode() ([]byte, error) {
	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	if err := enc.Encode(le.RecvBytes); err != nil {
		return nil, err
	}
	if err := enc.Encode(le.SentBytes); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// GobDecode implements gob.GobDecoder
func (le *LogEntry) GobDecode(b []byte) error {
	r := bytes.NewReader(b)
	dec := gob.NewDecoder(r)
	var rb uint64
	if err := dec.Decode(&rb); err != nil {
		return err
	}
	var sb uint64
	if err := dec.Decode(&sb); err != nil {
		return err
	}
	atomic.StoreUint64(&le.RecvBytes, rb)
	atomic.StoreUint64(&le.SentBytes, sb)
	return nil
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
