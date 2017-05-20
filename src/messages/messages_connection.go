package messages

type ConnectionMessage struct {
	Sequence     uint32
	ConnectionId ConnectionId
	Order        uint32
	Total        uint32
	Payload      []byte
}

type ConnectionAck struct {
	Sequence     uint32
	ConnectionId ConnectionId
}
