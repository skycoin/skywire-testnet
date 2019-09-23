package app2

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIDManager_NextID(t *testing.T) {
	t.Run("simple call", func(t *testing.T) {
		m := newIDManager()

		nextKey, err := m.nextKey()
		require.NoError(t, err)
		v, ok := m.values[*nextKey]
		require.True(t, ok)
		require.Nil(t, v)
		require.Equal(t, *nextKey, uint16(1))
		require.Equal(t, *nextKey, m.lstKey)

		nextKey, err = m.nextKey()
		require.NoError(t, err)
		v, ok = m.values[*nextKey]
		require.True(t, ok)
		require.Nil(t, v)
		require.Equal(t, *nextKey, uint16(2))
		require.Equal(t, *nextKey, m.lstKey)
	})

	t.Run("call on full manager", func(t *testing.T) {
		m := newIDManager()
		for i := uint16(0); i < math.MaxUint16; i++ {
			m.values[i] = nil
		}
		m.values[math.MaxUint16] = nil

		_, err := m.nextKey()
		require.Error(t, err)
	})

	t.Run("concurrent run", func(t *testing.T) {
		m := newIDManager()

		valsToReserve := 10000

		errs := make(chan error)
		for i := 0; i < valsToReserve; i++ {
			go func() {
				_, err := m.nextKey()
				errs <- err
			}()
		}

		for i := 0; i < valsToReserve; i++ {
			require.NoError(t, <-errs)
		}
		close(errs)

		require.Equal(t, m.lstKey, uint16(valsToReserve))
		for i := uint16(1); i < uint16(valsToReserve); i++ {
			v, ok := m.values[i]
			require.True(t, ok)
			require.Nil(t, v)
		}
	})
}

func TestIDManager_Pop(t *testing.T) {
	t.Run("simple call", func(t *testing.T) {
		m := newIDManager()

		v := "value"

		m.values[1] = v

		gotV, err := m.pop(1)
		require.NoError(t, err)
		require.NotNil(t, gotV)
		require.Equal(t, gotV, v)

		_, ok := m.values[1]
		require.False(t, ok)
	})

	t.Run("no value", func(t *testing.T) {
		m := newIDManager()

		_, err := m.pop(1)
		require.Error(t, err)
	})

	t.Run("value not set", func(t *testing.T) {
		m := newIDManager()

		m.values[1] = nil

		_, err := m.pop(1)
		require.Error(t, err)
	})

	t.Run("concurrent run", func(t *testing.T) {
		m := newIDManager()

		m.values[1] = "value"

		concurrency := 1000
		errs := make(chan error, concurrency)
		for i := uint16(0); i < uint16(concurrency); i++ {
			go func() {
				_, err := m.pop(1)
				errs <- err
			}()
		}

		errsCount := 0
		for i := 0; i < concurrency; i++ {
			err := <-errs
			if err != nil {
				errsCount++
			}
		}
		close(errs)
		require.Equal(t, errsCount, concurrency-1)

		_, ok := m.values[1]
		require.False(t, ok)
	})
}

func TestIDManager_Set(t *testing.T) {
	t.Run("simple call", func(t *testing.T) {
		m := newIDManager()

		nextKey, err := m.nextKey()
		require.NoError(t, err)

		v := "value"

		err = m.set(*nextKey, v)
		require.NoError(t, err)
		gotV, ok := m.values[*nextKey]
		require.True(t, ok)
		require.Equal(t, gotV, v)
	})

	t.Run("key is not reserved", func(t *testing.T) {
		m := newIDManager()

		err := m.set(1, "value")
		require.Error(t, err)

		_, ok := m.values[1]
		require.False(t, ok)
	})

	t.Run("value already exists", func(t *testing.T) {
		m := newIDManager()

		v := "value"

		m.values[1] = v

		err := m.set(1, "value2")
		require.Error(t, err)
		gotV, ok := m.values[1]
		require.True(t, ok)
		require.Equal(t, gotV, v)
	})

	t.Run("concurrent run", func(t *testing.T) {
		m := newIDManager()

		concurrency := 1000

		nextKeyPtr, err := m.nextKey()
		require.NoError(t, err)

		nextKey := *nextKeyPtr

		errs := make(chan error)
		setV := make(chan int)
		for i := 0; i < concurrency; i++ {
			go func(v int) {
				err := m.set(nextKey, v)
				errs <- err
				if err == nil {
					setV <- v
				}
			}(i)
		}

		errsCount := 0
		for i := 0; i < concurrency; i++ {
			err := <-errs
			if err != nil {
				errsCount++
			}
		}
		close(errs)

		v := <-setV
		close(setV)

		gotV, ok := m.values[nextKey]
		require.True(t, ok)
		require.Equal(t, gotV, v)
	})
}

func TestIDManager_Get(t *testing.T) {
	prepManagerWithVal := func(v interface{}) (*idManager, uint16) {
		m := newIDManager()

		nextKey, err := m.nextKey()
		require.NoError(t, err)

		err = m.set(*nextKey, v)
		require.NoError(t, err)

		return m, *nextKey
	}

	t.Run("simple call", func(t *testing.T) {
		v := "value"

		m, key := prepManagerWithVal(v)

		gotV, ok := m.get(key)
		require.True(t, ok)
		require.Equal(t, gotV, v)

		_, ok = m.get(100)
		require.False(t, ok)
	})

	t.Run("concurrent run", func(t *testing.T) {
		v := "value"

		m, key := prepManagerWithVal(v)

		concurrency := 1000
		type getRes struct {
			v  interface{}
			ok bool
		}
		res := make(chan getRes)
		for i := 0; i < concurrency; i++ {
			go func() {
				val, ok := m.get(key)
				res <- getRes{
					v:  val,
					ok: ok,
				}
			}()
		}

		for i := 0; i < concurrency; i++ {
			r := <-res
			require.True(t, r.ok)
			require.Equal(t, r.v, v)
		}
		close(res)
	})
}
