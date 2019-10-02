package routing

import (
	"encoding/binary"
	"fmt"
	"math"
)

// Packet defines generic packet recognized by all skywire visors.
// The unit of communication for routing/router is called packets.
// Packet format:
//     | type (byte) | route ID (uint32) | payload size (uint16) | payload (~) |
//     | 1[0:1]      | 4[1:5]            | 2[5:7]                | [7:~]       |
type Packet []byte

// Packet sizes and offsets.
const (
	// PacketHeaderSize represents the base size of a packet.
	// All rules should have at-least this size.
	PacketHeaderSize        = 7
	PacketTypeOffset        = 0
	PacketRouteIDOffset     = 1
	PacketPayloadSizeOffset = 5
	PacketPayloadOffset     = PacketHeaderSize
)

// PacketType represents packet purpose.
type PacketType byte

func (t PacketType) String() string {
	switch t {
	case DataPacket:
		return "DataPacket"
	case ClosePacket:
		return "ClosePacket"
	case KeepAlivePacket:
		return "KeepAlivePacket"
	default:
		return fmt.Sprintf("Unknown(%d)", t)
	}
}

// Possible PacketType values:
// - DataPacket      - Payload is just the underlying data.
// - ClosePacket     - Payload is a type CloseCode byte.
// - KeepAlivePacket - Payload is empty.
const (
	DataPacket PacketType = iota
	ClosePacket
	KeepAlivePacket
)

// CloseCode represents close code for ClosePacket.
type CloseCode byte

func (cc CloseCode) String() string {
	switch cc {
	case CloseRequested:
		return "Closing requested by visor"
	default:
		return fmt.Sprintf("Unknown(%d)", byte(cc))
	}
}

const (
	CloseRequested CloseCode = iota
)

// RouteID represents ID of a Route in a Packet.
type RouteID uint32

// MakeDataPacket constructs a new DataPacket.
// If payload size is more than uint16, MakeDataPacket will panic.
func MakeDataPacket(id RouteID, payload []byte) Packet {
	if len(payload) > math.MaxUint16 {
		panic("packet size exceeded")
	}

	packet := make([]byte, PacketHeaderSize+len(payload))

	packet[PacketTypeOffset] = byte(DataPacket)
	binary.BigEndian.PutUint32(packet[PacketRouteIDOffset:], uint32(id))
	binary.BigEndian.PutUint16(packet[PacketPayloadSizeOffset:], uint16(len(payload)))
	copy(packet[PacketPayloadOffset:], payload)

	return packet
}

// MakeClosePacket constructs a new ClosePacket.
func MakeClosePacket(id RouteID, code CloseCode) Packet {
	packet := make([]byte, PacketHeaderSize+1)

	packet[PacketTypeOffset] = byte(ClosePacket)
	binary.BigEndian.PutUint32(packet[PacketRouteIDOffset:], uint32(id))
	binary.BigEndian.PutUint16(packet[PacketPayloadSizeOffset:], uint16(1))
	packet[PacketPayloadOffset] = byte(code)

	return packet
}

// MakeKeepAlivePacket constructs a new KeepAlivePacket.
func MakeKeepAlivePacket(id RouteID) Packet { // TODO(nkryuchkov): use it
	packet := make([]byte, PacketHeaderSize)

	packet[PacketTypeOffset] = byte(KeepAlivePacket)
	binary.BigEndian.PutUint32(packet[PacketRouteIDOffset:], uint32(id))
	binary.BigEndian.PutUint16(packet[PacketPayloadSizeOffset:], uint16(0))

	return packet
}

// Type returns Packet's type.
func (p Packet) Type() PacketType {
	return PacketType(p[PacketTypeOffset])
}

// Size returns Packet's payload size.
func (p Packet) Size() uint16 {
	return binary.BigEndian.Uint16(p[PacketPayloadSizeOffset:])
}

// RouteID returns RouteID from a Packet.
func (p Packet) RouteID() RouteID {
	return RouteID(binary.BigEndian.Uint32(p[PacketRouteIDOffset:]))
}

// Payload returns payload from a Packet.
func (p Packet) Payload() []byte {
	return p[PacketPayloadOffset:]
}
