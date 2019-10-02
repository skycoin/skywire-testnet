package routing

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMakePacket(t *testing.T) {
	packet := MakeDataPacket(2, []byte("foo"))
	assert.Equal(
		t,
		[]byte{0x0, 0x3, 0x0, 0x0, 0x0, 0x2, 0x66, 0x6f, 0x6f},
		[]byte(packet),
	)

	assert.Equal(t, uint16(3), packet.Size())
	assert.Equal(t, RouteID(2), packet.RouteID())
	assert.Equal(t, []byte("foo"), packet.Payload())
}
