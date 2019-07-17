package ioutil

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"io"
	"sync"
)

// Uint16Seq is part of the acknowledgement-waiting logic.
type Uint16Seq uint16

// DecodeUint16Seq decodes a slice to Uint16Seq.
func DecodeUint16Seq(b []byte) Uint16Seq {
	if len(b) < 2 {
		return 0
	}
	return Uint16Seq(binary.BigEndian.Uint16(b[:2]))
}

// Encode encodes the Uint16Seq to a 2-byte slice.
func (s Uint16Seq) Encode() []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(s))
	return b
}

// Uint16AckWaiter implements acknowledgement-waiting logic (with uint16 sequences).
type Uint16AckWaiter struct {
	nextSeq Uint16Seq
	waiters map[Uint16Seq]chan struct{}
	mx      sync.RWMutex
}

// NewUint16AckWaiter creates a new Uint16AckWaiter
func NewUint16AckWaiter() Uint16AckWaiter {
	return Uint16AckWaiter{
		waiters: make(map[Uint16Seq]chan struct{}),
	}
}

// RandSeq should only be run once on startup. It is not thread-safe.
func (w *Uint16AckWaiter) RandSeq() error {
	b := make([]byte, 2)
	if _, err := rand.Read(b); err != nil {
		return err
	}
	w.nextSeq = DecodeUint16Seq(b)
	return nil
}

func (w *Uint16AckWaiter) stopWaiter(seq Uint16Seq) {
	if waiter := w.waiters[seq]; waiter != nil {
		close(waiter)
		w.waiters[seq] = nil
	}
}

// StopAll stops all active waiters.
func (w *Uint16AckWaiter) StopAll() {
	w.mx.Lock()
	for seq := range w.waiters {
		w.stopWaiter(Uint16Seq(seq))
	}
	w.mx.Unlock()
}

// Wait performs the given action, and waits for given seq to be Done.
func (w *Uint16AckWaiter) Wait(ctx context.Context, action func(seq Uint16Seq) error) (err error) {
	ackCh := make(chan struct{}, 1)

	w.mx.Lock()
	seq := w.nextSeq
	w.nextSeq++
	w.waiters[seq] = ackCh
	w.mx.Unlock()

	if err = action(seq); err != nil {
		return err
	}

	select {
	case _, ok := <-ackCh:
		if !ok {
			// waiter stopped manually.
			err = io.ErrClosedPipe
		}
	case <-ctx.Done():
		err = ctx.Err()
	}

	w.mx.Lock()
	w.stopWaiter(seq)
	w.mx.Unlock()
	return err
}

// Done finishes given sequence.
func (w *Uint16AckWaiter) Done(seq Uint16Seq) {
	w.mx.RLock()
	select {
	case w.waiters[seq] <- struct{}{}:
	default:
	}
	w.mx.RUnlock()
}
