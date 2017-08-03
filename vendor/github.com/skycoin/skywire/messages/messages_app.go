package messages

type AppMessage struct {
	Sequence uint32
	Payload  []byte
}

type AppResponse struct {
	Response []byte
	Err      error
}

type ProxyMessage struct {
	Data       []byte
	RemoteAddr string
	NeedClose  bool
}
