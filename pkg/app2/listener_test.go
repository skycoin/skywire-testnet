package app2

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/skycoin/src/util/logging"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/pkg/app2/network"
	"github.com/skycoin/skywire/pkg/routing"
)

func TestListener_Accept(t *testing.T) {
	l := logging.MustGetLogger("app2_listener")

	lisID := uint16(1)
	localPK, _ := cipher.GenerateKeyPair()
	local := network.Addr{
		Net:    network.TypeDMSG,
		PubKey: localPK,
		Port:   routing.Port(100),
	}

	t.Run("ok", func(t *testing.T) {
		acceptConnID := uint16(1)
		acceptRemotePK, _ := cipher.GenerateKeyPair()
		acceptRemote := network.Addr{
			Net:    network.TypeDMSG,
			PubKey: acceptRemotePK,
			Port:   routing.Port(100),
		}
		var acceptErr error

		rpc := &MockRPCClient{}
		rpc.On("Accept", acceptConnID).Return(acceptConnID, acceptRemote, acceptErr)

		lis := &Listener{
			id:   lisID,
			rpc:  rpc,
			addr: local,
			cm:   newIDManager(),
		}

		wantConn := &Conn{
			id:     acceptConnID,
			rpc:    rpc,
			local:  local,
			remote: acceptRemote,
		}

		conn, err := lis.Accept()
		require.NoError(t, err)

		appConn, ok := conn.(*Conn)
		require.True(t, ok)
		require.Equal(t, wantConn.id, appConn.id)
		require.Equal(t, wantConn.rpc, appConn.rpc)
		require.Equal(t, wantConn.local, appConn.local)
		require.Equal(t, wantConn.remote, appConn.remote)
		require.NotNil(t, appConn.freeConn)

		connIfc, ok := lis.cm.values[acceptConnID]
		require.True(t, ok)

		appConn, ok = connIfc.(*Conn)
		require.True(t, ok)
		require.NotNil(t, appConn.freeConn)
	})

	t.Run("conn already exists", func(t *testing.T) {
		acceptConnID := uint16(1)
		acceptRemotePK, _ := cipher.GenerateKeyPair()
		acceptRemote := network.Addr{
			Net:    network.TypeDMSG,
			PubKey: acceptRemotePK,
			Port:   routing.Port(100),
		}
		var acceptErr error

		var closeErr error

		rpc := &MockRPCClient{}
		rpc.On("Accept", acceptConnID).Return(acceptConnID, acceptRemote, acceptErr)
		rpc.On("CloseConn", acceptConnID).Return(closeErr)

		lis := &Listener{
			id:   lisID,
			rpc:  rpc,
			addr: local,
			cm:   newIDManager(),
		}

		lis.cm.values[acceptConnID] = nil

		conn, err := lis.Accept()
		require.Equal(t, err, errValueAlreadyExists)
		require.Nil(t, conn)
	})

	t.Run("conn already exists, conn closed with error", func(t *testing.T) {
		acceptConnID := uint16(1)
		acceptRemotePK, _ := cipher.GenerateKeyPair()
		acceptRemote := network.Addr{
			Net:    network.TypeDMSG,
			PubKey: acceptRemotePK,
			Port:   routing.Port(100),
		}
		var acceptErr error

		closeErr := errors.New("close error")

		rpc := &MockRPCClient{}
		rpc.On("Accept", acceptConnID).Return(acceptConnID, acceptRemote, acceptErr)
		rpc.On("CloseConn", acceptConnID).Return(closeErr)

		lis := &Listener{
			log:  l,
			id:   lisID,
			rpc:  rpc,
			addr: local,
			cm:   newIDManager(),
		}

		lis.cm.values[acceptConnID] = nil

		conn, err := lis.Accept()
		require.Equal(t, err, errValueAlreadyExists)
		require.Nil(t, conn)
	})

	t.Run("accept error", func(t *testing.T) {
		acceptConnID := uint16(0)
		acceptRemote := network.Addr{}
		acceptErr := errors.New("accept error")

		rpc := &MockRPCClient{}
		rpc.On("Accept", lisID).Return(acceptConnID, acceptRemote, acceptErr)

		lis := &Listener{
			id:   lisID,
			rpc:  rpc,
			addr: local,
			cm:   newIDManager(),
		}

		conn, err := lis.Accept()
		require.Equal(t, acceptErr, err)
		require.Nil(t, conn)
	})
}

func TestListener_Close(t *testing.T) {
	l := logging.MustGetLogger("app2_listener")

	lisID := uint16(1)
	localPK, _ := cipher.GenerateKeyPair()
	local := network.Addr{
		Net:    network.TypeDMSG,
		PubKey: localPK,
		Port:   routing.Port(100),
	}

	t.Run("ok", func(t *testing.T) {
		var closeNoErr error
		closeErr := errors.New("close error")

		rpc := &MockRPCClient{}
		rpc.On("CloseListener", lisID).Return(closeNoErr)

		cm := newIDManager()

		connID1 := uint16(1)
		connID2 := uint16(2)
		connID3 := uint16(3)

		rpc.On("CloseConn", connID1).Return(closeNoErr)
		rpc.On("CloseConn", connID2).Return(closeErr)
		rpc.On("CloseConn", connID3).Return(closeNoErr)

		conn1 := &Conn{id: connID1, rpc: rpc}
		free1, err := cm.add(connID1, conn1)
		require.NoError(t, err)
		conn1.freeConn = free1

		conn2 := &Conn{id: connID2, rpc: rpc}
		free2, err := cm.add(connID2, conn2)
		require.NoError(t, err)
		conn2.freeConn = free2

		conn3 := &Conn{id: connID3, rpc: rpc}
		free3, err := cm.add(connID3, conn3)
		require.NoError(t, err)
		conn3.freeConn = free3

		lis := &Listener{
			log:     l,
			id:      lisID,
			rpc:     rpc,
			addr:    local,
			cm:      cm,
			freeLis: func() {},
		}

		err = lis.Close()
		require.NoError(t, err)

		_, ok := lis.cm.values[connID1]
		require.False(t, ok)

		_, ok = lis.cm.values[connID2]
		require.False(t, ok)

		_, ok = lis.cm.values[connID3]
		require.False(t, ok)
	})

	t.Run("close error", func(t *testing.T) {
		lisCloseErr := errors.New("close error")

		rpc := &MockRPCClient{}
		rpc.On("CloseListener", lisID).Return(lisCloseErr)

		lis := &Listener{
			log:  l,
			id:   lisID,
			rpc:  rpc,
			addr: local,
			cm:   newIDManager(),
		}

		err := lis.Close()
		require.Equal(t, err, lisCloseErr)
	})
}
