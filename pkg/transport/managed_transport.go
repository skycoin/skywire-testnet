package transport

import (
	"context"
	"sync"
	"time"
)

const logWriteInterval = time.Second * 3

// ManagedTransport is a wrapper transport. It stores status and ID of
// the Transport and can notify about network errors.
type ManagedTransport struct {
	Transport
	Entry    Entry
	Accepted bool
	Setup    bool
	LogEntry *LogEntry

	done   chan struct{}
	update chan error
	mu     sync.RWMutex
	once   sync.Once
}

func newManagedTransport(tr Transport, entry Entry, accepted bool) *ManagedTransport {
	return &ManagedTransport{
		Transport: tr,
		Entry:     entry,
		Accepted:  accepted,
		done:      make(chan struct{}),
		update:    make(chan error, 16),
		LogEntry:  new(LogEntry),
	}
}

// Read reads using underlying transport.
func (tr *ManagedTransport) Read(p []byte) (n int, err error) {
	tr.mu.RLock()
	n, err = tr.Transport.Read(p)
	if n > 0 {
		tr.LogEntry.AddRecv(uint64(n))
	}
	if !tr.isClosing() {
		select {
		case tr.update <- err:
		default:
		}
	}
	tr.mu.RUnlock()
	return
}

// Write writes to an underlying transport.
func (tr *ManagedTransport) Write(p []byte) (n int, err error) {

	tr.mu.RLock()
	n, err = tr.Transport.Write(p)
	if n > 0 {
		tr.LogEntry.AddSent(uint64(n))
	}
	if !tr.isClosing() {
		select {
		case tr.update <- err:
		default:
		}
	}
	tr.mu.RUnlock()
	return
}

func (tr *ManagedTransport) killWorker() {
	tr.once.Do(func() {
		close(tr.done)
	})
}

func (tr *ManagedTransport) killUpdate() {
	tr.mu.Lock()
	close(tr.update)
	tr.update = nil
	tr.mu.Unlock()
}

// Close closes underlying transport and kills worker.
func (tr *ManagedTransport) Close() error {
	if tr == nil {
		return nil
	}
	tr.killWorker()
	return tr.Transport.Close()
}

func (tr *ManagedTransport) isClosing() bool {
	select {
	case <-tr.done:
		return true
	default:
		return false
	}
}

func (tr *ManagedTransport) updateTransport(ctx context.Context, newTr Transport, dc DiscoveryClient) error {
	tr.mu.Lock()
	tr.Transport = newTr
	_, err := dc.UpdateStatuses(ctx, &Status{ID: tr.Entry.ID, IsUp: true, Updated: time.Now().UnixNano()})
	tr.mu.Unlock()
	return err
}
