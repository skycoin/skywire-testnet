package ioutil_test

import (
	"context"
	"github.com/skycoin/skywire/internal/ioutil"
	"sync"
	"testing"
)

func TestUint16AckWaiter_Wait(t *testing.T) {

	// Ensure that no race conditions occurs when
	// each concurrent call to 'Uint16AckWaiter.Wait()' is met with
	// multiple concurrent calls to 'Uint16AckWaiter.Done()' with the same seq.
	t.Run("ensure_no_race_conditions", func(*testing.T) {
		w := new(ioutil.Uint16AckWaiter)
		defer w.StopAll()

		seqChan := make(chan ioutil.Uint16Seq)
		defer close(seqChan)

		wg := new(sync.WaitGroup)

		for i := 0; i < 64; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = w.Wait(context.TODO(), func(seq ioutil.Uint16Seq) error { //nolint:errcheck,unparam
					seqChan <- seq
					return nil
				})
			}()

			seq := <-seqChan
			for j := 0; j <= i; j++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					w.Done(seq)
				}()
			}
		}

		wg.Wait()
	})
}
