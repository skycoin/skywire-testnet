package network

import (
	"context"
	"net"
	"testing"

	"github.com/skycoin/skywire/pkg/routing"

	"github.com/skycoin/dmsg/cipher"

	"github.com/stretchr/testify/require"
)

func TestAddNetworker(t *testing.T) {
	clearNetworkers()

	nType := TypeDMSG
	var n Networker

	err := AddNetworker(nType, n)
	require.NoError(t, err)

	err = AddNetworker(nType, n)
	require.Equal(t, err, ErrNetworkerAlreadyExists)
}

func TestResolveNetworker(t *testing.T) {
	clearNetworkers()

	nType := TypeDMSG
	var n Networker

	n, err := ResolveNetworker(nType)
	require.Equal(t, err, ErrNoSuchNetworker)

	err = AddNetworker(nType, n)
	require.NoError(t, err)

	gotN, err := ResolveNetworker(nType)
	require.NoError(t, err)
	require.Equal(t, gotN, n)
}

func TestDial(t *testing.T) {
	addr := prepAddr()

	t.Run("no such networker", func(t *testing.T) {
		clearNetworkers()

		_, err := Dial(addr)
		require.Equal(t, err, ErrNoSuchNetworker)
	})

	t.Run("ok", func(t *testing.T) {
		clearNetworkers()

		dialCtx := context.Background()
		var (
			dialConn net.Conn
			dialErr  error
		)

		n := &MockNetworker{}
		n.On("DialContext", dialCtx, addr).Return(dialConn, dialErr)

		err := AddNetworker(addr.Net, n)
		require.NoError(t, err)

		conn, err := Dial(addr)
		require.NoError(t, err)
		require.Equal(t, conn, dialConn)
	})
}

func TestListen(t *testing.T) {
	addr := prepAddr()

	t.Run("no such networker", func(t *testing.T) {
		clearNetworkers()

		_, err := Listen(addr)
		require.Equal(t, err, ErrNoSuchNetworker)
	})

	t.Run("ok", func(t *testing.T) {
		clearNetworkers()

		listenCtx := context.Background()
		var (
			listenLis net.Listener
			listenErr error
		)

		n := &MockNetworker{}
		n.On("ListenContext", listenCtx, addr).Return(listenLis, listenErr)

		err := AddNetworker(addr.Net, n)
		require.NoError(t, err)

		lis, err := Listen(addr)
		require.NoError(t, err)
		require.Equal(t, lis, listenLis)
	})
}

func prepAddr() Addr {
	addrPK, _ := cipher.GenerateKeyPair()
	addrPort := routing.Port(100)

	return Addr{
		Net:    TypeDMSG,
		PubKey: addrPK,
		Port:   addrPort,
	}
}

func clearNetworkers() {
	networkersMx.Lock()
	defer networkersMx.Unlock()

	networkers = make(map[Type]Networker)
}
