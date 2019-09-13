package routing

import (
	"encoding/json"
	"testing"

	"github.com/skycoin/dmsg/cipher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMakePacket(t *testing.T) {
	packet := MakePacket(2, []byte("foo"))
	assert.Equal(
		t,
		[]byte{0x0, 0x3, 0x0, 0x0, 0x0, 0x2, 0x66, 0x6f, 0x6f},
		[]byte(packet),
	)

	assert.Equal(t, uint16(3), packet.Size())
	assert.Equal(t, RouteID(2), packet.RouteID())
	assert.Equal(t, []byte("foo"), packet.Payload())
}

func TestEncoding(t *testing.T) {
	pka, _ := cipher.GenerateKeyPair()
	pkb, _ := cipher.GenerateKeyPair()

	edges1 := PathEdges{pka, pkb}
	edges2 := PathEdges{pkb, pka}

	m := map[PathEdges]string{edges1: "a", edges2: "b"}

	b, err := json.Marshal(m)
	require.NoError(t, err)

	m2 := make(map[PathEdges]string)

	err = json.Unmarshal(b, &m2)
	require.NoError(t, err)
	assert.Equal(t, m, m2)
}
