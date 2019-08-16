package network

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/skycoin/dmsg"
	"github.com/skycoin/dmsg/cipher"
)

func TestDisassembleAddr(t *testing.T) {
	pk, _ := cipher.GenerateKeyPair()
	port := uint16(2)
	addr := dmsg.Addr{
		PK: pk, Port: port,
	}
	gotPK, gotPort := disassembleAddr(addr)
	require.Equal(t, pk, gotPK)
	require.Equal(t, port, gotPort)
}
