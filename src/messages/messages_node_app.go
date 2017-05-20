package messages

type NodeAppMessage struct {
	Sequence uint32
	AppId    AppId
	Payload  []byte
}

type NodeAppResponse struct {
	Sequence uint32
}

type SendFromAppMessage struct {
	ConnectionId ConnectionId
	Payload      []byte
}

type RegisterAppMessage struct {
	AppType uint32
}

type AssignConnectionNAM struct {
	ConnectionId ConnectionId
}

type ConnectToAppMessage struct {
	Address string
	AppFrom AppId
	AppTo   AppId
}
