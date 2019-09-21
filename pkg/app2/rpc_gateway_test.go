package app2

import (
	"context"
	"testing"

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

	dialCtx := context.Background()
	dialAddrPK, _ := cipher.GenerateKeyPair()
	dialAddrPort := routing.Port(100)
	dialAddr := network.Addr{
		Net:    nType,
		PubKey: dialAddrPK,
		Port:   dialAddrPort,
	}
	dialConn := &dmsg.Transport{}
	var dialErr error

	n := &network.MockNetworker{}
	n.On("DialContext", dialCtx, dialAddr).Return(dialConn, dialErr)

	err := network.AddNetworker(nType, n)
	require.NoError(t, err)

	rpc := newRPCGateway(l)

	t.Run("ok", func(t *testing.T) {
		var connID uint16

		err := rpc.Dial(&dialAddr, &connID)
		require.NoError(t, err)
		require.Equal(t, connID, uint16(1))
	})
}
