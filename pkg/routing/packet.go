package routing

import (
	"encoding/binary"
	"math"

	th "github.com/skycoin/skywire/internal/testhelpers"
)

// RouteID represents ID of a Route in a Packet.
type RouteID uint32

// Packet defines generic packet recognized by all skywire visors.
type Packet []byte

// MakePacket constructs a new Packet. If payload size is more than
// uint16, MakePacket will panic.
func MakePacket(id RouteID, payload []byte) Packet {
	log.Debug(th.Trace("ENTER"))
	defer log.Debug(th.Trace("ENTER"))

	if len(payload) > math.MaxUint16 {
		panic("packet size exceeded")
	}

	packet := make([]byte, 6)
	binary.BigEndian.PutUint16(packet, uint16(len(payload)))
	binary.BigEndian.PutUint32(packet[2:], uint32(id))
	return Packet(append(packet, payload...))
}

// Size returns Packet's payload size.
func (p Packet) Size() uint16 {
	return binary.BigEndian.Uint16(p)
}

// RouteID returns RouteID from a Packet.
func (p Packet) RouteID() RouteID {
	return RouteID(binary.BigEndian.Uint32(p[2:]))
}

// Payload returns payload from a Packet.
func (p Packet) Payload() []byte {
	return p[6:]
}
