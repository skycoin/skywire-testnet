package messages

import (
	"github.com/skycoin/skycoin/src/cipher"
)

type InControlMessage struct {
	ChannelId      ChannelId
	Sequence       uint32
	PayloadMessage []byte
}

type AddRouteCM struct {
	IncomingTransportId TransportId
	OutgoingTransportId TransportId
	IncomingRouteId     RouteId
	OutgoingRouteId     RouteId
}

type RemoveRouteCM struct {
	RouteId RouteId
}

// ==================== control messages ========================

type RegisterNodeCM struct {
	Hostname string
	Host     string
	Connect  bool
}

type RegisterNodeCMAck struct {
	Ok                bool
	NodeId            cipher.PubKey
	MaxBuffer         uint64
	MaxPacketSize     uint32
	TimeUnit          uint32
	SendInterval      uint32
	ConnectionTimeout uint32
}

type AssignPortCM struct {
	Port uint32
}

type TransportCreateCM struct {
	Id                TransportId
	PairId            TransportId
	PairedNodeId      cipher.PubKey
	MaxBuffer         uint64
	TimeUnit          uint32
	TransportTimeout  uint32
	SimulateDelay     bool
	MaxSimulatedDelay uint32
	RetransmitLimit   uint32
}

type TransportTickCM struct {
	Id TransportId
}

type TransportShutdownCM struct {
	Id TransportId
}

type OpenUDPCM struct {
	Id    TransportId
	PeerA Peer
	PeerB Peer
}

type CommonCMAck struct {
	Ok bool
}

type ConnectDirectlyCM struct {
	Sequence uint32
	From     cipher.PubKey
	To       string
}

type ConnectDirectlyCMAck struct {
	Sequence uint32
	Ok       bool
}

type ConnectWithRouteCM struct {
	Sequence  uint32
	AppIdFrom AppId
	AppIdTo   AppId
	From      cipher.PubKey
	To        string
}

type ConnectWithRouteCMAck struct {
	Sequence     uint32
	Ok           bool
	ConnectionId ConnectionId
}

type AssignConnectionCM struct {
	ConnectionId ConnectionId
	RouteId      RouteId
	AppId        AppId
}

type ConnectionOnCM struct {
	NodeId       cipher.PubKey
	ConnectionId ConnectionId
}

type RegisterAppCM struct {
	ServiceInfo ServiceInfo
	NodeId      cipher.PubKey
}

type RegisterAppCMAck struct {
	Ok    bool
	Error string
}

type ShutdownCM struct {
	NodeId cipher.PubKey
}
