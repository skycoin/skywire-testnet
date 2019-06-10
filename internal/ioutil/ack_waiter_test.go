package ioutil

import (
	"context"
	"testing"
)

// Ensure that no race conditions occurs.
func TestUint16AckWaiter_Wait(t *testing.T) {
	w := new(Uint16AckWaiter)

	seqChan := make(chan Uint16Seq)
	defer close(seqChan)
	for i := 0; i < 64; i++ {
		go w.Wait(context.TODO(), func(seq Uint16Seq) error { //nolint:errcheck,unparam
			seqChan <- seq
			return nil
		})
		seq := <-seqChan
		for j := 0; j < i; j++ {
			go w.Done(seq)
		}
	}
}
