package app2

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/skywire/pkg/routing"
)

func TestListener_Accept(t *testing.T) {
	lisID := uint16(1)
	localPK, _ := cipher.GenerateKeyPair()
	local := routing.Addr{
		PubKey: localPK,
		Port:   routing.Port(100),
	}

	t.Run("ok", func(t *testing.T) {
		acceptConnID := uint16(1)
		remotePK, _ := cipher.GenerateKeyPair()
		acceptRemote := routing.Addr{
			PubKey: remotePK,
			Port:   routing.Port(100),
		}
		var acceptErr error

		rpc := &MockServerRPCClient{}
		rpc.On("Accept", acceptConnID).Return(acceptConnID, acceptRemote, acceptErr)

		lis := &Listener{
			id:   lisID,
			rpc:  rpc,
			addr: local,
		}

		wantConn := &Conn{
			id:     acceptConnID,
			rpc:    rpc,
			local:  local,
			remote: acceptRemote,
		}

		conn, err := lis.Accept()
		require.NoError(t, err)
		require.Equal(t, conn, wantConn)
	})

	t.Run("accept error", func(t *testing.T) {
		acceptConnID := uint16(0)
		acceptRemote := routing.Addr{}
		acceptErr := errors.New("accept error")

		rpc := &MockServerRPCClient{}
		rpc.On("Accept", lisID).Return(acceptConnID, acceptRemote, acceptErr)

		lis := &Listener{
			id:   lisID,
			rpc:  rpc,
			addr: local,
		}

		conn, err := lis.Accept()
		require.Equal(t, acceptErr, err)
		require.Nil(t, conn)
	})
}

func TestListener_Close(t *testing.T) {
	lisID := uint16(1)
	localPK, _ := cipher.GenerateKeyPair()
	local := routing.Addr{
		PubKey: localPK,
		Port:   routing.Port(100),
	}

	t.Run("ok", func(t *testing.T) {
		var closeErr error

		rpc := &MockServerRPCClient{}
		rpc.On("CloseListener", lisID).Return(closeErr)

		lis := &Listener{
			id:       lisID,
			rpc:      rpc,
			addr:     local,
			freePort: func() {},
		}

		err := lis.Close()
		require.NoError(t, err)
	})

	t.Run("close error", func(t *testing.T) {
		closeErr := errors.New("close error")

		rpc := &MockServerRPCClient{}
		rpc.On("CloseListener", lisID).Return(closeErr)

		lis := &Listener{
			id:       lisID,
			rpc:      rpc,
			addr:     local,
			freePort: func() {},
		}

		err := lis.Close()
		require.Equal(t, closeErr, err)
	})
}
