package app2

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestManager_NextID(t *testing.T) {
	t.Run("simple call", func(t *testing.T) {
		m := newManager()

		nextKey, err := m.nextID()
		require.NoError(t, err)
		require.Equal(t, *nextKey, uint16(1))
		require.Equal(t, *nextKey, m.lstKey)

		nextKey, err = m.nextID()
		require.NoError(t, err)
		require.Equal(t, *nextKey, uint16(2))
		require.Equal(t, *nextKey, m.lstKey)
	})

	t.Run("call on full manager", func(t *testing.T) {
		m := newManager()
		for i := uint16(0); i < math.MaxUint16; i++ {
			m.values[i] = nil
		}
		m.values[math.MaxUint16] = nil

		_, err := m.nextID()
		require.Error(t, err)
	})

	t.Run("concurrent run", func(t *testing.T) {
		m := newManager()

		valsToReserve := 10000

		errs := make(chan error)
		for i := 0; i < valsToReserve; i++ {
			go func() {
				_, err := m.nextID()
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

func TestManager_GetAndRemove(t *testing.T) {
	t.Run("simple call", func(t *testing.T) {
		m := newManager()

		v := "value"

		m.values[1] = v

		gotV, err := m.getAndRemove(1)
		require.NoError(t, err)
		require.NotNil(t, gotV)
		require.Equal(t, gotV, v)

		_, ok := m.values[1]
		require.False(t, ok)
	})

	t.Run("no value", func(t *testing.T) {
		m := newManager()

		_, err := m.getAndRemove(1)
		require.Error(t, err)
	})

	t.Run("value not set", func(t *testing.T) {
		m := newManager()

		m.values[1] = nil

		_, err := m.getAndRemove(1)
		require.Error(t, err)
	})

	t.Run("concurrent run", func(t *testing.T) {
		// TODO(Darkren): finish
	})
}
