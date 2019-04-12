package appnet

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
		errCh1 <- proto1.ServeJSON(func(f FrameType, _ []byte) (interface{}, error) {
			if f != FrameData {
				return nil, errors.New("unexpected frame")
			}

			return nil, nil
		})
	}()

	errCh2 := make(chan error)
	go func() {
		errCh2 <- proto2.ServeJSON(func(f FrameType, _ []byte) (interface{}, error) {
			if f != FrameCreateLoop {
				return nil, errors.New("unexpected frame")
			}

			return nil, nil
		})
	}()

	require.NoError(t, proto1.CallJSON(FrameCreateLoop, "foo", nil))
	require.NoError(t, proto2.CallJSON(FrameData, "foo", nil))

	err = proto1.CallJSON(FrameData, "foo", nil)
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
		errCh1 <- proto1.ServeJSON(func(f FrameType, _ []byte) (interface{}, error) {
			if f != FrameCreateLoop {
				return nil, errors.New("unexpected frame")
			}

			return nil, proto1.CallJSON(FrameConfirmLoop, "foo", nil)
		})
	}()

	errCh2 := make(chan error)
	go func() {
		errCh2 <- proto2.ServeJSON(func(f FrameType, _ []byte) (interface{}, error) {
			if f != FrameConfirmLoop {
				return nil, errors.New("unexpected frame")
			}

			return nil, nil
		})
	}()

	require.NoError(t, proto2.CallJSON(FrameCreateLoop, "foo", nil))

	require.NoError(t, proto1.Close())
	require.NoError(t, proto2.Close())

	require.NoError(t, <-errCh1)
	require.NoError(t, <-errCh2)
}

func TestNewProtocol(t *testing.T) {

}
