package messages

//Node Messages

// Transport -> Node Messages

//message received by node, from transport
//message comes in by a channel
type InRouteMessage struct {
	TransportId TransportId //who sent it
	RouteId     RouteId     //the incoming route
	Datagram    []byte      //length prefixed message
}

// Node -> Transport Messages

//message node, writes to the channel of the transport
type OutRouteMessage struct {
	RouteId  RouteId //the incoming route
	Datagram []byte  //length prefixed message
}

type CongestionPacket struct {
	TransportId TransportId //who sent it
	Congestion  bool        // true - increase throttle, false - decrease
}

// Transport -> Transport

//simulates one end of a transport, sending data to other end of the pair
type TransportDatagramTransfer struct {
	//put seq number for confirmation/ACK
	RouteId  RouteId
	Sequence uint32 //sequential sequence number of ACK
	Datagram []byte
}

type TransportDatagramACK struct {
	LowestSequence uint32 //ACK anything below this SEQ number
	Bitarray       uint32 //ACK packets at LowestSequence + Bit offset, if equal to 1
}
