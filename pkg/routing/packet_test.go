package routing

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMakeDataPacket(t *testing.T) {
	packet := MakeDataPacket(2, []byte("foo"))
	expected := []byte{0x0, 0x0, 0x0, 0x0, 0x2, 0x0, 0x3, 0x66, 0x6f, 0x6f}

	assert.Equal(t, expected, []byte(packet))
	assert.Equal(t, uint16(3), packet.Size())
	assert.Equal(t, RouteID(2), packet.RouteID())
	assert.Equal(t, []byte("foo"), packet.Payload())
}

func TestMakeClosePacket(t *testing.T) {
	packet := MakeClosePacket(3, CloseRequested)
	expected := []byte{0x1, 0x0, 0x0, 0x0, 0x3, 0x0, 0x1, 0x0}

	assert.Equal(t, expected, []byte(packet))
	assert.Equal(t, uint16(1), packet.Size())
	assert.Equal(t, RouteID(3), packet.RouteID())
	assert.Equal(t, []byte{0x0}, packet.Payload())
}

func TestMakeKeepAlivePacket(t *testing.T) {
	packet := MakeKeepAlivePacket(4)
	expected := []byte{0x2, 0x0, 0x0, 0x0, 0x4, 0x0, 0x0}

	assert.Equal(t, expected, []byte(packet))
	assert.Equal(t, uint16(0), packet.Size())
	assert.Equal(t, RouteID(4), packet.RouteID())
	assert.Equal(t, []byte{}, packet.Payload())
}
