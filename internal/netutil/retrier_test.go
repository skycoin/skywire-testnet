package netutil

import (
	"errors"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestRetrier_Do(t *testing.T) {
	r := NewRetrier(time.Millisecond*100,time.Millisecond*500,2)
	c := 0
	threshold := 2
	f := func() error {
		c++
		if c >= threshold {
			return nil
		}

		return errors.New("foo")
	}

	t.Run("should retry", func(t *testing.T) {
		c = 0

		err := r.Do(f)
		require.NoError(t, err)
	})

	t.Run("if retry reaches threshold should error", func(t *testing.T){
		c = 0
		threshold = 4
		defer func() {
			threshold = 2
		}()

		err := r.Do(f)
		require.Error(t, err)
	})

	t.Run("if function times out should error", func(t *testing.T) {
		c = 0
		slowF := func() error {
			time.Sleep(time.Second)
			return nil
		}

		err := r.Do(slowF)
		require.Error(t, err)
	})

	t.Run("should return whitelisted errors if any instead of retry", func(t *testing.T){
		bar := errors.New("bar")
		wR := NewRetrier(50*time.Millisecond, time.Second, 2).WithErrWhitelist(bar)
		barF := func() error {
			return bar
		}

		err := wR.Do(barF)
		require.EqualError(t, err, bar.Error())
	})
}