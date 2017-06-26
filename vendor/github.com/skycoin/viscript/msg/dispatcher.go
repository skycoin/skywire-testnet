package msg

func SerializeAndDispatch(out chan []byte, msgType uint16, m interface{}) {
	// Serialize and send message interface to the out channel
	out <- Serialize(msgType, m)
}
