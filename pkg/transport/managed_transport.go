package transport

import (
	"math/big"
	"sync"

	"github.com/google/uuid"
)

// ManagedTransport is a wrapper transport. It stores status and ID of
// the Transport and can notify about network errors.
type ManagedTransport struct {
	Transport
	ID       uuid.UUID
	Public   bool
	LogEntry *LogEntry

	doneChan chan struct{}
	errChan  chan error
	mu       sync.RWMutex

	readLogChan  chan int
	writeLogChan chan int
}

func newManagedTransport(id uuid.UUID, tr Transport, public bool) *ManagedTransport {
	return &ManagedTransport{
		ID:           id,
		Transport:    tr,
		Public:       public,
		doneChan:     make(chan struct{}),
		errChan:      make(chan error),
		readLogChan:  make(chan int),
		writeLogChan: make(chan int),
		LogEntry:     &LogEntry{new(big.Int), new(big.Int)},
	}
}

// Read reads using underlying
func (tr *ManagedTransport) Read(p []byte) (n int, err error) {
	tr.mu.RLock()
	n, err = tr.Transport.Read(p)
	tr.mu.RUnlock()
	if err == nil {
		select {
		case <-tr.doneChan:
			return
		case tr.readLogChan <- n:
		}

		return
	}

	select {
	case <-tr.doneChan:
		return
	case tr.errChan <- err:
	}

	return
}

// Write writes to an underlying
func (tr *ManagedTransport) Write(p []byte) (n int, err error) {
	tr.mu.RLock()
	n, err = tr.Transport.Write(p)
	tr.mu.RUnlock()
	if err == nil {
		select {
		case <-tr.doneChan:
			return
		case tr.writeLogChan <- n:
		}

		return
	}

	select {
	case <-tr.doneChan:
		return
	case tr.errChan <- err:
	}

	return
}

// Close closes underlying
func (tr *ManagedTransport) Close() error {
	tr.mu.Lock()
	err := tr.Transport.Close()

	select {
	case <-tr.doneChan:
	default:
		tr.doneChan <- struct{}{}
		//close(tr.doneChan)
	}
	tr.mu.Unlock()

	return err
}

func (tr *ManagedTransport) updateTransport(newTr Transport) {
	tr.mu.Lock()
	tr.Transport = newTr
	tr.mu.Unlock()
}
