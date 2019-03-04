package app

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProtocol(t *testing.T) {
	rw1, rw2, err := OpenPipeConn()
	require.NoError(t, err)
	proto1 := NewProtocol(rw1)
	proto2 := NewProtocol(rw2)

	errCh1 := make(chan error)
	go func() {
		errCh1 <- proto1.Serve(func(f Frame, _ []byte) (interface{}, error) {
			if f != FrameSend {
				return nil, errors.New("unexpected frame")
			}

			return nil, nil
		})
	}()

	errCh2 := make(chan error)
	go func() {
		errCh2 <- proto2.Serve(func(f Frame, _ []byte) (interface{}, error) {
			if f != FrameCreateLoop {
				return nil, errors.New("unexpected frame")
			}

			return nil, nil
		})
	}()

	errCh3 := make(chan error)
	go func() {
		errCh3 <- proto1.Send(FrameCreateLoop, "foo", nil)
	}()

	errCh4 := make(chan error)
	go func() {
		errCh4 <- proto2.Send(FrameSend, "foo", nil)
	}()

	errCh5 := make(chan error)
	go func() {
		errCh5 <- proto1.Send(FrameSend, "foo", nil)
	}()

	require.NoError(t, <-errCh3)
	require.NoError(t, <-errCh4)
	err = <-errCh5
	require.Error(t, err)
	assert.Equal(t, "unexpected frame", err.Error())

	require.NoError(t, proto1.Close())
	require.NoError(t, proto2.Close())

	require.NoError(t, <-errCh1)
	require.NoError(t, <-errCh2)
}

func TestProtocolParallel(t *testing.T) {
	rw1, rw2, err := OpenPipeConn()
	require.NoError(t, err)
	proto1 := NewProtocol(rw1)
	proto2 := NewProtocol(rw2)

	errCh1 := make(chan error)
	go func() {
		errCh1 <- proto1.Serve(func(f Frame, _ []byte) (interface{}, error) {
			if f != FrameCreateLoop {
				return nil, errors.New("unexpected frame")
			}

			return nil, proto1.Send(FrameConfirmLoop, "foo", nil)
		})
	}()

	errCh2 := make(chan error)
	go func() {
		errCh2 <- proto2.Serve(func(f Frame, _ []byte) (interface{}, error) {
			if f != FrameConfirmLoop {
				return nil, errors.New("unexpected frame")
			}

			return nil, nil
		})
	}()

	require.NoError(t, proto2.Send(FrameCreateLoop, "foo", nil))

	require.NoError(t, proto1.Close())
	require.NoError(t, proto2.Close())

	require.NoError(t, <-errCh1)
	require.NoError(t, <-errCh2)
}
