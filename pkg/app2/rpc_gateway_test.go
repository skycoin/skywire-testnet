package app2

import (
	"context"
	"math"
	"net"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/dmsg"

	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/skywire/pkg/routing"

	"github.com/skycoin/skywire/pkg/app2/network"

	"github.com/skycoin/skycoin/src/util/logging"
)

func TestRPCGateway_Dial(t *testing.T) {
	l := logging.MustGetLogger("rpc_gateway")
	nType := network.TypeDMSG

	dialAddr := prepAddr(nType)

	t.Run("ok", func(t *testing.T) {
		network.ClearNetworkers()

		dialCtx := context.Background()
		dialConn := &dmsg.Transport{}
		var dialErr error

		n := &network.MockNetworker{}
		n.On("DialContext", dialCtx, dialAddr).Return(dialConn, dialErr)

		err := network.AddNetworker(nType, n)
		require.NoError(t, err)

		rpc := newRPCGateway(l)

		var connID uint16

		err = rpc.Dial(&dialAddr, &connID)
		require.NoError(t, err)
		require.Equal(t, connID, uint16(1))
	})

	t.Run("no more slots for a new conn", func(t *testing.T) {
		rpc := newRPCGateway(l)
		for i := uint16(0); i < math.MaxUint16; i++ {
			rpc.cm.values[i] = nil
		}
		rpc.cm.values[math.MaxUint16] = nil

		var connID uint16

		err := rpc.Dial(&dialAddr, &connID)
		require.Equal(t, err, errNoMoreAvailableValues)
	})

	t.Run("dial error", func(t *testing.T) {
		network.ClearNetworkers()

		dialCtx := context.Background()
		var dialConn net.Conn
		dialErr := errors.New("dial error")

		n := &network.MockNetworker{}
		n.On("DialContext", dialCtx, dialAddr).Return(dialConn, dialErr)

		err := network.AddNetworker(nType, n)
		require.NoError(t, err)

		rpc := newRPCGateway(l)

		var connID uint16

		err = rpc.Dial(&dialAddr, &connID)
		require.Equal(t, err, dialErr)
	})
}

func TestRPCGateway_Listen(t *testing.T) {
	l := logging.MustGetLogger("rpc_gateway")
	nType := network.TypeDMSG

	listenAddr := prepAddr(nType)

	t.Run("ok", func(t *testing.T) {
		network.ClearNetworkers()

		listenCtx := context.Background()
		listenLis := &dmsg.Listener{}
		var listenErr error

		n := &network.MockNetworker{}
		n.On("ListenContext", listenCtx, listenAddr).Return(listenLis, listenErr)

		err := network.AddNetworker(nType, n)
		require.Equal(t, err, listenErr)

		rpc := newRPCGateway(l)

		var lisID uint16

		err = rpc.Listen(&listenAddr, &lisID)
		require.NoError(t, err)
		require.Equal(t, lisID, uint16(1))
	})

	t.Run("no more slots for a new listener", func(t *testing.T) {
		rpc := newRPCGateway(l)
		for i := uint16(0); i < math.MaxUint16; i++ {
			rpc.lm.values[i] = nil
		}
		rpc.lm.values[math.MaxUint16] = nil

		var lisID uint16

		err := rpc.Listen(&listenAddr, &lisID)
		require.Equal(t, err, errNoMoreAvailableValues)
	})

	t.Run("listen error", func(t *testing.T) {
		network.ClearNetworkers()

		listenCtx := context.Background()
		var listenLis net.Listener
		listenErr := errors.New("listen error")

		n := &network.MockNetworker{}
		n.On("ListenContext", listenCtx, listenAddr).Return(listenLis, listenErr)

		err := network.AddNetworker(nType, n)
		require.NoError(t, err)

		rpc := newRPCGateway(l)

		var lisID uint16

		err = rpc.Listen(&listenAddr, &lisID)
		require.Equal(t, err, listenErr)
	})
}

func TestRPCGateway_Accept(t *testing.T) {
	l := logging.MustGetLogger("rpc_gateway")

	rpc := newRPCGateway(l)

	lisID, err := rpc.lm.nextKey()
	require.NoError(t, err)

	acceptConn := &dmsg.Transport{}
	var acceptErr error

	lis := &MockListener{}
	lis.On("Accept").Return(acceptConn, acceptErr)

	err = rpc.lm.set(*lisID, lis)
	require.NoError(t, err)

	var resp AcceptResp
	err = rpc.Accept(lisID, &resp)
	require.NoError(t, err)
}

func prepAddr(nType network.Type) network.Addr {
	pk, _ := cipher.GenerateKeyPair()
	port := routing.Port(100)

	return network.Addr{
		Net:    nType,
		PubKey: pk,
		Port:   port,
	}
}
