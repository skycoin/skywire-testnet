package router

import (
	"net"
	"sort"
	"testing"

	"github.com/skycoin/skywire/pkg/app"

	"github.com/skycoin/skywire/internal/appnet"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/pkg/cipher"
)

func TestPortManager(t *testing.T) {
	pm := newPortManager(10)

	in, _ := net.Pipe()
	proto := appnet.NewProtocol(in)

	p1 := pm.Alloc(proto)
	assert.Equal(t, uint16(10), p1)

	require.Error(t, pm.Open(10, proto))
	require.NoError(t, pm.Open(8, proto))
	require.Error(t, pm.Open(8, proto))

	pk, _ := cipher.GenerateKeyPair()
	raddr := &app.LoopAddr{PubKey: pk, Port: 3}
	require.NoError(t, pm.SetLoop(8, raddr, &loop{}))
	require.Error(t, pm.SetLoop(7, raddr, &loop{}))

	assert.Equal(t, []*appnet.Protocol{proto}, pm.AppLinks())

	ports := pm.AppPorts(proto)
	sort.Slice(ports, func(i, j int) bool { return ports[i] < ports[j] })
	assert.Equal(t, []uint16{8, 10}, ports)

	b, err := pm.Get(10)
	require.NoError(t, err)
	require.NotNil(t, b)

	_, err = pm.Get(7)
	require.Error(t, err)

	l, err := pm.GetLoop(8, raddr)
	require.NoError(t, err)
	require.NotNil(t, l)

	_, err = pm.GetLoop(10, raddr)
	require.Error(t, err)

	_, err = pm.GetLoop(7, raddr)
	require.Error(t, err)

	require.Error(t, pm.RemoveLoop(7, raddr))
	require.NoError(t, pm.RemoveLoop(8, raddr))
	_, err = pm.GetLoop(8, raddr)
	require.Error(t, err)

	require.NoError(t, pm.SetLoop(8, raddr, &loop{}))

	assert.Empty(t, pm.Close(10))
	assert.Empty(t, pm.Close(7))
	assert.Equal(t, []app.LoopAddr{*raddr}, pm.Close(8))
}
