package setup

import (
	"encoding/json"
	"fmt"
	"net"
	"testing"

	"github.com/skycoin/skywire/pkg/cipher"
	"github.com/skycoin/skywire/pkg/routing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ExampleNewSetupProtocol() {
	in, _ := net.Pipe()
	defer in.Close()

	sProto := NewSetupProtocol(in)
	fmt.Printf("Success: %v\n", sProto != nil)

	// Output: Success: true
}

func ExampleLoopData() {

	pk, _, _ := cipher.GenerateDeterministicKeyPair([]byte("loopData"))

	loopData := LoopData{
		RemotePK:     pk,
		RemotePort:   0,
		LocalPort:    0,
		RouteID:      routing.RouteID(0),
		NoiseMessage: []byte{},
	}
	fmt.Printf("%v\n", loopData)

	// Output: {02de45c828055aa84aa687d958caa9e5bd758a59c6bff530c71a6372940496f722 0 0 0 []}
}

func TestNewProtocol(t *testing.T) {
	connA, connB := net.Pipe()
	protoA := NewSetupProtocol(connA)
	protoB := NewSetupProtocol(connB)

	cases := []struct {
		Type PacketType
		Data string
	}{
		{PacketType(0), "this is a test!"},
		{PacketType(255), "this is another test!"},
		{PacketAddRules, string(PacketAddRules)},
		{PacketDeleteRules, string(PacketDeleteRules)},
		{PacketCreateLoop, string(PacketCreateLoop)},
		{PacketConfirmLoop, string(PacketConfirmLoop)},
		{PacketCloseLoop, string(PacketCloseLoop)},
		{PacketLoopClosed, string(PacketLoopClosed)},
		{RespFailure, string(RespFailure)},
		{RespSuccess, string(RespSuccess)},
	}

	for _, c := range cases {
		errChan := make(chan error, 1)
		go func() {
			errChan <- protoA.WritePacket(c.Type, []byte(c.Data))
		}()

		pt, data, err := protoB.ReadPacket()

		var decoded []byte
		require.NoError(t, json.Unmarshal(data, &decoded))

		assert.NoError(t, err)
		assert.Equal(t, c.Type, pt)
		assert.Equal(t, c.Data, string(decoded))

		assert.NoError(t, <-errChan)
	}
}
