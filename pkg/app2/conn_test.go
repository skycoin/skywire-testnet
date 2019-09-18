package app2

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConn_Read(t *testing.T) {
	connID := uint16(1)

	tt := []struct {
		name      string
		readBuff  []byte
		readN     int
		readBytes []byte
		readErr   error
		wantBuff  []byte
	}{
		{
			name:      "ok",
			readBuff:  make([]byte, 10),
			readN:     2,
			readBytes: []byte{1, 1, 0, 0, 0, 0, 0, 0, 0, 0},
			wantBuff:  []byte{1, 1, 0, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			name:     "read error",
			readBuff: make([]byte, 10),
			readErr:  errors.New("read error"),
			wantBuff: make([]byte, 10),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			rpc := &MockServerRPCClient{}
			rpc.On("Read", connID, tc.readBuff).Return(tc.readN, tc.readBytes, tc.readErr)

			conn := &Conn{
				id:  connID,
				rpc: rpc,
			}

			n, err := conn.Read(tc.readBuff)
			require.Equal(t, tc.readErr, err)
			require.Equal(t, tc.readN, n)
			require.Equal(t, tc.wantBuff, tc.readBuff)
		})
	}
}

func TestConn_Write(t *testing.T) {
	connID := uint16(1)

	tt := []struct {
		name      string
		writeBuff []byte
		writeN    int
		writeErr  error
	}{
		{
			name:      "ok",
			writeBuff: make([]byte, 10),
			writeN:    2,
		},
		{
			name:      "write error",
			writeBuff: make([]byte, 10),
			writeErr:  errors.New("write error"),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			rpc := &MockServerRPCClient{}
			rpc.On("Write", connID, tc.writeBuff).Return(tc.writeN, tc.writeErr)

			conn := &Conn{
				id:  connID,
				rpc: rpc,
			}

			n, err := conn.Write(tc.writeBuff)
			require.Equal(t, tc.writeErr, err)
			require.Equal(t, tc.writeN, n)
		})
	}
}

func TestConn_Close(t *testing.T) {
	connID := uint16(1)

	tt := []struct {
		name     string
		closeErr error
	}{
		{
			name: "ok",
		},
		{
			name:     "close error",
			closeErr: errors.New("close error"),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			rpc := &MockServerRPCClient{}
			rpc.On("CloseConn", connID).Return(tc.closeErr)

			conn := &Conn{
				id:  connID,
				rpc: rpc,
			}

			err := conn.Close()
			require.Equal(t, tc.closeErr, err)
		})
	}
}
