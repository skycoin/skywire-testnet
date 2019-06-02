package ioutil

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"io"
	"math"
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
	waiters [math.MaxUint16 + 1]chan struct{}
	mx      sync.RWMutex
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

// Wait performs the given action, and waits for given seq to be Done.
func (w *Uint16AckWaiter) Wait(ctx context.Context, done <-chan struct{}, action func(seq Uint16Seq) error) error {
	ackCh := make(chan struct{})
	defer close(ackCh)

	w.mx.Lock()
	seq := w.nextSeq
	w.nextSeq++
	w.waiters[seq] = ackCh
	w.mx.Unlock()

	if err := action(seq); err != nil {
		return err
	}

	select {
	case <-ackCh:
		return nil
	case <-done:
		return io.ErrClosedPipe
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Done finishes given sequence.
func (w *Uint16AckWaiter) Done(seq Uint16Seq) {
	w.mx.RLock()
	ackCh := w.waiters[seq]
	w.mx.RUnlock()

	select {
	case ackCh <- struct{}{}:
	default:
	}
}
