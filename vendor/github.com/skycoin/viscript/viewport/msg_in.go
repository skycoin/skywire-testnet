package viewport

import (
	_ "strconv"

	"github.com/skycoin/viscript/app"
	"github.com/skycoin/viscript/config"
	"github.com/skycoin/viscript/msg"
	"github.com/skycoin/viscript/viewport/gl"
	t "github.com/skycoin/viscript/viewport/terminal"
)

func UnpackMessage(msgIn []byte) []byte {
	switch msg.GetType(msgIn) {

	case msg.TypeMousePos:
		var m msg.MessageMousePos
		msg.MustDeserialize(msgIn, &m)
		onMouseCursorPos(m)

	case msg.TypeMouseScroll:
		var m msg.MessageMouseScroll
		msg.MustDeserialize(msgIn, &m)
		onMouseScroll(m)

		if t.Terms.Focused != nil {
			t.Terms.Focused.RelayToTask(msgIn)
		}

	case msg.TypeMouseButton:
		var m msg.MessageMouseButton
		msg.MustDeserialize(msgIn, &m)
		onMouseButton(m)

	case msg.TypeChar:
		var m msg.MessageChar
		msg.MustDeserialize(msgIn, &m)
		onChar(m)

		if t.Terms.Focused != nil {
			t.Terms.Focused.RelayToTask(msgIn)
		}

	case msg.TypeKey:
		var m msg.MessageKey
		msg.MustDeserialize(msgIn, &m)
		onKey(m)

		if t.Terms.Focused != nil {
			t.Terms.Focused.RelayToTask(msgIn)
		}

	case msg.TypeFrameBufferSize:
		var m msg.MessageFrameBufferSize
		msg.MustDeserialize(msgIn, &m)
		onFrameBufferSize(m)

	default:
		app.At("viewport/msg_in", "************ UNHANDLED MESSAGE TYPE! ************")
	}

	return msgIn
}

func onFrameBufferSize(m msg.MessageFrameBufferSize) {
	if config.DebugPrintInputEvents() {
		print("msg.TypeFrameBufferSize")
		showUInt32("X", m.X)
		showUInt32("Y", m.Y)
		println()
	}

	gl.SetSize(int32(m.X), int32(m.Y))
}
