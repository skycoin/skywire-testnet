package app2

import (
	"testing"

	"github.com/skycoin/skywire/pkg/app2/network"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/pkg/routing"

	"github.com/skycoin/dmsg/cipher"

	"github.com/skycoin/dmsg/netutil"
)

func TestClient_Dial(t *testing.T) {
	localPK, _ := cipher.GenerateKeyPair()
	pid := ProcID(1)

	remotePK, _ := cipher.GenerateKeyPair()
	remotePort := routing.Port(120)
	remote := network.Addr{
		PubKey: remotePK,
		Port:   remotePort,
	}

	t.Run("ok", func(t *testing.T) {
		dialConnID := uint16(1)
		var dialErr error

		rpc := &MockServerRPCClient{}
		rpc.On("Dial", remote).Return(dialConnID, dialErr)

		cl := NewClient(localPK, pid, rpc, netutil.NewPorter(netutil.PorterMinEphemeral))

		wantConn := &Conn{
			id:  dialConnID,
			rpc: rpc,
			local: network.Addr{
				PubKey: localPK,
			},
			remote: remote,
		}

		conn, err := cl.Dial(remote)
		appConn, ok := conn.(*Conn)
		require.True(t, ok)

		require.NoError(t, err)
		require.Equal(t, wantConn.id, appConn.id)
		require.Equal(t, wantConn.rpc, appConn.rpc)
		require.Equal(t, wantConn.local.PubKey, appConn.local.PubKey)
		require.Equal(t, wantConn.remote, appConn.remote)
		require.NotNil(t, appConn.freeLocalPort)
		portVal, ok := cl.porter.PortValue(uint16(appConn.local.Port))
		require.True(t, ok)
		require.Nil(t, portVal)
	})

	t.Run("dial error", func(t *testing.T) {
		dialErr := errors.New("dial error")

		rpc := &MockServerRPCClient{}
		rpc.On("Dial", remote).Return(uint16(0), dialErr)

		cl := NewClient(localPK, pid, rpc, netutil.NewPorter(netutil.PorterMinEphemeral))

		conn, err := cl.Dial(remote)
		require.Equal(t, dialErr, err)
		require.Nil(t, conn)
	})
}

func TestClient_Listen(t *testing.T) {
	localPK, _ := cipher.GenerateKeyPair()
	pid := ProcID(1)

	port := routing.Port(1)
	local := network.Addr{
		PubKey: localPK,
		Port:   port,
	}

	t.Run("ok", func(t *testing.T) {
		listenLisID := uint16(1)
		var listenErr error

		rpc := &MockServerRPCClient{}
		rpc.On("Listen", local).Return(listenLisID, listenErr)

		cl := NewClient(localPK, pid, rpc, netutil.NewPorter(netutil.PorterMinEphemeral))

		wantListener := &Listener{
			id:   listenLisID,
			rpc:  rpc,
			addr: local,
		}

		listener, err := cl.Listen(port)
		require.Nil(t, err)
		appListener, ok := listener.(*Listener)
		require.True(t, ok)
		require.Equal(t, wantListener.id, appListener.id)
		require.Equal(t, wantListener.rpc, appListener.rpc)
		require.Equal(t, wantListener.addr, appListener.addr)
		require.NotNil(t, appListener.freePort)
		portVal, ok := cl.porter.PortValue(uint16(port))
		require.True(t, ok)
		require.Nil(t, portVal)
	})

	t.Run("port is already bound", func(t *testing.T) {
		porter := netutil.NewPorter(netutil.PorterMinEphemeral)
		ok, _ := porter.Reserve(uint16(port), nil)
		require.True(t, ok)

		rpc := &MockServerRPCClient{}

		cl := NewClient(localPK, pid, rpc, porter)

		wantErr := ErrPortAlreadyBound

		listener, err := cl.Listen(port)
		require.Equal(t, wantErr, err)
		require.Nil(t, listener)
	})

	t.Run("listen error", func(t *testing.T) {
		listenErr := errors.New("listen error")

		rpc := &MockServerRPCClient{}
		rpc.On("Listen", local).Return(uint16(0), listenErr)

		cl := NewClient(localPK, pid, rpc, netutil.NewPorter(netutil.PorterMinEphemeral))

		listener, err := cl.Listen(port)
		require.Equal(t, listenErr, err)
		require.Nil(t, listener)
		_, ok := cl.porter.PortValue(uint16(port))
		require.False(t, ok)
	})
}
