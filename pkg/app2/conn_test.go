package app2

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConn_Read(t *testing.T) {
	connID := uint16(1)

	t.Run("ok", func(t *testing.T) {
		readBuff := make([]byte, 100)
		readN := 20
		readBytes := make([]byte, 100)
		for i := 0; i < readN; i++ {
			readBytes[i] = 2
		}
		var readErr error

		rpc := &MockServerRPCClient{}
		rpc.On("Read", connID, readBuff).Return(readN, readBytes, readErr)

		conn := &Conn{
			id:  connID,
			rpc: rpc,
		}

		n, err := conn.Read(readBuff)
		require.NoError(t, err)
		require.Equal(t, n, readN)
		require.Equal(t, readBuff[:n], readBytes[:n])
	})

	t.Run("read error", func(t *testing.T) {
		readBuff := make([]byte, 100)
		readN := 0
		var readBytes []byte
		readErr := errors.New("read error")

		rpc := &MockServerRPCClient{}
		rpc.On("Read", connID, readBuff).Return(readN, readBytes, readErr)

		conn := &Conn{
			id:  connID,
			rpc: rpc,
		}

		n, err := conn.Read(readBuff)
		require.Equal(t, readErr, err)
		require.Equal(t, readN, n)
	})
}

func TestConn_Write(t *testing.T) {
	connID := uint16(1)

	t.Run("ok", func(t *testing.T) {
		writeBuff := make([]byte, 100)
		writeN := 20
		var writeErr error

		rpc := &MockServerRPCClient{}
		rpc.On("Write", connID, writeBuff).Return(writeN, writeErr)

		conn := &Conn{
			id:  connID,
			rpc: rpc,
		}

		n, err := conn.Write(writeBuff)
		require.NoError(t, err)
		require.Equal(t, writeN, n)
	})

	t.Run("write error", func(t *testing.T) {
		writeBuff := make([]byte, 100)
		writeN := 0
		writeErr := errors.New("write error")

		rpc := &MockServerRPCClient{}
		rpc.On("Write", connID, writeBuff).Return(writeN, writeErr)

		conn := &Conn{
			id:  connID,
			rpc: rpc,
		}

		n, err := conn.Write(writeBuff)
		require.Equal(t, writeErr, err)
		require.Equal(t, writeN, n)
	})
}

func TestConn_Close(t *testing.T) {
	connID := uint16(1)

	t.Run("ok", func(t *testing.T) {
		var closeErr error

		rpc := &MockServerRPCClient{}
		rpc.On("CloseConn", connID).Return(closeErr)

		conn := &Conn{
			id:  connID,
			rpc: rpc,
		}

		err := conn.Close()
		require.NoError(t, err)
	})

	t.Run("close error", func(t *testing.T) {
		closeErr := errors.New("close error")

		rpc := &MockServerRPCClient{}
		rpc.On("CloseConn", connID).Return(closeErr)

		conn := &Conn{
			id:  connID,
			rpc: rpc,
		}

		err := conn.Close()
		require.Equal(t, closeErr, err)
	})
}
