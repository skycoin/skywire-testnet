package process

import (
	"github.com/skycoin/viscript/app"
	"github.com/skycoin/viscript/msg"
)

func (st *State) UnpackMessage(msgType uint16, message []byte) []byte {
	switch msgType {

	case msg.TypeChar:
		var m msg.MessageChar
		msg.MustDeserialize(message, &m)
		st.onChar(m)

	case msg.TypeKey:
		var m msg.MessageKey
		msg.MustDeserialize(message, &m)
		st.onKey(m, message)

	case msg.TypeMouseScroll:
		var m msg.MessageMouseScroll
		msg.MustDeserialize(message, &m)
		st.onMouseScroll(m, message)

	case msg.TypeVisualInfo:
		var m msg.MessageVisualInfo
		msg.MustDeserialize(message, &m)
		st.makePageOfLog(m)

	case msg.TypeTerminalIds:
		var m msg.MessageTerminalIds
		msg.MustDeserialize(message, &m)
		st.onTerminalIds(m)

	default:
		app.At("hypervisor/process/terminal/msg_in", "UNKNOWN MESSAGE TYPE!!!")

	}

	return message
}
